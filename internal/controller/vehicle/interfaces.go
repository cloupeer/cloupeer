package vehicle

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	iovv1alpha2 "github.com/autopeer-io/autopeer/pkg/apis/iov/v1alpha2"
)

// SubReconciler defines the interface for a modular reconciliation step.
// Each sub-reconciler is responsible for one specific aspect of the Vehicle's logic
// (e.g., state machine, config management, health checks).
type SubReconciler interface {
	// Reconcile is called in order by the main controller.
	// It operates on the provided Vehicle object *in-memory*.
	//
	// It MUST NOT perform its own API calls (like Patch or Update).
	// The main Reconcile loop is responsible for the final, single API call.
	//
	// It returns a ctrl.Result to allow a sub-reconciler to request a
	// requeue *after a delay* (e.g., Result{RequeueAfter: 5s}).
	//
	// To request an *immediate* requeue (e.g., after a state transition),
	// the function should simply return (ctrl.Result{}, nil). The main
	// loop will detect the status change and its subsequent Patch
	// will trigger the new reconcile event.
	Reconcile(ctx context.Context, v *iovv1alpha2.Vehicle) (ctrl.Result, error)
}
