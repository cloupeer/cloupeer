package vehicle

import (
	"context"
	"math"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
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
		otaRequired := true
		if cond := FindLatestCondition(v.Status.Conditions); cond != nil {
			switch cond.Type {
			case iovv1alpha1.ConditionTypeRebooted:
				err = f.Event(ctx, EventSuccess, v)
				otaRequired = false
			case iovv1alpha1.ConditionTypeFailed:
				err = f.Event(ctx, EventFail, v)
				otaRequired = false
			}
		}

		if otaRequired {
			// 模拟 OTA: 下载、安装、重启
			return SimulateOTA(v)
		}

	case iovv1alpha1.VehiclePhaseSucceeded:
		// (Active) Finalize the successful update.
		err = f.Event(ctx, EventFinalize, v)

	case iovv1alpha1.VehiclePhaseFailed:
		// (Active) Handle automated retry logic
		logger.Info("Entering 'Failed' state handler.", "currentAttempt", v.Status.RetryCount)

		lastFailedCond := meta.FindStatusCondition(v.Status.Conditions, iovv1alpha1.ConditionTypeFailed)
		if lastFailedCond == nil {
			// Safeguard
			return ctrl.Result{}, nil
		}

		// 1. Check for manual intervention (new firmware version)
		if lastFailedCond.ObservedGeneration < v.Generation {
			if !isNewVersion(v) {
				// --- User wants to CANCEL ---
				// e.g., Spec changed from v2.0.0 (Failed) -> v1.0.0 (Reported)
				// Per our design decision, we do nothing and remain in Failed state.
				logger.Info("Firmware version reset to reported version. Update canceled. Staying in Failed phase.", "newGeneration", v.Generation)
				// Do nothing.
				break
			}

			// --- User wants to RETRY with a new version ---
			// e.g., Spec changed from v2.0.0 (Failed) -> v2.0.1
			logger.Info("New firmware version specified by user, retrying update immediately.", "newGeneration", v.Generation)
			err = f.Event(ctx, EventRetry, v) // Trigger Failed -> Pending
			break
		}

		// 2. Check max retry count
		const maxRetryCount = 5
		if v.Status.RetryCount >= maxRetryCount {
			logger.Info("Max retry count reached. Giving up.", "attempts", v.Status.RetryCount, "max", maxRetryCount)
			return ctrl.Result{}, nil // Do nothing
		}

		// 3. Calculate exponential backoff
		const baseDelay = 1 * time.Minute
		// 1st retry (RetryCount=0): 2^0 * 1m = 1m
		// 2nd retry (RetryCount=1): 2^1 * 1m = 2m
		// 3rd retry (RetryCount=2): 2^2 * 1m = 4m
		backoffDuration := time.Duration(math.Pow(2, float64(v.Status.RetryCount))) * baseDelay

		elapsed := time.Since(lastFailedCond.LastTransitionTime.Time)
		if elapsed < backoffDuration {
			requeueAfter := backoffDuration - elapsed
			logger.Info("Waiting for exponential backoff before next retry", "nextAttempt", v.Status.RetryCount+1, "requeueAfter", requeueAfter)
			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}

		// 4. Backoff time has passed. Trigger the retry.
		logger.Info("Backoff complete. Triggering retry.", "nextAttempt", v.Status.RetryCount+1)
		err = f.Event(ctx, EventRetry, v) // Trigger EventRetry (Failed -> Pending)

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
