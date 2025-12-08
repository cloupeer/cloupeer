package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloupeer.io/cloupeer/internal/cloudhub/core/model"
	"cloupeer.io/cloupeer/internal/pkg/util"
)

// RegisterVehicle handles the registration of a vehicle when it connects.
// Flow:
// 1. Check if vehicle exists in K8s (via Repo).
// 2. If not found, create a new Vehicle CRD.
// 3. If found, assume it's a reconnection (logic can be extended to update firmware version here).
func (s *Service) RegisterVehicle(ctx context.Context, v *model.Vehicle) error {
	// Default to Online=true upon registration
	v.Online = true
	v.LastSeen = time.Now()

	// Check existence
	existing, err := s.vehicleRepo.Get(ctx, v.ID)
	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			// Create new vehicle
			if err = s.vehicleRepo.Create(ctx, v); err != nil {
				return fmt.Errorf("failed to create vehicle: %w", err)
			}
		}
		return err
	}

	if existing != nil {
		// Vehicle exists.
		// Optional: We could update the Description or FirmwareVersion if changed.
		// For high concurrency, we might skip heavy updates here unless necessary.
		return nil
	}

	return nil
}

// UpdateOnlineStatus processes heartbeat or connection state changes (Online/Offline).
// This is a high-frequency operation.
func (s *Service) UpdateOnlineStatus(ctx context.Context, vehicleID string, online bool) error {
	update := &model.VehicleStatusUpdate{
		ID:       vehicleID,
		Online:   online,
		LastSeen: time.Now(),
	}

	// This calls the Repository's optimized (buffered) update method.
	if err := s.vehicleRepo.BatchUpdateStatus(ctx, update); err != nil {
		return fmt.Errorf("failed to update online status: %w", err)
	}

	return nil
}
