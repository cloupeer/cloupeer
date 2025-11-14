package main

import (
	"context"
	"log"
	"strings"
)

type SubFinalizer struct{}

func NewSubFinalizer() *SubFinalizer {
	return &SubFinalizer{}
}

func (s *SubFinalizer) Reconcile(ctx context.Context, vehicle *Vehicle) (ReconcileResult, error) {

	if !strings.HasSuffix(vehicle.Name, "_finalizer") {
		log.Println("Adding Finalizer to Vehicle.")
		vehicle.Name += "_finalizer"

		return ReconcileResult{}, nil
	}

	// Finalizer exists, continue to the next step.
	return ReconcileResult{}, nil
}

func init() {
	Register(KeySubFinalizer, NewSubFinalizer())
}
