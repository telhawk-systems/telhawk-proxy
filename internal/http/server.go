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

// ProxyHandler implements a reverse proxy
type ProxyHandler struct {
	destination string
	client      *http.Client
	hmacAuth    *HMACAuth
}

// NewProxyHandler creates a new proxy handler for the given destination
func NewProxyHandler(destination string, hmacAuth *HMACAuth) *ProxyHandler {
	return &ProxyHandler{
		destination: destination,
		hmacAuth:    hmacAuth,
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

	// Create and execute proxy request
	resp, err := p.executeProxyRequest(w, r, targetURL)
	if err != nil {
		return // Error already handled in executeProxyRequest
	}
	defer resp.Body.Close()

	// Copy response headers
	copyHeaders(w.Header(), resp.Header)

	// Process and write response
	if isHTMLContent(resp.Header.Get("Content-Type")) {
		p.handleHTMLResponse(w, r, resp)
	} else {
		p.handleNonHTMLResponse(w, resp)
	}
}

// executeProxyRequest creates and executes the proxy request
func (p *ProxyHandler) executeProxyRequest(w http.ResponseWriter, r *http.Request, targetURL *url.URL) (*http.Response, error) {
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
		return nil, err
	}

	// Copy headers from the original request
	copyHeaders(proxyReq.Header, r.Header)

	// Set the Host header to the destination host
	proxyReq.Host = targetURL.Host

	// Forward the request
	resp, err := p.client.Do(proxyReq)
	if err != nil {
		log.Printf("proxy: request to %s failed: %v", targetURL.String(), err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return nil, err
	}

	return resp, nil
}

// handleHTMLResponse processes HTML responses with pixel injection
func (p *ProxyHandler) handleHTMLResponse(w http.ResponseWriter, r *http.Request, resp *http.Response) {
	isGzipped := strings.Contains(strings.ToLower(resp.Header.Get("Content-Encoding")), "gzip")

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("proxy: failed to read response body for pixel injection: %v", err)
		w.WriteHeader(resp.StatusCode)
		return
	}

	// Decompress if needed
	htmlBody, err := p.decompressIfNeeded(body, isGzipped)
	if err != nil {
		// Fall back to serving as-is
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
		return
	}

	// Inject pixel into HTML
	modifiedBody := injectPixel(htmlBody, r, p.hmacAuth)

	// Re-compress if needed
	finalBody, err := p.compressIfNeeded(modifiedBody, isGzipped)
	if err != nil {
		w.WriteHeader(resp.StatusCode)
		return
	}

	// Update Content-Length header
	w.Header().Set("Content-Length", strconv.Itoa(len(finalBody)))

	w.WriteHeader(resp.StatusCode)
	_, err = w.Write(finalBody)
	if err != nil {
		log.Printf("proxy: failed to write modified response body: %v", err)
	}
}

// handleNonHTMLResponse copies non-HTML responses as-is
func (p *ProxyHandler) handleNonHTMLResponse(w http.ResponseWriter, resp *http.Response) {
	w.WriteHeader(resp.StatusCode)
	_, err := io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("proxy: failed to copy response body: %v", err)
	}
}

// decompressIfNeeded decompresses gzipped content if needed
func (p *ProxyHandler) decompressIfNeeded(body []byte, isGzipped bool) ([]byte, error) {
	if !isGzipped {
		return body, nil
	}

	gzReader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		log.Printf("proxy: failed to create gzip reader: %v", err)
		return nil, err
	}
	defer gzReader.Close()

	htmlBody, err := io.ReadAll(gzReader)
	if err != nil {
		log.Printf("proxy: failed to decompress gzipped body: %v", err)
		return nil, err
	}

	return htmlBody, nil
}

// compressIfNeeded compresses content if needed
func (p *ProxyHandler) compressIfNeeded(body []byte, shouldCompress bool) ([]byte, error) {
	if !shouldCompress {
		return body, nil
	}

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	_, err := gzWriter.Write(body)
	if err != nil {
		log.Printf("proxy: failed to write gzipped body: %v", err)
		return nil, err
	}
	err = gzWriter.Close()
	if err != nil {
		log.Printf("proxy: failed to close gzip writer: %v", err)
		return nil, err
	}

	return buf.Bytes(), nil
}

// copyHeaders copies HTTP headers from source to destination
func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
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
func NewMiddlewareRouter(trackingMux *http.ServeMux, destination string, hmacAuth *HMACAuth, collectHandler http.HandlerFunc) *MiddlewareRouter {
	return &MiddlewareRouter{
		trackingMux:    trackingMux,
		proxy:          NewProxyHandler(destination, hmacAuth),
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

	// Check if this is potentially a collection request (POST with HMAC header)
	// We route to collect handler if HMAC header exists
	// The collect handler will validate and either accept or reject
	if r.Method == http.MethodPost && r.Header.Get("X-GoTrack-HMAC") != "" {
		// Route to collect handler - it will validate HMAC
		// If HMAC is invalid, collect handler returns 401 (not proxied)
		m.collectHandler(w, r)
		return
	}

	// No HMAC header = normal request, proxy to destination
	m.proxy.ServeHTTP(w, r)
}

// statusRecorder captures the status code (removed, not needed)

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

	//  wrap with proxy
	if e.Cfg.ForwardDestination != "" {
		// Validate the destination URL
		if _, err := url.Parse(e.Cfg.ForwardDestination); err != nil {
			log.Fatalf("WARNING: Invalid FORWARD_DESTINATION URL: %v.", err)
			return RequestLogger(cors(mux))
		}

		router := NewMiddlewareRouter(mux, e.Cfg.ForwardDestination, e.HMACAuth, e.Collect)
		return RequestLogger(MetricsMiddleware(e.Metrics)(cors(router)))
	}

	// Apply CORS, metrics, and request logging middleware
	return RequestLogger(MetricsMiddleware(e.Metrics)(cors(mux)))
}
