package core

import (
	"context"

	"cloupeer.io/cloupeer/internal/cloudhub/core/model"
)

// VehicleRepository defines the interface for interacting with vehicle persistent data.
// In Cloupeer, this is implemented by the K8s Adapter.
type VehicleRepository interface {
	// Get retrieves a vehicle by its ID.
	Get(ctx context.Context, id string) (*model.Vehicle, error)

	// Create registers a new vehicle in the system.
	Create(ctx context.Context, vehicle *model.Vehicle) error

	UpdateStatus(ctx context.Context, update *model.Vehicle) error

	// BatchUpdateStatus updates the status fields (Online, LastSeen, Version) of a vehicle.
	// Note: Implementations should handle high-concurrency batching/buffering.
	BatchUpdateStatus(ctx context.Context, update *model.VehicleStatusUpdate) error
}

// CommandRepository defines the interface for interacting with command persistent data.
type CommandRepository interface {
	// UpdateStatus updates the lifecycle phase of a command (e.g., Received -> Running).
	UpdateStatus2(ctx context.Context, cmdID string, status model.CommandStatus, message string) error
}
