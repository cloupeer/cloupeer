package vehiclecommand

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	iovv1alpha2 "github.com/autopeer-io/autopeer/pkg/apis/iov/v1alpha2"
)

// Reconciler reconciles a VehicleCommand object
type Reconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	runners []manager.Runnable

	// subReconcilers is the list of logic processors
	subReconcilers []SubReconciler
}

// NewReconciler creates a new Reconciler for VehicleCommand.
func NewReconciler(cli client.Client, sche *runtime.Scheme, recorder record.EventRecorder, hubAddr string) *Reconciler {
	// Initialize the Hub Client
	hubClient := NewGrpcHubClient(hubAddr)

	return &Reconciler{
		Client:   cli,
		Scheme:   sche,
		Recorder: recorder,
		runners:  []manager.Runnable{hubClient},
		// Register the pipeline steps
		subReconcilers: []SubReconciler{
			NewSenderReconciler(hubClient),
		},
	}
}

//+kubebuilder:rbac:groups=iov.autopeer.io,resources=vehiclecommands,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=iov.autopeer.io,resources=vehiclecommands/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=iov.autopeer.io,resources=vehiclecommands/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile handles the lifecycle of a VehicleCommand.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Fetch the VehicleCommand
	var cmd iovv1alpha2.VehicleCommand
	if err := r.Get(ctx, req.NamespacedName, &cmd); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Handle Deletion (Currently no-op, but ready for Finalizers)
	if !cmd.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// 3. Initialize Status (if new)
	// This ensures the object has a valid Phase before entering SubReconcilers
	if cmd.Status.Phase == "" {
		logger.Info("Initializing VehicleCommand status")
		cmd.Status.Phase = iovv1alpha2.CommandPhasePending
		cmd.Status.Message = "Command created, waiting to be sent"
		if err := r.Status().Update(ctx, &cmd); err != nil {
			logger.Error(err, "Failed to initialize status")
			return ctrl.Result{}, err
		}
		// Status update triggers immediate requeue
		return ctrl.Result{}, nil
	}

	// 4. Create DeepCopy for Patch calculation
	// We modify 'cmd' in place, then compare with 'originalCmd'
	originalCmd := cmd.DeepCopy()

	// 5. Run SubReconcilers
	var aggregatedResult ctrl.Result
	for _, sub := range r.subReconcilers {
		res, err := sub.Reconcile(ctx, &cmd)
		if err != nil {
			// If a step fails, record an event and return error
			logger.Error(err, "Sub-reconciler failed")
			r.Recorder.Event(&cmd, corev1.EventTypeWarning, "ReconcileFailed", err.Error())
			return ctrl.Result{}, err
		}

		// Prioritize the shortest requeue time if multiple steps request it
		if res.RequeueAfter > 0 {
			if aggregatedResult.RequeueAfter == 0 || res.RequeueAfter < aggregatedResult.RequeueAfter {
				aggregatedResult = res
			}
		}
	}

	// 6. Apply Status Patch
	// We only patch if the status has actually changed to reduce API load
	if !equality.Semantic.DeepEqual(originalCmd.Status, cmd.Status) {
		// Log specific transitions
		logger.Info("Patching VehicleCommand Status",
			"oldPhase", originalCmd.Status.Phase,
			"newPhase", cmd.Status.Phase)

		if err := r.Status().Patch(ctx, &cmd, client.MergeFrom(originalCmd)); err != nil {
			logger.Error(err, "Failed to patch status")
			return ctrl.Result{}, err
		}

		// Emit events for phase transitions
		if originalCmd.Status.Phase != cmd.Status.Phase {
			r.Recorder.Eventf(&cmd, corev1.EventTypeNormal, "PhaseChanged",
				"Phase transitioned from %s to %s", originalCmd.Status.Phase, cmd.Status.Phase)
		}
	}

	return aggregatedResult, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	gc := &GarbageCollector{
		Client:            mgr.GetClient(),
		Log:               mgr.GetLogger().WithName("gc-vehicle-command"),
		RetentionDuration: 30 * 24 * time.Hour, // Configurable via options later
		CleanupInterval:   1 * time.Hour,       // Check every hour
	}

	r.runners = append(r.runners, gc)

	for _, runner := range r.runners {
		if err := mgr.Add(runner); err != nil {
			return err
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&iovv1alpha2.VehicleCommand{}).
		Complete(r)
}
