package vehicle

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	iovv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
)

// SubStateMachine 实现了 SubReconciler 接口
type SubStateMachine struct{}

// NewStateMachine 创建一个新的 state machine sub-reconciler.
func NewSubStateMachine() SubReconciler {
	return &SubStateMachine{}
}

// Reconcile 实现了 SubReconciler 接口
func (s *SubStateMachine) Reconcile(ctx context.Context, v *iovv1alpha1.Vehicle) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 初始化状态
	if v.Status.Phase == "" {
		logger.Info("Initializing Vehicle status: Phase not set, defaulting to Idle.")
		v.Status.Phase = iovv1alpha1.VehiclePhaseIdle
		SetCondition(v, iovv1alpha1.ConditionTypeReady, metav1.ConditionTrue, "Initialized", "Vehicle is ready")
		return ctrl.Result{}, nil // Patching a new status will trigger requeue
	}

	var err error
	f := NewFiniteStateMachine(string(v.Status.Phase))

	// 根据当前状态触发事件
	switch v.Status.Phase {

	case iovv1alpha1.VehiclePhaseIdle:
		// (Active) Try to start an update.
		err = f.Event(ctx, EventUpdate, v)

	case iovv1alpha1.VehiclePhasePending:
		// TODO: 根据上报的 Condition Type 和 Status 判断 成功 or 失败 ？
		// err = f.Event(ctx, EventSuccess, v)
		// err = f.Event(ctx, EventFail, v)

	case iovv1alpha1.VehiclePhaseSucceeded:
		// (Active) Finalize the successful update.
		err = f.Event(ctx, EventFinalize, v)

	case iovv1alpha1.VehiclePhaseFailed:
		// TODO: 失败可等待人工干预，或者自动重试
		// ...

	// Default: Do nothing for Downloading, Installing, Rebooting.
	// Their 'enter_' callbacks handle the logic, preventing Reconcile
	// from interfering.
	default:
		// ...
	}

	// Handle FSM transition errors (e.g., CanceledError)
	if isFsmRealError(err) {
		logger.Error(err, "Error during FSM event processing")
		return ctrl.Result{}, err // 向上抛出，触发指数退避
	}

	// Sync FSM's internal state back to our CRD status.
	v.Status.Phase = iovv1alpha1.VehiclePhase(f.Current())

	// Return empty result. If the status changed, the main controller's
	// Patch() will trigger the next Reconcile.
	return ctrl.Result{}, nil
}
