package main

import (
	"context"
	"strconv"
)

type SubReconcilerOrderKey int

const (
	KeySubFinalizer SubReconcilerOrderKey = iota
	KeySubStateMachine
	KeySubHealthCheck
)

var ListSubReconcilerOrderKey = []SubReconcilerOrderKey{
	KeySubFinalizer,
	KeySubStateMachine,
	KeySubHealthCheck,
}

type ReconcileResult struct{}

type ISubReconciler interface {
	Reconcile(ctx context.Context, vehicle *Vehicle) (ReconcileResult, error)
}

type IRegistryGetter interface {
	Get(key SubReconcilerOrderKey) (ISubReconciler, bool)
}

type Registry struct {
	items map[SubReconcilerOrderKey]ISubReconciler
}

var defaultRegistry = &Registry{items: make(map[SubReconcilerOrderKey]ISubReconciler)}

func Register(key SubReconcilerOrderKey, sub ISubReconciler) {
	if _, ok := defaultRegistry.items[key]; ok {
		panic("duplicate sub reconciler: " + strconv.Itoa(int(key)))
	}

	defaultRegistry.items[key] = sub
}

func GetDefaultRegistry() IRegistryGetter {
	return defaultRegistry
}

func (r *Registry) Get(key SubReconcilerOrderKey) (ISubReconciler, bool) {
	sub, ok := r.items[key]
	return sub, ok
}
