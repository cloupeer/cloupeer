package main

import (
	"context"
	"log"
)

type SubHealthCheck struct{}

func NewSubHealthCheck() *SubHealthCheck {
	return &SubHealthCheck{}
}

func (h *SubHealthCheck) Reconcile(ctx context.Context, vehicle *Vehicle) (ReconcileResult, error) {

	log.Println("Health check...", "name", vehicle.Name)

	return ReconcileResult{}, nil
}

func init() {
	Register(KeySubHealthCheck, NewSubHealthCheck())
}
