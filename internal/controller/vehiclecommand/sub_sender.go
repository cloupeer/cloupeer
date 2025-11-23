package vehiclecommand

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pb "cloupeer.io/cloupeer/api/proto/v1"
	iovv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
)

// SenderReconciler is responsible for sending pending commands to the Hub.
type SenderReconciler struct {
	HubClient HubClient
}

var _ SubReconciler = (*SenderReconciler)(nil)

func NewSenderReconciler(hubClient HubClient) *SenderReconciler {
	return &SenderReconciler{
		HubClient: hubClient,
	}
}

// Reconcile implements the SubReconciler interface.
func (s *SenderReconciler) Reconcile(ctx context.Context, cmd *iovv1alpha1.VehicleCommand) (ctrl.Result, error) {
	// 1. Filter: Only process commands in 'Pending' phase
	if cmd.Status.Phase != iovv1alpha1.CommandPhasePending {
		return ctrl.Result{}, nil
	}

	logger := log.FromContext(ctx)
	logger.Info("Processing Pending command", "command", cmd.Spec.Command, "vehicle", cmd.Spec.VehicleName)

	// 2. Construct the gRPC request
	req := &pb.SendCommandRequest{
		CommandName: cmd.Name,
		VehicleId:   cmd.Spec.VehicleName,
		CommandType: string(cmd.Spec.Command),
		Parameters:  cmd.Spec.Parameters,
	}

	// 3. Call Hub via interface
	resp, err := s.HubClient.SendCommand(ctx, req)
	if err != nil {
		logger.Error(err, "Failed to send command to Hub")
		// Return error to trigger exponential backoff requeue by controller-runtime
		return ctrl.Result{}, err
	}

	// 4. Handle Hub Rejection
	if !resp.Accepted {
		logger.Info("Hub rejected the command", "reason", resp.Message)
		MarkFailed(cmd, fmt.Sprintf("Hub rejected: %s", resp.Message))
		return ctrl.Result{}, nil
	}

	// 5. Handle Success
	logger.Info("Command successfully sent to Hub", "hubMessage", resp.Message)
	MarkSent(cmd, "Command successfully forwarded to Hub")

	// This is strictly "Sent", not yet "Acknowledged" by the vehicle agent.
	// The Hub/Agent async flow will update it to 'Received' later.

	return ctrl.Result{}, nil
}
