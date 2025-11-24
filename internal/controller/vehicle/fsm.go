package vehicle

import (
	"context"
	"fmt"

	"github.com/looplab/fsm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fsmutil "cloupeer.io/cloupeer/internal/pkg/util/fsm"
	iovv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
)

const (
	// EventUpdate (Active) checks if an update is required.
	EventUpdate = "event_update"
	// EventSuccess
	EventSuccess = "event_success"
	// EventFail
	EventFail = "event_fail"
	// EventRetry
	EventRetry = "event_retry"
	// EventFinalize (Active) cleans up a Succeeded state back to Idle.
	EventFinalize = "event_finalize"
)

type FiniteStateMachine struct {
	*fsm.FSM
}

func NewFiniteStateMachine(initialstate string) *FiniteStateMachine {
	f := &FiniteStateMachine{}

	events := fsm.Events{
		{Name: EventUpdate, Src: []string{string(iovv1alpha1.VehiclePhaseIdle)}, Dst: string(iovv1alpha1.VehiclePhasePending)},
		{Name: EventSuccess, Src: []string{string(iovv1alpha1.VehiclePhasePending)}, Dst: string(iovv1alpha1.VehiclePhaseSucceeded)},
		{Name: EventFail, Src: []string{string(iovv1alpha1.VehiclePhasePending)}, Dst: string(iovv1alpha1.VehiclePhaseFailed)},
		{Name: EventFinalize, Src: []string{string(iovv1alpha1.VehiclePhaseSucceeded)}, Dst: string(iovv1alpha1.VehiclePhaseIdle)},

		// Failed retry
		{Name: EventRetry, Src: []string{string(iovv1alpha1.VehiclePhaseFailed)}, Dst: string(iovv1alpha1.VehiclePhasePending)},
	}

	callbacks := fsm.Callbacks{
		// Guards (before_...): Decide if a transition is allowed
		"before_" + EventUpdate: fsmutil.WrapEvent(f.GuardUpdateRequired),

		// Side-Effects (enter_...): Set fields upon entering a state
		"enter_" + string(iovv1alpha1.VehiclePhasePending):   fsmutil.WrapEvent(f.ActionEnterPending),
		"enter_" + string(iovv1alpha1.VehiclePhaseSucceeded): fsmutil.WrapEvent(f.ActionEnterSucceeded),
		"enter_" + string(iovv1alpha1.VehiclePhaseFailed):    fsmutil.WrapEvent(f.ActionEnterFailed),
		"enter_" + string(iovv1alpha1.VehiclePhaseIdle):      fsmutil.WrapEvent(f.ActionEnterIdle),
	}

	f.FSM = fsm.NewFSM(initialstate, events, callbacks)
	return f
}

// GuardUpdateRequired is a "Guard" callback.
// It checks if an update is needed and cancels the transition if not.
func (f *FiniteStateMachine) GuardUpdateRequired(ctx context.Context, e *fsm.Event) error {
	v := e.Args[0].(*iovv1alpha1.Vehicle)
	if !(isNewVersion(v)) {
		// No update needed. Cancel the transition.
		e.Cancel(fsm.NoTransitionError{})
	}
	return nil
}

// ActionEnterPending is a "Side-Effect" callback.
// It resets the status for a new update attempt (either new or retry).
func (f *FiniteStateMachine) ActionEnterPending(ctx context.Context, e *fsm.Event) error {
	v := e.Args[0].(*iovv1alpha1.Vehicle)

	switch e.Event {
	case EventUpdate:
		// This is a NEW update (from Idle)
		v.Status.RetryCount = 0
	case EventRetry:
		// This is a RETRY (from Failed)
		v.Status.RetryCount++
	}

	// Reset status fields (Conditions, ErrorMessage) to prepare for a new update cycle.
	v.Status.Conditions = []metav1.Condition{}
	SetCondition(v, iovv1alpha1.ConditionTypeReady, metav1.ConditionFalse, "Pending", "Update process started")
	return nil
}

// ActionEnterSucceeded is a "Side-Effect" callback.
func (f *FiniteStateMachine) ActionEnterSucceeded(ctx context.Context, e *fsm.Event) error {
	v := e.Args[0].(*iovv1alpha1.Vehicle)

	v.Status.ReportedFirmwareVersion = v.Spec.FirmwareVersion
	SetCondition(v, iovv1alpha1.ConditionTypeReady, metav1.ConditionTrue, "Succeeded", "Firmware update applied successfully")
	SetCondition(v, iovv1alpha1.ConditionTypeSynced, metav1.ConditionTrue, "Synced", fmt.Sprintf("Version %s is active", v.Spec.FirmwareVersion))
	return nil
}

// ActionEnterFailed is a "Side-Effect" callback.
func (f *FiniteStateMachine) ActionEnterFailed(ctx context.Context, e *fsm.Event) error {
	v := e.Args[0].(*iovv1alpha1.Vehicle)
	errMsg := "unknown error"
	if len(e.Args) > 1 && e.Args[1] != nil {
		if err, ok := e.Args[1].(error); ok {
			errMsg = err.Error()
		} else if s, ok := e.Args[1].(string); ok {
			errMsg = s
		}
	}
	// We embed the spec version in the error message for the Reconcile loop's retry logic.
	msg := fmt.Sprintf("Failed on version %s: %s", v.Spec.FirmwareVersion, errMsg)
	SetCondition(v, iovv1alpha1.ConditionTypeReady, metav1.ConditionFalse, "Failed", msg)
	SetCondition(v, iovv1alpha1.ConditionTypeSynced, metav1.ConditionFalse, "SyncFailed", msg)
	return nil
}

// ActionEnterIdle is a "Side-Effect" callback.
func (f *FiniteStateMachine) ActionEnterIdle(ctx context.Context, e *fsm.Event) error {
	v := e.Args[0].(*iovv1alpha1.Vehicle)
	SetCondition(v, iovv1alpha1.ConditionTypeReady, metav1.ConditionTrue, "Idle", "Vehicle is ready for new commands")
	return nil
}
