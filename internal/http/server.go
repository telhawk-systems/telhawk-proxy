package httpx

import "net/http"

func NewMux(e Env) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", e.Healthz)
	mux.HandleFunc("/readyz", e.Readyz)
	mux.HandleFunc("/px.gif", e.Pixel)
	mux.HandleFunc("/collect", e.Collect)
	return mux
}
