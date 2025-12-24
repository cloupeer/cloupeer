package vehiclecommand

import (
	iovv1alpha2 "github.com/autopeer-io/autopeer/pkg/apis/iov/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MarkSent updates the command status to Sent and records the timestamp.
func MarkSent(cmd *iovv1alpha2.VehicleCommand, msg string) {
	now := metav1.Now()
	cmd.Status.Phase = iovv1alpha2.CommandPhaseSent
	cmd.Status.Message = msg
	cmd.Status.LastUpdateTime = &now
}

// MarkFailed updates the command status to Failed, records error message and completion time.
func MarkFailed(cmd *iovv1alpha2.VehicleCommand, errMessage string) {
	now := metav1.Now()
	cmd.Status.Phase = iovv1alpha2.CommandPhaseFailed
	cmd.Status.Message = errMessage
	cmd.Status.LastUpdateTime = &now
	cmd.Status.CompletionTime = &now
}

// MarkSucceeded updates the command status to Succeeded.
func MarkSucceeded(cmd *iovv1alpha2.VehicleCommand) {
	now := metav1.Now()
	cmd.Status.Phase = iovv1alpha2.CommandPhaseSucceeded
	cmd.Status.Message = "Command executed successfully"
	cmd.Status.LastUpdateTime = &now
	cmd.Status.CompletionTime = &now
}
