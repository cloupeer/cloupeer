package vehiclecommand

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	iovv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
)

// SubReconciler defines the interface for a modular reconciliation step.
// It operates on the in-memory VehicleCommand object.
// It should return a ctrl.Result if it wants to request a requeue (e.g. for exponential backoff),
// otherwise return empty result.
type SubReconciler interface {
	Reconcile(ctx context.Context, cmd *iovv1alpha1.VehicleCommand) (ctrl.Result, error)
}
