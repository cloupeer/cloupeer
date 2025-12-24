package vehiclecommand

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iovv1alpha2 "github.com/autopeer-io/autopeer/pkg/apis/iov/v1alpha2"
)

// GarbageCollector handles the periodic cleanup of stale VehicleCommand resources.
// It implements the manager.Runnable interface to run in the background.
type GarbageCollector struct {
	Client            client.Client
	Log               logr.Logger
	RetentionDuration time.Duration // e.g., 30 days
	CleanupInterval   time.Duration // e.g., 1 hour
}

// Start begins the garbage collection loop.
// It blocks until the context is cancelled.
func (gc *GarbageCollector) Start(ctx context.Context) error {
	gc.Log.Info("Starting VehicleCommand Garbage Collector",
		"retention", gc.RetentionDuration,
		"interval", gc.CleanupInterval)

	ticker := time.NewTicker(gc.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			gc.cleanup(ctx)
		case <-ctx.Done():
			gc.Log.Info("Stopping VehicleCommand Garbage Collector")
			return nil
		}
	}
}

// cleanup performs the actual list and delete logic.
func (gc *GarbageCollector) cleanup(ctx context.Context) {
	gc.Log.V(1).Info("Running scheduled cleanup for VehicleCommands")

	// List all VehicleCommands
	// Note: In a production environment with millions of records, consider using
	// Pagination (Continue/Limit) or Listing with specific labels to reduce memory footprint.
	// For now, listing from the specialized controller-runtime cache is efficient enough.
	cmdList := &iovv1alpha2.VehicleCommandList{}
	if err := gc.Client.List(ctx, cmdList); err != nil {
		gc.Log.Error(err, "Failed to list VehicleCommands for GC")
		return
	}

	threshold := time.Now().Add(-gc.RetentionDuration)
	deletedCount := 0

	for _, cmd := range cmdList.Items {
		// 1. Skip if the command is not in a terminal state.
		// We assume strictly that only finished commands should be deleted.
		if !isTerminalState(&cmd) {
			continue
		}

		// 2. Check the timestamp.
		// Using CompletionTime is better, but if nil, fallback to CreationTimestamp.
		// Here we assume Status.LastUpdateTime is maintained, or use CreationTimestamp as a fallback.
		checkTime := cmd.CreationTimestamp.Time
		// If you have a specific Status.FinishedAt field, use it here:
		// if cmd.Status.FinishedAt != nil { checkTime = cmd.Status.FinishedAt.Time }

		if checkTime.Before(threshold) {
			// Perform deletion
			toDelete := cmd // Copy to avoid memory aliasing in loop
			if err := gc.Client.Delete(ctx, &toDelete); err != nil {
				// Log error but continue processing others.
				// We do not return REQUEUE here because this is a background loop.
				gc.Log.Error(err, "Failed to delete stale VehicleCommand", "name", toDelete.Name, "namespace", toDelete.Namespace)
			} else {
				deletedCount++
				gc.Log.V(2).Info("Deleted stale VehicleCommand", "name", toDelete.Name, "age", time.Since(checkTime))
			}
		}
	}

	if deletedCount > 0 {
		gc.Log.Info("Completed GC cycle", "deleted_count", deletedCount)
	}
}

// isTerminalState determines if the command has finished its lifecycle.
func isTerminalState(cmd *iovv1alpha2.VehicleCommand) bool {
	phase := cmd.Status.Phase
	return phase == iovv1alpha2.CommandPhaseSucceeded ||
		phase == iovv1alpha2.CommandPhaseFailed
}
