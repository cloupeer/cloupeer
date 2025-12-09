package server

import (
	"context"

	"golang.org/x/sync/errgroup"

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
func NewManager(servers ...Server) *Manager {
	return &Manager{
		servers: servers,
	}
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
