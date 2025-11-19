package vehiclecommand

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pb "cloupeer.io/cloupeer/api/proto/v1"
	iovv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
)

// Reconciler reconciles a VehicleCommand object
type Reconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	HubAddr  string
}

// NewReconciler creates a new Reconciler for VehicleCommand.
func NewReconciler(cli client.Client, sche *runtime.Scheme, recorder record.EventRecorder, hubAddr string) *Reconciler {
	return &Reconciler{
		Client:   cli,
		Scheme:   sche,
		Recorder: recorder,
		HubAddr:  hubAddr, // TODO: Inject real gRPC client later
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

		// 建立 gRPC 连接
		// 在生产环境中，最好维护一个全局单例的 Connection，而不是每次 Reconcile 都 Dial。
		// 但为了演示清晰和简单，我们这里采用短连接（或者依靠 gRPC 内部的连接池机制）。
		// 使用 Insecure 凭证，因为集群内部通过 Service 通信通常是可信网络，或者是通过 mTLS (Linkerd/Istio) 处理的。
		conn, err := grpc.NewClient(r.HubAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Error(err, "Failed to connect to Hub")
			return ctrl.Result{}, err
		}
		defer conn.Close()

		hubClient := pb.NewHubServiceClient(conn)

		req := &pb.SendCommandRequest{
			VehicleId:   cmd.Spec.VehicleName,
			CommandType: string(cmd.Spec.Command),
			Parameters:  cmd.Spec.Parameters,
		}

		// Send to Hub
		resp, err := hubClient.SendCommand(ctx, req)
		if err != nil {
			logger.Error(err, "gRPC SendCommand call failed")
			r.Recorder.Event(&cmd, corev1.EventTypeWarning, "SendFailed", err.Error())
			return ctrl.Result{}, err
		}

		if !resp.Accepted {
			logger.Info("Hub rejected the command", "reason", resp.Message)
			r.Recorder.Eventf(&cmd, corev1.EventTypeWarning, "Rejected", "Hub rejected: %s", resp.Message)
			// 如果 Hub 明确拒绝，可能不需要重试，而是标记为 Failed？
			// 这里我们暂时按 Failed 处理
			cmd.Status.Phase = iovv1alpha1.CommandPhaseFailed
			cmd.Status.Message = fmt.Sprintf("Hub rejected: %s", resp.Message)
			r.Status().Update(ctx, &cmd)
			return ctrl.Result{}, nil
		}

		// 4. 成功，更新状态
		logger.Info("Command successfully sent to Hub", "hubMessage", resp.Message)

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
