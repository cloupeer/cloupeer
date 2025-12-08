package server

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"cloupeer.io/cloupeer/internal/cloudhub/core/service"
	"cloupeer.io/cloupeer/internal/cloudhub/server/grpc"
	"cloupeer.io/cloupeer/internal/cloudhub/server/http"
	"cloupeer.io/cloupeer/internal/cloudhub/server/mqtt"
	"cloupeer.io/cloupeer/pkg/log"
)

// Server defines the common interface for all sub-servers (grpc, mqtt, http).
type Server interface {
	Start(ctx context.Context) error
}

// Manager manages the lifecycle of all protocol servers.
type Manager struct {
	servers []Server
}

// NewManager creates a new server manager and initializes all sub-servers.
func NewManager(cfg *Config, svc *service.Service) (*Manager, error) {
	var servers []Server

	// 1. Initialize MQTT Server (The Data Plane Gateway)
	mqttSrv, err := mqtt.NewServer(cfg.MqttOptions, svc)
	if err != nil {
		return nil, fmt.Errorf("failed to init mqtt server: %w", err)
	}
	servers = append(servers, mqttSrv)

	// 2. Initialize gRPC Server (The Control Plane Gateway)
	grpcSrv, err := grpc.NewServer(cfg.GrpcOptions, svc)
	if err != nil {
		return nil, fmt.Errorf("failed to init grpc server: %w", err)
	}
	servers = append(servers, grpcSrv)

	// 3. Initialize HTTP Server (Health & Metrics)
	httpSrv := http.NewServer(cfg.HttpOptions)
	servers = append(servers, httpSrv)

	return &Manager{
		servers: servers,
	}, nil
}

// Start launches all servers in parallel and waits for termination.
func (m *Manager) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, s := range m.servers {
		srv := s // capture loop variable
		g.Go(func() error {
			return srv.Start(ctx)
		})
	}

	log.Info("All servers starting...")
	return g.Wait()
}
