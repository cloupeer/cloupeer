package vehicle

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	iovv1alpha2 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha2"
)

type Reconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	// subReconcilers is the chain of business logic plugins.
	// They are executed sequentially on each reconciliation.
	subReconcilers []SubReconciler
}

// NewReconciler creates a new vehicle Reconciler.
// This constructor follows the "encapsulated" pattern (vs. dependency injection)
// by instantiating its own sub-reconciler chain. This simplifies
// the registration in manager.go.
func NewReconciler(cli client.Client, sche *runtime.Scheme, recorder record.EventRecorder) *Reconciler {
	r := &Reconciler{
		Client:   cli,
		Scheme:   sche,
		Recorder: recorder,
	}

	// This is the "plugin" registration.
	// We can add more sub-reconcilers here (e.g., NewConfigReconciler())
	// and they will be executed in order.
	r.subReconcilers = []SubReconciler{
		NewSubStateMachine(cli),
	}

	return r
}

// RBAC markers are used by controller-gen to generate the ClusterRole
// +kubebuilder:rbac:groups=iov.cloupeer.io,resources=vehicles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=iov.cloupeer.io,resources=vehicles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=iov.cloupeer.io,resources=vehicles/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is the core logic for the Vehicle controller.
// This function is driven by events (Create, Update, Delete) and aims to
// move the current state (Status) of the cluster closer to the desired state (Spec).
//
// This simulation implements an instantaneous state machine, where each state
// transition occurs immediately in the subsequent reconcile loop, driven by
// the Status.Patch() event.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Get the logger from the context, which is the standard
	// controller-runtime practice.
	logger := log.FromContext(ctx)
	logger.Info("Starting reconcile cycle...")

	// Fetch the Vehicle resource
	var vehicle iovv1alpha2.Vehicle
	if err := r.Get(ctx, req.NamespacedName, &vehicle); err != nil {
		// We use client.IgnoreNotFound(err) to gracefully handle
		// deletion events. When a resource is deleted, a reconcile is
		// triggered, r.Get() fails with "not found", and IgnoreNotFound
		// returns 'nil', causing us to exit cleanly.
		// Any other error (e.g., permissions) will be returned,
		// triggering a requeue.
		if client.IgnoreNotFound(err) != nil {
			// This is a "real" error (e.g., network, RBAC)
			logger.Error(err, "unable to fetch Vehicle")
		}

		// If the error was "not found", we just return an empty result
		// and stop the reconciliation.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create a deep copy of the original object.
	// This is the best practice for using r.Status().Patch().
	// client.MergeFrom() will calculate the "diff" between originalVehicle
	// and the modified 'vehicle' object.
	originalVehicle := vehicle.DeepCopy()

	// 【Hack】Force patch generation for RetryCount=0.
	if originalVehicle.Status.UpgradeStatus.RetryCount == 0 {
		originalVehicle.Status.UpgradeStatus.RetryCount = -1
	}

	// Handle Finalizer logic
	if !vehicle.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleVehicleDeletion(ctx, logger, &vehicle, originalVehicle)
	}

	// --- The object is NOT being deleted ---
	if !controllerutil.ContainsFinalizer(&vehicle, iovv1alpha2.VehicleFinalizer) {
		return r.addFinalizer(ctx, logger, &vehicle, originalVehicle)
	}

	// Run the sub-reconciler chain.
	// We aggregate the result. The first request for a delayed requeue wins.
	var aggregatedResult ctrl.Result
	for _, sub := range r.subReconcilers {
		result, err := sub.Reconcile(ctx, &vehicle)
		if err != nil {
			logger.Error(err, "Sub-reconciler failed", "subReconciler", sub)
			// Create a Kubernetes event to broadcast the failure
			r.Recorder.Event(&vehicle, corev1.EventTypeWarning, "ReconcileFailed", err.Error())
			return ctrl.Result{}, err
		}

		// Aggregate the requeue result.
		// If the current result wants a requeue *after a delay*,
		// and we don't already have one, or if its RequeueAfter is
		// *shorter* than the one we have, we take it.
		// Immediate requeues are handled implicitly by the patch below.
		if result.RequeueAfter > 0 {
			if aggregatedResult.RequeueAfter == 0 || result.RequeueAfter < aggregatedResult.RequeueAfter {
				aggregatedResult = result
			}
		}
	}

	// Compare and Patch Spec (if changed)
	// We must compare *before* patching to avoid unnecessary API calls.
	// We compare Spec separately because it uses a different API endpoint
	// than the Status subresource.
	if !equality.Semantic.DeepEqual(originalVehicle.Spec, vehicle.Spec) {
		logger.Info("Patching Vehicle Spec")
		if err := r.Patch(ctx, &vehicle, client.MergeFrom(originalVehicle)); err != nil {
			logger.Error(err, "Failed to patch Vehicle Spec")
			return ctrl.Result{}, err
		}
	}

	// Compare and Patch Status (if changed)
	// This is the critical check to prevent infinite reconciliation loops.
	// If the status has not changed, we DO NOT patch.
	if !equality.Semantic.DeepEqual(originalVehicle.Status, vehicle.Status) {
		oldPhase := originalVehicle.Status.UpgradeStatus.Phase
		newPhase := vehicle.Status.UpgradeStatus.Phase
		logger.Info("Patching Vehicle Status", "oldPhase", oldPhase, "newPhase", newPhase)

		if err := r.Status().Patch(ctx, &vehicle, client.MergeFrom(originalVehicle)); err != nil {
			logger.Error(err, "Failed to patch Vehicle Status")
			return ctrl.Result{}, err
		}

		// If the phase changed, record a human-readable event
		if oldPhase != newPhase {
			r.Recorder.Eventf(&vehicle, corev1.EventTypeNormal, "PhaseChanged", "Vehicle phase changed from %s to %s", oldPhase, newPhase)
		}
	}

	// Return the aggregated result (likely just an empty result or a RequeueAfter).
	return aggregatedResult, nil
}

