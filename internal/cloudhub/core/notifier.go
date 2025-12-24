package core

import (
	"context"

	"github.com/autopeer-io/autopeer/internal/cloudhub/core/model"
)

// CommandNotifier defines the interface for sending asynchronous commands to vehicles.
// In Autopeer, this is implemented by the MQTT Outbound Adapter.
type CommandNotifier interface {
	// Notify sends a command payload to the target vehicle.
	Notify(ctx context.Context, cmd *model.Command) error
}
