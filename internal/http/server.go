package httpx

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// ProxyHandler implements a reverse proxy for middleware mode
type ProxyHandler struct {
	destination string
	client      *http.Client
}

// NewProxyHandler creates a new proxy handler for the given destination
func NewProxyHandler(destination string) *ProxyHandler {
	return &ProxyHandler{
		destination: destination,
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
	
	// Set the status code
	w.WriteHeader(resp.StatusCode)
	
	// Copy the response body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("proxy: failed to copy response body: %v", err)
	}
}

// MiddlewareRouter wraps a handler and forwards unmatched requests to a proxy
type MiddlewareRouter struct {
	trackingMux *http.ServeMux
	proxy       *ProxyHandler
}

// NewMiddlewareRouter creates a new middleware router that handles tracking routes
// and forwards everything else to the destination
func NewMiddlewareRouter(trackingMux *http.ServeMux, destination string) *MiddlewareRouter {
	return &MiddlewareRouter{
		trackingMux: trackingMux,
		proxy:       NewProxyHandler(destination),
	}
}

// ServeHTTP handles requests by first trying the tracking mux, then proxying on 404
func (m *MiddlewareRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if this is a tracking-related path
	if isTrackingPath(r.URL.Path) {
		m.trackingMux.ServeHTTP(w, r)
		return
	}
	
	// For non-tracking paths, proxy to the destination
	m.proxy.ServeHTTP(w, r)
}

// isTrackingPath determines if a path should be handled by the tracking server
func isTrackingPath(path string) bool {
	trackingPaths := []string{"/px.gif", "/collect", "/healthz", "/readyz", "/metrics"}
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
	
	// If middleware mode is enabled and we have a destination, wrap with proxy
	if e.Cfg.MiddlewareMode && e.Cfg.ForwardDestination != "" {
		// Validate the destination URL
		if _, err := url.Parse(e.Cfg.ForwardDestination); err != nil {
			log.Printf("WARNING: Invalid FORWARD_DESTINATION URL: %v. Middleware mode disabled.", err)
			return RequestLogger(cors(mux))
		}
		
		log.Printf("Middleware mode enabled, forwarding to: %s", e.Cfg.ForwardDestination)
		router := NewMiddlewareRouter(mux, e.Cfg.ForwardDestination)
		return RequestLogger(cors(router))
	}
	
	if e.Cfg.MiddlewareMode && e.Cfg.ForwardDestination == "" {
		log.Printf("WARNING: MIDDLEWARE_MODE=true but FORWARD_DESTINATION is empty. Middleware mode disabled.")
	}
	
	// Apply CORS and request logging middleware
	return RequestLogger(cors(mux))
}