func (r *Reconciler) handleVehicleDeletion(ctx context.Context, logger logr.Logger, vehicle, originalVehicle *iovv1alpha2.Vehicle) (ctrl.Result, error) {
	// --- The object is being deleted ---
	if controllerutil.ContainsFinalizer(vehicle, iovv1alpha2.VehicleFinalizer) {
		logger.Info("Handling Finalizer: Deletion detected, running cleanup logic...")

		// Execute our cleanup logic
		if err := r.clearVehicle(ctx, vehicle); err != nil {
			// If cleanup fails, return the error. Kubernetes will retry.
			logger.Error(err, "Failed to execute deletion handler")
			r.Recorder.Event(vehicle, corev1.EventTypeWarning, "CleanupFailed", err.Error())
			return ctrl.Result{}, err
		}

		// Cleanup successful, remove the Finalizer
		logger.Info("Cleanup successful, removing Finalizer.")
		controllerutil.RemoveFinalizer(vehicle, iovv1alpha2.VehicleFinalizer)

		// Patch the object to remove the finalizer
		// We use the originalVehicle from the start of the reconcile
		if err := r.Patch(ctx, vehicle, client.MergeFrom(originalVehicle)); err != nil {
			logger.Error(err, "Failed to remove Finalizer by patching")
			return ctrl.Result{}, err
		}

		// Stop the reconcile loop, as the object is being deleted
		return ctrl.Result{}, nil
	}

	// Finalizer already removed, nothing to do.
	return ctrl.Result{}, nil
}

func (r *Reconciler) addFinalizer(ctx context.Context, logger logr.Logger, vehicle, originalVehicle *iovv1alpha2.Vehicle) (ctrl.Result, error) {
	logger.Info("Adding Finalizer to new/updated Vehicle.")
	controllerutil.AddFinalizer(vehicle, iovv1alpha2.VehicleFinalizer)

	// Patch the object to add the finalizer
	if err := r.Patch(ctx, vehicle, client.MergeFrom(originalVehicle)); err != nil {
		logger.Error(err, "Failed to add Finalizer by patching")
		return ctrl.Result{}, err
	}

	// Return immediately. The Patch operation will trigger a
	// new Reconcile event. This ensures we process the
	// sub-reconcilers only after the finalizer is confirmed.
	return ctrl.Result{}, nil
}

// clearVehicle contains the business logic required to clean up
// before a Vehicle resource is deleted.
func (r *Reconciler) clearVehicle(ctx context.Context, v *iovv1alpha2.Vehicle) error {
	logger := log.FromContext(ctx)

	// --- Simulate cleanup logic ---
	// In a real-world scenario, this is where you would:
	// 1. Call the vehicle's telematics API to remotely unbind/deactivate.
	// 2. Notify an external inventory system.
	// 3. Delete associated resources (e.g., cloud-side digital twin).

	logger.Info("Simulating vehicle unbinding...", "vehicleName", v.Name)

	// Simulate a process that could fail.
	// if v.Name == "vh-fail-delete" {
	//     return errors.New("simulated telematics API failure")
	// }

	// Simulate success
	logger.Info("Vehicle unbinding simulation complete.")

	return nil
}

func (r *Reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iovv1alpha2.Vehicle{}).
		Owns(&iovv1alpha2.VehicleCommand{}).
		Complete(r)
}
