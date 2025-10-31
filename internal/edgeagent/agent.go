package edgeagent

import (
	"context"
	"net/http"

	"cloupeer.io/cloupeer/pkg/log"
)

// Agent is the core struct for the edge agent business logic.
type Agent struct {
	client *http.Client
}

// Run starts the main loop of the agent and handles graceful shutdown.
func (a *Agent) Run(ctx context.Context) error {
	log.Info("Starting cpeer-edge-agent")

	return nil
}
