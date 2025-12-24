package k8s

import (
	"context"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/autopeer-io/autopeer/internal/bridge/core/model"
	"github.com/autopeer-io/autopeer/internal/pkg/util"
	iovv1alpha2 "github.com/autopeer-io/autopeer/pkg/apis/iov/v1alpha2"
)

type vehicleRepository struct {
	namespace string
	client    client.Client
	pipeline  *StatusPipeline
}

func newVehicleRepository(ns string, c client.Client, p *StatusPipeline) *vehicleRepository {
	return &vehicleRepository{namespace: ns, client: c, pipeline: p}
}

func (r *vehicleRepository) Get(ctx context.Context, vin string) (*model.Vehicle, error) {
	crd := &iovv1alpha2.Vehicle{}
	key := types.NamespacedName{Name: vinToMetaName(vin), Namespace: r.namespace}

	if err := r.client.Get(ctx, key, crd); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, util.ErrNotFound // Return nil for not found as per Go conventions (or custom ErrNotFound)
		}
		return nil, err
	}

	return ToModel(crd), nil
}

func (r *vehicleRepository) Create(ctx context.Context, v *model.Vehicle) error {
	crd := ToCRD(r.namespace, v)
	if err := r.client.Create(ctx, crd); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil // Idempotency
		}
		return err
	}

	return nil
}

func (r *vehicleRepository) UpdateStatus(ctx context.Context, v *model.Vehicle) error {
	crd := &iovv1alpha2.Vehicle{}
	key := types.NamespacedName{Name: vinToMetaName(v.VIN), Namespace: r.namespace}
	if err := r.client.Get(ctx, key, crd); err != nil {
		return fmt.Errorf("failed to get vehicle for status update: %w", err)
	}

	now := metav1.Now()
	crd.Status.Online = v.Online
	crd.Status.Profile.Firmware.Version = v.ReportedVersion
	if !v.LastHeartbeatTime.IsZero() {
		crd.Status.LastHeartbeatTime = &now
	}

	if err := r.client.Status().Update(ctx, crd); err != nil {
		return fmt.Errorf("failed to init status: %w", err)
	}
	return nil
}

// UpdateStatus delegates the update to the async pipeline.
// This returns immediately, ensuring high throughput for the caller.
func (r *vehicleRepository) BatchUpdateStatus(ctx context.Context, update *model.VehicleStatusUpdate) error {
	if r.pipeline == nil {
		return errors.New("pipeline not initialized")
	}

	// Non-blocking push
	r.pipeline.Push(update)
	return nil
}
