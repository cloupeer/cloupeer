package service

import (
	"cloupeer.io/cloupeer/internal/cloudhub/core"
)

// Service implements the core business logic (Use Cases) for CloudHub.
// It orchestrates calls between the Model entities and the Adapters (Ports).
type Service struct {
	vehicleRepo core.VehicleRepository
	cmdRepo     core.CommandRepository
	notifier    core.CommandNotifier
	storage     core.Storage
}

// New creates a new instance of the CloudHub core service.
// Dependency Injection happens here.
func New(
	vehicleRepo core.VehicleRepository,
	cmdRepo core.CommandRepository,
	notifier core.CommandNotifier,
	storage core.Storage,
) *Service {
	return &Service{
		vehicleRepo: vehicleRepo,
		cmdRepo:     cmdRepo,
		notifier:    notifier,
		storage:     storage,
	}
}
