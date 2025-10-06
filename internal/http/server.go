package httpx

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shortontech/gotrack/internal/assets"
)

// ProxyHandler implements a reverse proxy for middleware mode
type ProxyHandler struct {
	destination     string
	client          *http.Client
	autoInjectPixel bool
	hmacAuth        *HMACAuth
}

// NewProxyHandler creates a new proxy handler for the given destination
func NewProxyHandler(destination string, autoInjectPixel bool, hmacAuth *HMACAuth) *ProxyHandler {
	return &ProxyHandler{
		destination:     destination,
		autoInjectPixel: autoInjectPixel,
		hmacAuth:        hmacAuth,
		client: &http.Client{
			Timeout: 30 * time.Second, // 30 second timeout for proxied requests
		},
	}
}

// ServeHTTP proxies requests to the destination server
func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Build the target URL
	targetURL, err := url.Parse(p.destination)
	if err != nil {
		log.Printf("proxy: invalid destination URL: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Create the target URL with the original path and query
	targetURL.Path = r.URL.Path
	targetURL.RawQuery = r.URL.RawQuery

	// Create a context with timeout for the proxy request
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	// Create a new request to the destination
	proxyReq, err := http.NewRequestWithContext(ctx, r.Method, targetURL.String(), r.Body)
	if err != nil {
		log.Printf("proxy: failed to create request: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Copy headers from the original request
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Set the Host header to the destination host
	proxyReq.Host = targetURL.Host

	// Forward the request
	resp, err := p.client.Do(proxyReq)
	if err != nil {
		log.Printf("proxy: request to %s failed: %v", targetURL.String(), err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Check if we should inject pixel for HTML content
	if p.autoInjectPixel && isHTMLContent(resp.Header.Get("Content-Type")) {
		// Check if response is gzip encoded
		isGzipped := strings.Contains(strings.ToLower(resp.Header.Get("Content-Encoding")), "gzip")
		
		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("proxy: failed to read response body for pixel injection: %v", err)
			w.WriteHeader(resp.StatusCode)
			return
		}

		// Decompress if gzipped
		var htmlBody []byte
		if isGzipped {
			gzReader, err := gzip.NewReader(bytes.NewReader(body))
			if err != nil {
				log.Printf("proxy: failed to create gzip reader: %v", err)
				// Fall back to serving as-is
				w.WriteHeader(resp.StatusCode)
				_, _ = w.Write(body)
				return
			}
			defer gzReader.Close()
			
			htmlBody, err = io.ReadAll(gzReader)
			if err != nil {
				log.Printf("proxy: failed to decompress gzipped body: %v", err)
				// Fall back to serving as-is
				w.WriteHeader(resp.StatusCode)
				_, _ = w.Write(body)
				return
			}
		} else {
			htmlBody = body
		}

		// Inject pixel into decompressed HTML
		modifiedBody := injectPixel(htmlBody, r, p.hmacAuth)

		// Re-compress if original was gzipped
		var finalBody []byte
		if isGzipped {
			var buf bytes.Buffer
			gzWriter := gzip.NewWriter(&buf)
			_, err = gzWriter.Write(modifiedBody)
			if err != nil {
				log.Printf("proxy: failed to write gzipped body: %v", err)
				w.WriteHeader(resp.StatusCode)
				return
			}
			err = gzWriter.Close()
			if err != nil {
				log.Printf("proxy: failed to close gzip writer: %v", err)
				w.WriteHeader(resp.StatusCode)
				return
			}
			finalBody = buf.Bytes()
		} else {
			finalBody = modifiedBody
		}

		// Update Content-Length header
		w.Header().Set("Content-Length", strconv.Itoa(len(finalBody)))

		w.WriteHeader(resp.StatusCode)
		_, err = w.Write(finalBody)
		if err != nil {
			log.Printf("proxy: failed to write modified response body: %v", err)
		}
	} else {
		// For non-HTML content, copy response as-is
		w.WriteHeader(resp.StatusCode)
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			log.Printf("proxy: failed to copy response body: %v", err)
		}
	}
}

// MiddlewareRouter wraps a handler and forwards unmatched requests to a proxy
type MiddlewareRouter struct {
	trackingMux    *http.ServeMux
	proxy          *ProxyHandler
	collectHandler http.HandlerFunc
}

// isHTMLContent checks if the content type indicates HTML content (case-insensitive)
func isHTMLContent(contentType string) bool {
	if contentType == "" {
		return false
	}

	// Convert to lowercase for case-insensitive comparison
	ct := strings.ToLower(strings.TrimSpace(contentType))

	// Check for HTML content types
	return strings.Contains(ct, "text/html") ||
		strings.Contains(ct, "application/xhtml+xml") ||
		strings.Contains(ct, "application/xhtml")
}

// injectPixel adds a tracking pixel to HTML content before the closing </body> tag
// It inlines the entire JavaScript library to avoid ad-blocker detection
func injectPixel(body []byte, r *http.Request, hmacAuth *HMACAuth) []byte {
	// Convert to string for easier manipulation
	html := string(body)

	// Create the pixel tracking image tag with full URL including query parameters
	fullURL := r.URL.Path
	if r.URL.RawQuery != "" {
		fullURL = r.URL.Path + "?" + r.URL.RawQuery
	}
	pixelURL := "/px.gif?e=pageview&auto=1&url=" + url.QueryEscape(fullURL)

	// Build injected content with INLINED tracking library and pixel
	// By inlining the entire script, we avoid ad-blocker detection on script src URLs
	var injectedContent string
	if hmacAuth != nil {
		// Include HMAC script (keep as src since it needs server state), inline tracking library, and pixel
		// nosemgrep: go.lang.security.injection.raw-html-format.raw-html-format
		injectedContent = fmt.Sprintf(`<script src="/hmac.js"></script>
<script>%s</script>
<img src="%s" width="1" height="1" style="display:none" alt="">`, 
			string(assets.PixelUMDJS), 
			template.HTMLEscapeString(pixelURL)) // nosemgrep: go.lang.security.injection.raw-html-format.raw-html-format
	} else {
		// Inline tracking library and pixel without HMAC
		// nosemgrep: go.lang.security.injection.raw-html-format.raw-html-format
		injectedContent = fmt.Sprintf(`<script>%s</script>
<img src="%s" width="1" height="1" style="display:none" alt="">`, 
			string(assets.PixelUMDJS), 
			template.HTMLEscapeString(pixelURL)) // nosemgrep: go.lang.security.injection.raw-html-format.raw-html-format
	}

	// Try to inject before </body> tag (case-insensitive)
	bodyCloseRegex := regexp.MustCompile(`(?i)</body>`)
	if bodyCloseRegex.MatchString(html) {
		// Inject before </body>
		modified := bodyCloseRegex.ReplaceAllString(html, injectedContent+"\n</body>")
		return []byte(modified)
	}

	// If no </body> tag found, try to inject before </html> (case-insensitive)
	htmlCloseRegex := regexp.MustCompile(`(?i)</html>`)
	if htmlCloseRegex.MatchString(html) {
		// Inject before </html>
		modified := htmlCloseRegex.ReplaceAllString(html, injectedContent+"\n</html>")
		return []byte(modified)
	}

	// If neither tag found, append to the end
	return bytes.Join([][]byte{body, []byte(injectedContent)}, []byte("\n"))
}

// NewMiddlewareRouter creates a new middleware router that handles tracking routes
// and forwards everything else to the destination
func NewMiddlewareRouter(trackingMux *http.ServeMux, destination string, autoInjectPixel bool, hmacAuth *HMACAuth, collectHandler http.HandlerFunc) *MiddlewareRouter {
	return &MiddlewareRouter{
		trackingMux:    trackingMux,
		proxy:          NewProxyHandler(destination, autoInjectPixel, hmacAuth),
		collectHandler: collectHandler,
	}
}

// ServeHTTP handles requests by first trying the tracking mux, then proxying on 404
func (m *MiddlewareRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if this is a tracking-related path
	if isTrackingPath(r.URL.Path) {
		m.trackingMux.ServeHTTP(w, r)
		return
	}

	// Check if this is a collection request (POST with HMAC header to any path)
	// This allows tracking data to be sent to any URL, making it harder to block
	if r.Method == http.MethodPost && r.Header.Get("X-GoTrack-HMAC") != "" {
		// Route to collect handler directly
		m.collectHandler(w, r)
		return
	}

	// For non-tracking paths, proxy to the destination
	m.proxy.ServeHTTP(w, r)
}

// isTrackingPath determines if a path should be handled by the tracking server
func isTrackingPath(path string) bool {
	trackingPaths := []string{
		"/px.gif",
		"/collect",
		"/healthz",
		"/readyz",
		"/metrics",
		"/hmac.js",
		"/hmac/public-key",
		"/pixel.js",
		"/pixel.umd.js",
		"/pixel.esm.js",
	}
	for _, trackingPath := range trackingPaths {
		if path == trackingPath {
			return true
		}
	}
	return false
}

func NewMux(e Env) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", e.Healthz)
	mux.HandleFunc("/readyz", e.Readyz)
	mux.HandleFunc("/px.gif", e.Pixel)
	mux.HandleFunc("/collect", e.Collect)

	// HMAC authentication endpoints
	mux.HandleFunc("/hmac.js", e.HMACScript)
	mux.HandleFunc("/hmac/public-key", e.HMACPublicKey)

	// Pixel JS distribution endpoints
	mux.HandleFunc("/pixel.js", e.ServePixelJS)
	mux.HandleFunc("/pixel.umd.js", e.ServePixelJS)
	mux.HandleFunc("/pixel.esm.js", e.ServePixelJS)

	// If middleware mode is enabled and we have a destination, wrap with proxy
	if e.Cfg.MiddlewareMode && e.Cfg.ForwardDestination != "" {
		// Validate the destination URL
		if _, err := url.Parse(e.Cfg.ForwardDestination); err != nil {
			log.Printf("WARNING: Invalid FORWARD_DESTINATION URL: %v. Middleware mode disabled.", err)
			return RequestLogger(cors(mux))
		}

		log.Printf("Middleware mode enabled, forwarding to: %s", e.Cfg.ForwardDestination)
		if e.Cfg.AutoInjectPixel {
			log.Printf("Auto pixel injection enabled for HTML content")
		}
		router := NewMiddlewareRouter(mux, e.Cfg.ForwardDestination, e.Cfg.AutoInjectPixel, e.HMACAuth, e.Collect)
		return RequestLogger(MetricsMiddleware(e.Metrics)(cors(router)))
	}

	if e.Cfg.MiddlewareMode && e.Cfg.ForwardDestination == "" {
		log.Printf("WARNING: MIDDLEWARE_MODE=true but FORWARD_DESTINATION is empty. Middleware mode disabled.")
	}

	// Apply CORS, metrics, and request logging middleware
	return RequestLogger(MetricsMiddleware(e.Metrics)(cors(mux)))
}
