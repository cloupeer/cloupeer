package http

import (
	"context"
	"net/http"
	"time"

	"github.com/autopeer-io/autopeer/pkg/log"
	"github.com/autopeer-io/autopeer/pkg/options"
)

type Server struct {
	server  *http.Server
	options *options.HttpOptions
}

func NewServer(opts *options.HttpOptions) *Server {
	mux := http.NewServeMux()

	// Basic Liveness Probe
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Readiness Probe (Should check MQTT/K8s connection in production)
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	return &Server{
		server: &http.Server{
			Addr:    opts.Addr,
			Handler: mux,
		},
		options: opts,
	}
}

func (s *Server) Start(ctx context.Context) error {
	log.Info("Starting HTTP Server", "addr", s.server.Addr)

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	}
}
