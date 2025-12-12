package vehiclecommand

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	iovv1alpha2 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha2"
)

// SubReconciler defines the interface for a modular reconciliation step.
// It operates on the in-memory VehicleCommand object.
// It should return a ctrl.Result if it wants to request a requeue (e.g. for exponential backoff),
// otherwise return empty result.
type SubReconciler interface {
	Reconcile(ctx context.Context, cmd *iovv1alpha2.VehicleCommand) (ctrl.Result, error)
}
