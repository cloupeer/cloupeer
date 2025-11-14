package main

import (
	"context"
	"log"
)

type SubStateMachine struct{}

func NewSubStateMachine() *SubStateMachine {
	return &SubStateMachine{}
}

func (s *SubStateMachine) Reconcile(ctx context.Context, vehicle *Vehicle) (ReconcileResult, error) {

	log.Println("The FSM logic is running...", "name", vehicle.Name)

	// Return the Result from the FSM logic.
	return ReconcileResult{}, nil
}

func init() {
	Register(KeySubStateMachine, NewSubStateMachine())
}
