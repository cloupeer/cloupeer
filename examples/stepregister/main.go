package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func main() {
	fmt.Println("main start...")
	vh001 := NewReconciler("vh001")

	for {
		vh001.Reconcile(context.Background())
		time.Sleep(5 * time.Second)
		fmt.Println()
	}
}

type Vehicle struct {
	Name  string
	Value int64
}

type VehicleReconciler struct {
	vehicle  Vehicle
	registry IRegistryGetter
}

func NewReconciler(name string, registry ...IRegistryGetter) *VehicleReconciler {
	v := &VehicleReconciler{
		vehicle:  Vehicle{Name: name},
		registry: GetDefaultRegistry(),
	}

	if len(registry) > 0 {
		v.registry = registry[0]
	}

	return v
}

func (r *VehicleReconciler) Reconcile(ctx context.Context) error {
	vehicle := &r.vehicle
	log.Println("Reconcile...", "name", vehicle.Name, "value", vehicle.Value)

	// Fetch Vehicle

	// Deepcopy

	// Deletion
	if vehicle.Value != 0 {
		//
		return nil
	}

	for _, key := range ListSubReconcilerOrderKey {
		sub, ok := r.registry.Get(key)
		if !ok {
			// This is a developer error (forgot to register)
			return fmt.Errorf("pipeline step %d not registered", key)
		}

		_, err := sub.Reconcile(ctx, vehicle)
		if err != nil {
			return err
		}
	}

	// some logic

	return nil
}
