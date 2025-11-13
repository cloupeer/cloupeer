package vehicle

import (
	"errors"

	"github.com/looplab/fsm"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	iovv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
)

func isNewVersion(v *iovv1alpha1.Vehicle) bool {
	return v.Spec.FirmwareVersion != "" && v.Spec.FirmwareVersion != v.Status.ReportedFirmwareVersion
}

func isFsmRealError(err error) bool {
	if err == nil {
		return false
	}

	var noTransition fsm.NoTransitionError
	var canceled fsm.CanceledError

	if errors.As(err, &noTransition) || errors.As(err, &canceled) {
		return false
	}

	return true
}

// --- K8s Condition Helpers ---

// SetCondition 辅助函数，用于设置 Vehicle 的 Condition
func SetCondition(v *iovv1alpha1.Vehicle, conditionType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&v.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: v.Generation,
		LastTransitionTime: metav1.Now(),
	})
}

func FindLatestCondition(conditions []metav1.Condition) *metav1.Condition {
	var latestCond *metav1.Condition

	for i := range conditions {
		if conditions[i].Type == iovv1alpha1.ConditionTypeReady {
			continue
		}

		if latestCond == nil || !conditions[i].LastTransitionTime.Before(&latestCond.LastTransitionTime) {
			latestCond = &conditions[i]
		}
	}

	if latestCond != nil {
		return latestCond.DeepCopy()
	}

	return nil
}
