package vehicleagent

import (
	"context"
	"fmt"
	"time"

	pb "cloupeer.io/cloupeer/api/proto/v1"
	"cloupeer.io/cloupeer/internal/vehicleagent/core"
	"cloupeer.io/cloupeer/internal/vehicleagent/hub"
	"cloupeer.io/cloupeer/pkg/log"
)

type Agent struct {
	vehicleID string
	hub       *hub.Hub
}

func NewAgent(vid string, hub *hub.Hub) *Agent {
	return &Agent{
		vehicleID: vid,
		hub:       hub,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	log.Info("Starting cpeer-edge-agent", "vehicleID", a.vehicleID)

	for _, m := range core.GetModules() {
		if err := m.Setup(ctx, a.hub); err != nil {
			return err
		}

		for event, handler := range m.Routes() {
			if err := a.hub.Register(event, handler); err != nil {
				return fmt.Errorf("module %s register event %s failed: %w", m.Name(), event, err)
			}
		}
	}

	if err := a.hub.Start(ctx); err != nil {
		return err
	}
	defer a.hub.Stop()

	// Send Registration/Online Message
	go a.registerIdentity(ctx)

	<-ctx.Done()
	log.Info("Agent shutting down...")

	return nil
}

// registerIdentity sends the initial registration packet to the Hub.
func (a *Agent) registerIdentity(ctx context.Context) {
	// Simulation: Get current version from local system
	// In production, this comes from a version file or API.
	currentVersion := "v1.0.0"

	req := &pb.RegisterVehicleRequest{
		VehicleId:       a.vehicleID,
		FirmwareVersion: currentVersion,
		Description:     "Edge Agent Auto-Registration",
		Timestamp:       time.Now().Unix(),
	}

	// Retry logic could be added here, but for now we send once (QoS 1 handles delivery)
	if err := a.hub.SendProto(ctx, core.EventRegister, req); err != nil {
		log.Error(err, "Failed to send registration request")
		return
	}

	log.Info("Sent registration request", "version", currentVersion)
}
