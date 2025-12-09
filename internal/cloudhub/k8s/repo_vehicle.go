package k8s

import (
	"context"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"cloupeer.io/cloupeer/internal/cloudhub/core/model"
	"cloupeer.io/cloupeer/internal/pkg/util"
	"cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
)

type vehicleRepository struct {
	namespace string
	client    client.Client
	pipeline  *StatusPipeline
}

func newVehicleRepository(ns string, c client.Client, p *StatusPipeline) *vehicleRepository {
	return &vehicleRepository{namespace: ns, client: c, pipeline: p}
}

func (r *vehicleRepository) Get(ctx context.Context, id string) (*model.Vehicle, error) {
	crd := &v1alpha1.Vehicle{}
	key := types.NamespacedName{Name: id, Namespace: "default"}

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
	crd := &v1alpha1.Vehicle{}
	key := types.NamespacedName{Name: v.ID, Namespace: r.namespace}
	if err := r.client.Get(ctx, key, crd); err != nil {
		return fmt.Errorf("failed to get vehicle for status update: %w", err)
	}

	now := metav1.Now()
	crd.Status.Online = v.Online
	crd.Status.ReportedFirmwareVersion = v.FirmwareVersion
	if !v.LastSeen.IsZero() {
		crd.Status.LastSeenTime = &now // 简单起见用当前时间，或转换 v.LastSeen
	}

	if v.IsRegister {
		crd.Status.Phase = v1alpha1.VehiclePhaseIdle
		crd.Status.Conditions = []metav1.Condition{
			{
				Type:               v1alpha1.ConditionTypeReady,
				Status:             metav1.ConditionTrue,
				Reason:             "AutoRegistered", // 翻译结果
				Message:            "Vehicle initialized via CloudHub registration",
				LastTransitionTime: now,
			},
			{
				Type:               v1alpha1.ConditionTypeSynced,
				Status:             metav1.ConditionTrue,
				Reason:             "InitialSync",
				Message:            fmt.Sprintf("Reported version: %s", v.FirmwareVersion),
				LastTransitionTime: now,
			},
		}
	}

	// 4. 执行更新
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
