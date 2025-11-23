package vehiclecommand

import (
	iovv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MarkSent updates the command status to Sent and records the timestamp.
func MarkSent(cmd *iovv1alpha1.VehicleCommand, msg string) {
	now := metav1.Now()
	cmd.Status.Phase = iovv1alpha1.CommandPhaseSent
	cmd.Status.Message = msg
	cmd.Status.LastUpdateTime = &now
}

// MarkFailed updates the command status to Failed, records error message and completion time.
func MarkFailed(cmd *iovv1alpha1.VehicleCommand, errMessage string) {
	now := metav1.Now()
	cmd.Status.Phase = iovv1alpha1.CommandPhaseFailed
	cmd.Status.Message = errMessage
	cmd.Status.LastUpdateTime = &now
	cmd.Status.CompletionTime = &now
}

// MarkSucceeded updates the command status to Succeeded.
func MarkSucceeded(cmd *iovv1alpha1.VehicleCommand) {
	now := metav1.Now()
	cmd.Status.Phase = iovv1alpha1.CommandPhaseSucceeded
	cmd.Status.Message = "Command executed successfully"
	cmd.Status.LastUpdateTime = &now
	cmd.Status.CompletionTime = &now
}
