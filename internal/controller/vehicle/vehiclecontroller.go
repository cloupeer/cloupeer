package vehicle

import (
	"context"
	"errors"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	iovv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
)

type VehicleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewReconciler(cli client.Client, sche *runtime.Scheme) *VehicleReconciler {
	return &VehicleReconciler{
		Client: cli,
		Scheme: sche,
	}
}

// +kubebuilder:rbac:groups=iov.cloupeer.io,resources=vehicles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=iov.cloupeer.io,resources=vehicles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=iov.cloupeer.io,resources=vehicles/finalizers,verbs=update

// Reconcile is the core logic for the Vehicle controller.
// This function is driven by events (Create, Update, Delete) and aims to
// move the current state (Status) of the cluster closer to the desired state (Spec).
//
// This simulation implements an instantaneous state machine, where each state
// transition occurs immediately in the subsequent reconcile loop, driven by
// the Status.Patch() event.
func (r *VehicleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Get the logger from the context, which is the standard
	// controller-runtime practice.
	logger := log.FromContext(ctx)
	logger.Info("Starting reconcile cycle...", "NamespacedName", req.NamespacedName)

	// 1. Fetch the Vehicle resource
	var vehicle iovv1alpha1.Vehicle
	if err := r.Get(ctx, req.NamespacedName, &vehicle); err != nil {
		logger.Error(err, "unable to fetch Vehicle")
		// We'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification).
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Create a deep copy of the original object.
	// This is the best practice for using r.Status().Patch().
	// client.MergeFrom() will calculate the "diff" between originalVehicle
	// and the modified 'vehicle' object.
	originalVehicle := vehicle.DeepCopy()

	// 3. Handle Status Initialization
	// If Phase is empty (zero-value), this is a newly created CR.
	// We must initialize its status to a known good state.
	if vehicle.Status.Phase == "" {
		logger.Info("Initializing Vehicle status: Phase not set, defaulting to Idle.", "vehicle", vehicle.Name)

		vehicle.Status.Phase = iovv1alpha1.VehiclePhaseIdle

		// For simulation purposes, we set a default reported version
		// if one isn't already present.
		if vehicle.Status.ReportedFirmwareVersion == "" {
			vehicle.Status.ReportedFirmwareVersion = "v1.0.0" // Simulation: Default version
		}

		// Use Patch to apply the status update.
		if err := r.Status().Patch(ctx, &vehicle, client.MergeFrom(originalVehicle)); err != nil {
			logger.Error(err, "Failed to patch Vehicle status for initialization")
			// Return error to requeue the request
			return ctrl.Result{}, err
		}

		// Status patch will trigger a new reconcile loop.
		// Return here to process the resource in its new 'Idle' state.
		return ctrl.Result{}, nil
	}

	// 4. State Machine Simulation
	// Process the resource based on its current phase.
	switch vehicle.Status.Phase {

	case iovv1alpha1.VehiclePhaseIdle:
		// In Idle state, we check if an update is required.
		updateRequired := vehicle.Spec.FirmwareVersion != "" &&
			vehicle.Spec.FirmwareVersion != vehicle.Status.ReportedFirmwareVersion

		if updateRequired {
			logger.Info("Update required, moving from Idle to Pending.",
				"specVersion", vehicle.Spec.FirmwareVersion,
				"reportedVersion", vehicle.Status.ReportedFirmwareVersion)

			vehicle.Status.Phase = iovv1alpha1.VehiclePhasePending
			vehicle.Status.ErrorMessage = "" // Clear any previous error

			if err := r.Status().Patch(ctx, &vehicle, client.MergeFrom(originalVehicle)); err != nil {
				logger.Error(err, "Failed to patch status from Idle to Pending")
				return ctrl.Result{}, err
			}
			// The patch will trigger the next reconcile loop.
			return ctrl.Result{}, nil
		}

		// No update needed. Log is commented out to prevent flooding.
		// logger.Info("Vehicle is Idle and up-to-date.")
		return ctrl.Result{}, nil

	case iovv1alpha1.VehiclePhasePending:
		// (Simulate) Controller "processes" the request and starts the download.
		logger.Info("Starting OTA process. Moving from Pending to Downloading.")

		vehicle.Status.Phase = iovv1alpha1.VehiclePhaseDownloading
		if err := r.Status().Patch(ctx, &vehicle, client.MergeFrom(originalVehicle)); err != nil {
			logger.Error(err, "Failed to patch status from Pending to Downloading")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil

	case iovv1alpha1.VehiclePhaseDownloading:
		// --- Simulation: Deterministic Failure ---
		// We simulate a network error for any vehicle whose name ends in "7".
		// This provides a predictable way to test the Failed state.
		if strings.HasSuffix(vehicle.Name, "7") {
			simulatedErr := errors.New("simulated network failure: vehicle name ends in '7'")
			logger.Error(simulatedErr, "Simulating deterministic network failure.", "vehicleName", vehicle.Name)

			vehicle.Status.Phase = iovv1alpha1.VehiclePhaseFailed
			vehicle.Status.ErrorMessage = simulatedErr.Error()

			if err := r.Status().Patch(ctx, &vehicle, client.MergeFrom(originalVehicle)); err != nil {
				logger.Error(err, "Failed to patch status to Failed")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}
		// --- End Simulation ---

		// (Simulate) Instantaneous download success.
		logger.Info("Download complete. Moving from Downloading to Installing.")

		vehicle.Status.Phase = iovv1alpha1.VehiclePhaseInstalling
		if err := r.Status().Patch(ctx, &vehicle, client.MergeFrom(originalVehicle)); err != nil {
			logger.Error(err, "Failed to patch status from Downloading to Installing")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil

	case iovv1alpha1.VehiclePhaseInstalling:
		// (Simulate) Instantaneous installation success.
		logger.Info("Installation complete. Moving from Installing to Rebooting.")

		vehicle.Status.Phase = iovv1alpha1.VehiclePhaseRebooting
		if err := r.Status().Patch(ctx, &vehicle, client.MergeFrom(originalVehicle)); err != nil {
			logger.Error(err, "Failed to patch status from Installing to Rebooting")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil

	case iovv1alpha1.VehiclePhaseRebooting:
		// (Simulate) Instantaneous reboot success.
		// This is the final step where we update the reported version.
		logger.Info("Reboot complete. Moving from Rebooting to Succeeded. Updating reported version.")

		vehicle.Status.Phase = iovv1alpha1.VehiclePhaseSucceeded
		// This is the most critical part of the simulation:
		// The status (ReportedFirmwareVersion) is updated to match the spec.
		vehicle.Status.ReportedFirmwareVersion = vehicle.Spec.FirmwareVersion
		vehicle.Status.ErrorMessage = ""

		if err := r.Status().Patch(ctx, &vehicle, client.MergeFrom(originalVehicle)); err != nil {
			logger.Error(err, "Failed to patch status from Rebooting to Succeeded")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil

	case iovv1alpha1.VehiclePhaseSucceeded:
		// (Simulate) Final transition to complete the loop.
		logger.Info("Update Succeeded. Moving back to Idle.")

		vehicle.Status.Phase = iovv1alpha1.VehiclePhaseIdle
		if err := r.Status().Patch(ctx, &vehicle, client.MergeFrom(originalVehicle)); err != nil {
			logger.Error(err, "Failed to patch status from Succeeded to Idle")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil

	case iovv1alpha1.VehiclePhaseFailed:
		// The vehicle is in a Failed state. We take no action.
		// The controller will wait for an external change (e.g., user
		// updating the spec.firmwareVersion) to trigger a new cycle.
		//
		// We log at the Error level, passing 'nil' as the error object
		// because this reconcile loop itself isn't failing; it's just
		// observing a resource that is already in a failed state.
		logger.Error(nil, "Vehicle is in terminal Failed state. No action will be taken.",
			"vehicleName", vehicle.Name,
			"error", vehicle.Status.ErrorMessage)

		// No requeue.
		return ctrl.Result{}, nil

	}

	// This line should be unreachable if all phases are handled
	// in the switch statement.
	logger.Error(nil, "Unhandled Reconcile state", "phase", vehicle.Status.Phase)
	return ctrl.Result{}, nil
}

func (r *VehicleReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iovv1alpha1.Vehicle{}).
		Complete(r)
}
