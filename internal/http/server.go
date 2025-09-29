package httpx

import "net/http"

func NewMux(e Env) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", e.Healthz)
	mux.HandleFunc("/readyz", e.Readyz)
	mux.HandleFunc("/px.gif", e.Pixel)
	mux.HandleFunc("/collect", e.Collect)
	
	// Apply CORS and request logging middleware
	return RequestLogger(cors(mux))
}
