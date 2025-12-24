package vehiclecommand

import (
	"context"
	"fmt"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pb "github.com/autopeer-io/autopeer/api/proto/v1"
	"github.com/autopeer-io/autopeer/internal/pkg/metrics"
	iovv1alpha2 "github.com/autopeer-io/autopeer/pkg/apis/iov/v1alpha2"
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
func (s *SenderReconciler) Reconcile(ctx context.Context, cmd *iovv1alpha2.VehicleCommand) (ctrl.Result, error) {
	// 1. Filter: Only process commands in 'Pending' phase
	if cmd.Status.Phase != iovv1alpha2.CommandPhasePending {
		return ctrl.Result{}, nil
	}

	logger := log.FromContext(ctx)
	logger.Info("Processing Pending command", "command", cmd.Spec.Method, "vehicle", cmd.Spec.VehicleName)

	// 2. Construct the gRPC request
	req := &pb.SendCommandRequest{
		CommandName: cmd.Name,
		VehicleId:   cmd.Spec.VehicleName,
		CommandType: cmd.Spec.Method,
		Parameters:  cmd.Spec.Parameters,
	}

	// 3. Call Hub via interface
	start := time.Now()
	resp, err := s.HubClient.SendCommand(ctx, req)
	duration := time.Since(start).Seconds()
	metrics.CommandLatency.WithLabelValues(string(cmd.Spec.Method)).Observe(duration)
	if err != nil {
		logger.Error(err, "Failed to send command to Hub")
		metrics.CommandSentTotal.WithLabelValues("failure", string(cmd.Spec.Method)).Inc()
		// Return error to trigger exponential backoff requeue by controller-runtime
		return ctrl.Result{}, err
	}

	// 4. Handle Hub Rejection
	if !resp.Accepted {
		logger.Info("Hub rejected the command", "reason", resp.Message)
		metrics.CommandSentTotal.WithLabelValues("rejected", string(cmd.Spec.Method)).Inc()
		MarkFailed(cmd, fmt.Sprintf("Hub rejected: %s", resp.Message))
		return ctrl.Result{}, nil
	}

	// 5. Handle Success
	logger.Info("Command successfully sent to Hub", "hubMessage", resp.Message)
	metrics.CommandSentTotal.WithLabelValues("success", string(cmd.Spec.Method)).Inc()
	MarkSent(cmd, "Command successfully forwarded to Hub")

	// This is strictly "Sent", not yet "Acknowledged" by the vehicle agent.
	// The Hub/Agent async flow will update it to 'Received' later.

	return ctrl.Result{}, nil
}
