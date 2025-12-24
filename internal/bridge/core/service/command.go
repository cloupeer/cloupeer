package service

import (
	"context"
	"fmt"

	"github.com/autopeer-io/autopeer/internal/bridge/core/model"
)

// UpdateCommandStatus handles status reports from the vehicle agent regarding a specific command.
// e.g., Agent reports "I have received command cmd-123" or "I have finished command cmd-123".
func (s *Service) UpdateCommandStatus(ctx context.Context, cmdID string, status model.CommandStatus, message string) error {
	if cmdID == "" {
		return nil // Ignore invalid status reports
	}

	// Delegate to the repository
	// The repository implementation (K8s adapter) will map this to a CRD Status update.
	if err := s.command.UpdateStatus(ctx, cmdID, status, message); err != nil {
		return fmt.Errorf("failed to update command status for %s: %w", cmdID, err)
	}

	return nil
}

// DispatchCommand sends a command to the vehicle via the notifier (MQTT).
func (s *Service) DispatchCommand(ctx context.Context, cmd *model.Command) error {
	// Optional: You could update command status to "Sent" here immediately
	// s.cmdRepo.UpdateStatus(ctx, cmd.ID, model.CommandStatusSent, "")

	return s.notifier.Notify(ctx, cmd)
}
