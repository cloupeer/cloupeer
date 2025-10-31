package hub

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"cloupeer.io/cloupeer/pkg/log"
)

type HubServer struct {
	namespace string
	addr      string
	k8sclient controllerclient.Client
}

func (s *HubServer) Run(ctx context.Context) error {
	// Initialize Kubernetes client
	if err := s.initK8sClient(); err != nil {
		return err
	}

	// Setup HTTP router
	mux := http.NewServeMux()

	// Setup HTTP server with graceful shutdown
	server := &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	// Start a goroutine to listen for shutdown signals
	go func() {
		<-ctx.Done() // Block until context is canceled
		log.Info("Shutting down cpeer-hub server...")
		// Create a new context for shutdown with a timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error(err, "cpeer-hub server shutdown failed")
		}
	}()

	// Start the server
	log.Info("cpeer-hub is listening on", "address", s.addr, "namespace", s.namespace)
	// ListenAndServe will block until the server is shut down.
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start cpeer-hub server: %w", err)
	}

	log.Info("cpeer-hub server stopped gracefully.")
	return nil
}

func (s *HubServer) initK8sClient() error {
	k8sconfig, err := config.GetConfig()
	if err != nil {
		log.Error(err, "failed to get kubernetes config")
		return err
	}

	// Create a new scheme and add all our API types and standard types
	cloupeerscheme := runtime.NewScheme()
	_ = scheme.AddToScheme(cloupeerscheme) // Add standard schemes like v1.Pod, etc.

	k8sclient, err := controllerclient.New(k8sconfig, controllerclient.Options{Scheme: cloupeerscheme})
	if err != nil {
		log.Error(err, "failed to create kubernetes client")
		return err
	}
	s.k8sclient = k8sclient
	return nil
}
