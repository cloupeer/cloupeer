package hub

import "net/http"

func (cfg *Config) NewHttpServer() *http.Server {
	mux := http.NewServeMux()
	// Add healthz handler
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	// Add readyz handler
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	return &http.Server{
		Addr:    cfg.HttpOptions.Addr,
		Handler: mux,
	}
}
