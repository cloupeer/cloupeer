package vehicle

import (
	"context"
	"errors"
	"strings"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	iovv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
)

// phaseHandler defines the signature for a function that handles a specific state machine phase.
// By passing the logger, we avoid the `log.FromContext(ctx)` boilerplate in each handler.
// This signature choice promotes stateless, easily testable handler functions.
type phaseHandler func(ctx context.Context, logger logr.Logger, v *iovv1alpha1.Vehicle) (ctrl.Result, error)

// stateMachine implements the SubReconciler interface for the Vehicle OTA state machine.
type stateMachine struct {
	// handlers is a map of all state-handling functions.
	//
	// This map is read-only after initialization in NewStateMachine(),
	// making it completely safe for concurrent Reconcile calls.
	handlers map[iovv1alpha1.VehiclePhase]phaseHandler
}

// NewStateMachine creates a new state machine sub-reconciler.
// It initializes the handler map, connecting phases to their logic functions.
func NewStateMachine() SubReconciler {
	return &stateMachine{
		handlers: map[iovv1alpha1.VehiclePhase]phaseHandler{
			"":                                  initHandler, // Map empty phase to Idle handler for initialization
			iovv1alpha1.VehiclePhaseIdle:        idleHandler,
			iovv1alpha1.VehiclePhasePending:     pendingHandler,
			iovv1alpha1.VehiclePhaseDownloading: downloadHandler,
			iovv1alpha1.VehiclePhaseInstalling:  installHandler,
			iovv1alpha1.VehiclePhaseRebooting:   rebootHandler,
			iovv1alpha1.VehiclePhaseSucceeded:   successHandler,
			iovv1alpha1.VehiclePhaseFailed:      failedHandler,
		},
	}
}

// Reconcile implements the SubReconciler interface.
// It acts as the dispatcher for the state machine, finding and executing
// the correct handler for the Vehicle's current phase.
func (s *stateMachine) Reconcile(ctx context.Context, v *iovv1alpha1.Vehicle) (ctrl.Result, error) {
	// Get the logger once from the context, which includes
	// reconciliation-specific details like NamespacedName.
	logger := log.FromContext(ctx)

	// Dispatch to the correct handler based on the current phase.
	handler, exists := s.handlers[v.Status.Phase]
	if !exists {
		// This should not happen if all phases are defined in the map.
		logger.Error(nil, "Unknown state machine phase", "phase", v.Status.Phase)
		// No requeue, as this is a programming error, not a transient state.
		return ctrl.Result{}, nil
	}

	// Execute the state-specific logic.
	return handler(ctx, logger, v)
}

// initHandler handles the initialization of a new Vehicle resource.
func initHandler(ctx context.Context, logger logr.Logger, v *iovv1alpha1.Vehicle) (ctrl.Result, error) {
	logger.Info("Initializing Vehicle status: Phase not set, defaulting to Idle.", "vehicle", v.Name)
	v.Status.Phase = iovv1alpha1.VehiclePhaseIdle

	// Simulation: Set a default reported version if one isn't present.
	if v.Status.ReportedFirmwareVersion == "" {
		v.Status.ReportedFirmwareVersion = "v1.0.0"
	}

	// Return an empty result. The main loop will detect the status
	// change, patch it, and the patch will trigger the next reconcile.
	return ctrl.Result{}, nil
}

// idleHandler handles the logic when the Vehicle is in the Idle phase.
func idleHandler(ctx context.Context, logger logr.Logger, v *iovv1alpha1.Vehicle) (ctrl.Result, error) {
	updateRequired := v.Spec.FirmwareVersion != "" && v.Spec.FirmwareVersion != v.Status.ReportedFirmwareVersion
	if updateRequired {
		logger.Info("Update required, moving from Idle to Pending.",
			"specVersion", v.Spec.FirmwareVersion,
			"reportedVersion", v.Status.ReportedFirmwareVersion)

		v.Status.Phase = iovv1alpha1.VehiclePhasePending
		v.Status.ErrorMessage = "" // Clear any previous error

		// State changed. Return empty result. Patch will trigger requeue.
		return ctrl.Result{}, nil
	}

	// No update needed. Stop reconciliation for this cycle.
	return ctrl.Result{}, nil
}

// pendingHandler handles the logic for the Pending phase.
func pendingHandler(ctx context.Context, logger logr.Logger, v *iovv1alpha1.Vehicle) (ctrl.Result, error) {
	logger.Info("Starting OTA process. Moving from Pending to Downloading.")
	v.Status.Phase = iovv1alpha1.VehiclePhaseDownloading
	return ctrl.Result{}, nil
}

// downloadHandler simulates the Downloading phase.
func downloadHandler(ctx context.Context, logger logr.Logger, v *iovv1alpha1.Vehicle) (ctrl.Result, error) {
	// Simulation: Deterministic failure for testing.
	// We simulate a network error for any vehicle whose name ends in "7".
	// This provides a predictable way to test the Failed state.
	if strings.HasSuffix(v.Name, "7") {
		simulatedErr := errors.New("simulated network failure: vehicle name ends in '7' ")
		logger.Error(simulatedErr, "Simulating deterministic network failure.", "vehicleName", v.Name)

		v.Status.Phase = iovv1alpha1.VehiclePhaseFailed
		v.Status.ErrorMessage = simulatedErr.Error()
		return ctrl.Result{}, nil
	}

	// Simulation: Instantaneous download success.
	logger.Info("Download complete. Moving from Downloading to Installing.")
	v.Status.Phase = iovv1alpha1.VehiclePhaseInstalling
	return ctrl.Result{}, nil
}

// installHandler simulates the Installing phase.
func installHandler(ctx context.Context, logger logr.Logger, v *iovv1alpha1.Vehicle) (ctrl.Result, error) {
	logger.Info("Installation complete. Moving from Installing to Rebooting.")
	v.Status.Phase = iovv1alpha1.VehiclePhaseRebooting
	return ctrl.Result{}, nil
}

// rebootHandler simulates the Rebooting phase.
func rebootHandler(ctx context.Context, logger logr.Logger, v *iovv1alpha1.Vehicle) (ctrl.Result, error) {
	logger.Info("Reboot complete. Moving from Rebooting to Succeeded. Updating reported version.")

	v.Status.Phase = iovv1alpha1.VehiclePhaseSucceeded
	v.Status.ReportedFirmwareVersion = v.Spec.FirmwareVersion // Critical: Status now matches Spec
	v.Status.ErrorMessage = ""
	return ctrl.Result{}, nil
}

// successHandler handles the final transition from Succeeded back to Idle.
func successHandler(ctx context.Context, logger logr.Logger, v *iovv1alpha1.Vehicle) (ctrl.Result, error) {
	logger.Info("Update Succeeded. Moving back to Idle.")
	v.Status.Phase = iovv1alpha1.VehiclePhaseIdle
	return ctrl.Result{}, nil
}

// failedHandler handles the logic when the Vehicle is in a terminal Failed state.
func failedHandler(ctx context.Context, logger logr.Logger, v *iovv1alpha1.Vehicle) (ctrl.Result, error) {
	// The vehicle is in a Failed state. We take no action.
	// The controller will wait for an external change (e.g., user
	// updating the spec.firmwareVersion) to trigger a new cycle.
	//
	// We log at the Error level, passing 'nil' as the error object
	// because this reconcile loop itself isn't failing; it's just
	// observing a resource that is already in a failed state.
	logger.Error(nil, "Vehicle is in terminal Failed state. No action will be taken.",
		"vehicleName", v.Name,
		"error", v.Status.ErrorMessage)
	return ctrl.Result{}, nil
}
