package vehiclecommand

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	iovv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
)

// HubClient defines the interface for communicating with the Cloupeer Hub.
// This allows us to mock the gRPC interaction for now.
type HubClient interface {
	// SendCommand transmits the command payload to the Hub.
	SendCommand(ctx context.Context, cmd *iovv1alpha1.VehicleCommand) error
}

// Reconciler reconciles a VehicleCommand object
type Reconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	HubClient HubClient
}

// NewReconciler creates a new Reconciler for VehicleCommand.
func NewReconciler(cli client.Client, sche *runtime.Scheme, recorder record.EventRecorder) *Reconciler {
	return &Reconciler{
		Client:    cli,
		Scheme:    sche,
		Recorder:  recorder,
		HubClient: &mockHubClient{}, // TODO: Inject real gRPC client later
	}
}

//+kubebuilder:rbac:groups=iov.cloupeer.io,resources=vehiclecommands,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=iov.cloupeer.io,resources=vehiclecommands/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=iov.cloupeer.io,resources=vehiclecommands/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile handles the lifecycle of a VehicleCommand.
// Logic flow:
// 1. New (Phase="") -> Pending
// 2. Pending -> Call Hub -> Sent
// 3. Sent -> (Wait for Hub/Agent to update status asynchronously)
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Fetch the VehicleCommand
	var cmd iovv1alpha1.VehicleCommand
	if err := r.Get(ctx, req.NamespacedName, &cmd); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Handle Deletion (Optional: Cancel command on Hub if possible)
	if !cmd.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// 3. Initialize Status
	if cmd.Status.Phase == "" {
		logger.Info("Initializing VehicleCommand status")
		cmd.Status.Phase = iovv1alpha1.CommandPhasePending
		cmd.Status.Message = "Command created, waiting to be sent"
		if err := r.Status().Update(ctx, &cmd); err != nil {
			logger.Error(err, "Failed to initialize status")
			return ctrl.Result{}, err
		}
		// Status update triggers immediate requeue
		return ctrl.Result{}, nil
	}

	// 4. State Machine
	switch cmd.Status.Phase {
	case iovv1alpha1.CommandPhasePending:
		logger.Info("Processing Pending command", "type", cmd.Spec.Command, "vehicle", cmd.Spec.VehicleName)

		// Send to Hub
		if err := r.HubClient.SendCommand(ctx, &cmd); err != nil {
			logger.Error(err, "Failed to send command to Hub")

			// Update status to Failed if it's a non-retriable error (simplified here)
			// For now, we let controller-runtime retry via exponential backoff by returning error
			return ctrl.Result{}, err
		}

		// Success: Update Phase to Sent
		now := metav1.Now()
		cmd.Status.Phase = iovv1alpha1.CommandPhaseSent
		cmd.Status.Message = "Command successfully sent to Hub"
		cmd.Status.LastUpdateTime = &now

		if err := r.Status().Update(ctx, &cmd); err != nil {
			logger.Error(err, "Failed to update status to Sent")
			return ctrl.Result{}, err
		}

		r.Recorder.Event(&cmd, corev1.EventTypeNormal, "Sent", "Command sent to Hub")

	case iovv1alpha1.CommandPhaseSent,
		iovv1alpha1.CommandPhaseReceived,
		iovv1alpha1.CommandPhaseRunning:
		// In these states, the Controller is PASSIVE.
		// It waits for the Hub (via a separate callback/webhook mechanism) to update the CR status.
		// Or, if we implement polling later, we might do it here.
		// For now, do nothing.

	case iovv1alpha1.CommandPhaseSucceeded, iovv1alpha1.CommandPhaseFailed:
		// Terminal states. Do nothing.
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iovv1alpha1.VehicleCommand{}).
		Complete(r)
}

// --- Mock Implementation (Temporary) ---

type mockHubClient struct{}

func (m *mockHubClient) SendCommand(ctx context.Context, cmd *iovv1alpha1.VehicleCommand) error {
	// Simulate gRPC call latency
	time.Sleep(100 * time.Millisecond)

	// In real implementation, this would call:
	// pb.NewHubClient(conn).SendCommand(...)

	fmt.Printf(">> [MOCK gRPC] Sending Command to Hub: Vehicle=%s Type=%s Payload=%v\n",
		cmd.Spec.VehicleName, cmd.Spec.Command, cmd.Spec.Parameters)

	return nil
}
