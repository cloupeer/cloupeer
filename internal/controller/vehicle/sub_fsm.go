package vehicle

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	iovv1alpha2 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha2"
)

// SubStateMachine 实现了 SubReconciler 接口
type SubStateMachine struct {
	client.Client
}

// NewStateMachine 创建一个新的 state machine sub-reconciler.
func NewSubStateMachine(cli client.Client) SubReconciler {
	return &SubStateMachine{Client: cli}
}

// Reconcile 实现了 SubReconciler 接口
func (s *SubStateMachine) Reconcile(ctx context.Context, v *iovv1alpha2.Vehicle) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 初始化状态
	if v.Status.UpgradeStatus.Phase == "" {
		logger.Info("Initializing Vehicle status: Phase not set, defaulting to Idle.")
		v.Status.UpgradeStatus.Phase = iovv1alpha2.VehiclePhaseIdle
		SetCondition(v, iovv1alpha2.ConditionTypeReady, metav1.ConditionTrue, "Initialized", "Vehicle is ready")
		return ctrl.Result{}, nil // Patching a new status will trigger requeue
	}

	var err error
	f := NewFiniteStateMachine(string(v.Status.UpgradeStatus.Phase))

	// 根据当前状态触发事件
	switch v.Status.UpgradeStatus.Phase {

	case iovv1alpha2.VehiclePhaseIdle:
		// (Active) Try to start an update.
		err = f.Event(ctx, EventUpdate, v)

	case iovv1alpha2.VehiclePhasePending:
		err = s.handlePendingPhase(ctx, f, v)

	case iovv1alpha2.VehiclePhaseSucceeded:
		// (Active) Finalize the successful update.
		err = f.Event(ctx, EventFinalize, v)

	case iovv1alpha2.VehiclePhaseFailed:
		// (Active) Handle automated retry logic
		logger.Info("Entering 'Failed' state handler.", "currentAttempt", v.Status.UpgradeStatus.RetryCount)

		failedCond := meta.FindStatusCondition(v.Status.Conditions, iovv1alpha2.ConditionTypeSynced)
		if failedCond == nil || failedCond.Status == metav1.ConditionTrue {
			// Safeguard
			return ctrl.Result{}, nil
		}

		// 1. Check for manual intervention (new firmware version)
		if failedCond.ObservedGeneration < v.Generation {
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
		if v.Status.UpgradeStatus.RetryCount >= maxRetryCount {
			logger.Info("Max retry count reached. Giving up.", "attempts", v.Status.UpgradeStatus.RetryCount, "max", maxRetryCount)
			return ctrl.Result{}, nil // Do nothing
		}

		// 3. Calculate exponential backoff
		const baseDelay = 1 * time.Minute
		// 1st retry (RetryCount=0): 2^0 * 1m = 1m
		// 2nd retry (RetryCount=1): 2^1 * 1m = 2m
		// 3rd retry (RetryCount=2): 2^2 * 1m = 4m
		backoffDuration := time.Duration(math.Pow(2, float64(v.Status.UpgradeStatus.RetryCount))) * baseDelay

		elapsed := time.Since(failedCond.LastTransitionTime.Time)
		if elapsed < backoffDuration {
			requeueAfter := backoffDuration - elapsed
			logger.Info("Waiting for exponential backoff before next retry", "nextAttempt", v.Status.UpgradeStatus.RetryCount+1, "requeueAfter", requeueAfter)
			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}

		// 4. Backoff time has passed. Trigger the retry.
		logger.Info("Backoff complete. Triggering retry.", "nextAttempt", v.Status.UpgradeStatus.RetryCount+1)
		err = f.Event(ctx, EventRetry, v) // Trigger EventRetry (Failed -> Pending)

	default:
		// Default: Do nothing.
	}

	// Handle FSM transition errors (e.g., CanceledError)
	if isFsmRealError(err) {
		logger.Error(err, "Error during FSM event processing")
		return ctrl.Result{}, err // 向上抛出，触发指数退避
	}

	// Sync FSM's internal state back to our CRD status.
	v.Status.UpgradeStatus.Phase = iovv1alpha2.VehiclePhase(f.Current())

	// Return empty result. If the status changed, the main controller's
	// Patch() will trigger the next Reconcile.
	return ctrl.Result{}, nil
}

func (s *SubStateMachine) handlePendingPhase(ctx context.Context, f *FiniteStateMachine, v *iovv1alpha2.Vehicle) error {
	logger := log.FromContext(ctx)

	// TODO: FirmwareVersion 可能包含 K8s 资源名称不允许的字符，需要对版本号进行 Slugify 处理或使用 Hash
	safeVersion := strings.ReplaceAll(v.Spec.Profile.Firmware.Version, "+", "-")
	cmdName := fmt.Sprintf("ota-%s-%s-%d", v.Name, safeVersion, v.Status.UpgradeStatus.RetryCount)

	var cmd iovv1alpha2.VehicleCommand
	if err := s.Get(ctx, types.NamespacedName{Namespace: v.Namespace, Name: cmdName}, &cmd); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		cmd = iovv1alpha2.VehicleCommand{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cmdName,
				Namespace: v.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(v, iovv1alpha2.GroupVersion.WithKind("Vehicle")),
				},
			},
			Spec: iovv1alpha2.VehicleCommandSpec{
				VehicleName: v.Name,
				Method:      "OTA", // TODO: VehicleModel
				Parameters: map[string]string{
					"version": v.Spec.Profile.Firmware.Version,
				},
			},
		}

		logger.Info("Creating new OTA Command", "command", cmdName, "targetVersion", v.Spec.Profile.Firmware.Version)
		SetCondition(v, iovv1alpha2.ConditionTypeSynced, metav1.ConditionFalse, "Updating", "Creating new OTA Command")
		return s.Create(ctx, &cmd)
	}

	switch cmd.Status.Phase {

	case iovv1alpha2.CommandPhaseSucceeded:
		return f.Event(ctx, EventSuccess, v)

	case iovv1alpha2.CommandPhaseFailed:
		return f.Event(ctx, EventFail, v, cmd.Status.Message)

	default:
		msg := fmt.Sprintf("Waiting for OTA command. Phase: %s, Message: %s", cmd.Status.Phase, cmd.Status.Message)
		SetCondition(v, iovv1alpha2.ConditionTypeSynced, metav1.ConditionFalse, "Updating", msg)
	}

	return nil
}
