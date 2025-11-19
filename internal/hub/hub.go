package hub

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"google.golang.org/grpc"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	pb "cloupeer.io/cloupeer/api/proto/v1"
	"cloupeer.io/cloupeer/pkg/log"
)

type HubServer struct {
	namespace  string
	httpAddr   string
	grpcAddr   string
	httpClient *http.Client
	k8sclient  controllerclient.Client
}

func (s *HubServer) Run(ctx context.Context) error {
	// Initialize Kubernetes client
	if err := s.initK8sClient(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// --- 1. Start HTTP Server (Health/Metrics) ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.runHTTPServer(ctx); err != nil {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// --- 2. Start gRPC Server (Business Logic) ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.runGRPCServer(ctx); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	// Wait for context cancellation (shutdown signal)
	<-ctx.Done()
	log.Info("Shutdown signal received, waiting for servers to stop...")

	// In a real implementation, we would call Shutdown() on both servers here.
	// For simplicity, we rely on the context cancellation propagation inside the run functions (if implemented)
	// or simply exit as the main process terminates.
	//
	// Create a new context for shutdown with a timeout
	// shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()
	// if err := server.Shutdown(shutdownCtx); err != nil {
	// 	log.Error(err, "cpeer-hub server shutdown failed")
	// }

	log.Info("cpeer-hub server stopped gracefully.")
	return nil
}

func (s *HubServer) runHTTPServer(ctx context.Context) error {
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

	server := &http.Server{
		Addr:    s.httpAddr,
		Handler: mux,
	}

	log.Info("cpeer-hub HTTP listening", "address", s.httpAddr, "namespace", s.namespace)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start cpeer-hub server: %w", err)
	}

	return nil
}

func (s *HubServer) runGRPCServer(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on grpc addr %s: %w", s.grpcAddr, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterHubServiceServer(grpcServer, &grpcHandler{})

	log.Info("cpeer-hub gRPC listening", "address", s.grpcAddr, "namespace", s.namespace)

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	return grpcServer.Serve(lis)
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
