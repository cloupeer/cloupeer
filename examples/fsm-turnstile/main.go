// This file is a standalone example for looplab/fsm.
// It is not part of the Cloupeer project itself.
package main

import (
	"context"
	"fmt"

	"github.com/looplab/fsm"
)

// --- 1. Define States and Events as constants ---
// This is not required by the library, but is a best practice.
const (
	StateLocked   = "locked"
	StateUnlocked = "unlocked"

	EventCoin = "coin"
	EventPush = "push"
)

// Turnstile represents our business object.
// Notice it does NOT hold the state. The FSM instance will hold it.
// In a real K8s app, the 'Reconciler' would create this FSM
// based on the CRD's 'status.phase'.
type Turnstile struct {
	FSM *fsm.FSM
}

// NewTurnstile creates a new turnstile and initializes its FSM
func NewTurnstile() *Turnstile {
	t := &Turnstile{}

	// Define all Callbacks (Actions and Guards)
	callbacks := fsm.Callbacks{
		// --- Guard ---
		// before_push is called *before* the 'push' event.
		// We use this to implement a Guard.
		"before_push": func(ctx context.Context, e *fsm.Event) {
			if e.FSM.Is(StateLocked) {
				fmt.Println("[Guard] Denied: cannot push a locked gate.")
				// e.Cancel() prevents the transition
				e.Cancel(fsm.InvalidEventError{Event: e.Event, State: e.Src})
			}
		},

		// --- Action ---
		// 'coin' callback is triggered *after* the 'coin' event.
		// It receives arguments passed from fsm.Event()
		"coin": func(ctx context.Context, e *fsm.Event) {
			if len(e.Args) > 0 {
				coinType, ok := e.Args[0].(string)
				if ok {
					fmt.Printf("[Action] Received coin of type: %s\n", coinType)
				}
			}
		},

		// --- Action ---
		// 'enter_unlocked' is called *after* entering the 'unlocked' state.
		"enter_unlocked": func(ctx context.Context, e *fsm.Event) {
			fmt.Println("[Action] Gate is now unlocked! Please proceed.")
		},

		// --- Action ---
		// 'enter_locked' is called *after* entering the 'locked' state.
		"enter_locked": func(ctx context.Context, e *fsm.Event) {
			fmt.Println("[Action] Gate is now locked.")
		},
	}

	// Define all Events (Transitions)
	events := fsm.Events{
		// {Name: <EVENT>, Src: []string{<FROM_STATE>}, Dst: <TO_STATE>}
		{Name: EventCoin, Src: []string{StateLocked}, Dst: StateUnlocked},
		{Name: EventPush, Src: []string{StateUnlocked}, Dst: StateLocked},
		// This transition is intentionally blocked by our Guard
		{Name: EventPush, Src: []string{StateLocked}, Dst: StateLocked},
	}

	// Create the new FSM instance
	t.FSM = fsm.NewFSM(
		StateLocked, // Initial state
		events,
		callbacks,
	)

	return t
}

func main() {
	// 1. Create our object
	gate := NewTurnstile()
	ctx := context.Background()

	fmt.Printf("[FSM] Initial state: %s\n", gate.FSM.Current())
	fmt.Println("----------------------------------------------------")

	// 2. Scenario 1: Try to push a locked gate (Guard fails)
	fmt.Println("[Event] ==> Pushing gate...")
	// We call fsm.Event() to trigger the 'push' event
	err := gate.FSM.Event(ctx, EventPush)
	if err != nil {
		fmt.Printf("[FSM] State unchanged: %s. Event failed: %v\n", gate.FSM.Current(), err)
	}
	fmt.Println("----------------------------------------------------")

	// 3. Scenario 2: Insert a coin (Action with args)
	fmt.Println("[Event] ==> Inserting a \"Quarter\"...")
	// Pass "Quarter" as an argument to the callback
	err = gate.FSM.Event(ctx, EventCoin, "Quarter")
	if err != nil {
		fmt.Printf("[FSM] Event failed: %v\n", err)
	}
	// We must *manually* check the state using .Current()
	fmt.Printf("[FSM] State changed: %s\n", gate.FSM.Current())
	fmt.Println("----------------------------------------------------")

	// 4. Scenario 3: Try to insert another coin (InvalidEvent)
	fmt.Println("[Event] ==> Inserting a \"Token\"...")
	err = gate.FSM.Event(ctx, EventCoin, "Token")
	if err != nil {
		// This fails because there is no 'coin' event with Src: 'unlocked'
		fmt.Printf("[FSM] State unchanged: %s. Event failed: %v\n", gate.FSM.Current(), err)
	}
	fmt.Println("----------------------------------------------------")

	// 5. Scenario 4: Push an unlocked gate (Success)
	fmt.Println("[Event] ==> Pushing gate...")
	err = gate.FSM.Event(ctx, EventPush)
	if err != nil {
		fmt.Printf("[FSM] Event failed: %v\n", err)
	}
	fmt.Printf("[FSM] State changed: %s\n", gate.FSM.Current())
}
