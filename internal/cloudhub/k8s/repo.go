package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"cloupeer.io/cloupeer/internal/cloudhub/core"
	"cloupeer.io/cloupeer/internal/cloudhub/core/model"
	"cloupeer.io/cloupeer/internal/pkg/util"
	"cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
)

// Repository implements core.VehicleRepository using K8s CRDs.
type Repository struct {
	namespace string
	client    client.Client
	pipeline  *StatusPipeline
}

// Ensure Repository implements the interface
var _ core.VehicleRepository = &Repository{}

func NewRepository(ns string, c client.Client, p *StatusPipeline) *Repository {
	return &Repository{
		namespace: ns,
		client:    c,
		pipeline:  p,
	}
}

func (r *Repository) Get(ctx context.Context, id string) (*model.Vehicle, error) {
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

func (r *Repository) Create(ctx context.Context, v *model.Vehicle) error {
	crd := ToCRD(r.namespace, v)
	if err := r.client.Create(ctx, crd); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil // Idempotency
		}
		return err
	}

	return nil
}

func (r *Repository) UpdateStatus(ctx context.Context, v *model.Vehicle) error {
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
func (r *Repository) BatchUpdateStatus(ctx context.Context, update *model.VehicleStatusUpdate) error {
	if r.pipeline == nil {
		return errors.New("pipeline not initialized")
	}

	// Non-blocking push
	r.pipeline.Push(update)
	return nil
}

// UpdateStatus implements core.CommandRepository.
// It maps the model status to the K8s CRD status.
func (r *Repository) UpdateStatus2(ctx context.Context, cmdID string, status model.CommandStatus, message string) error {
	// In a real high-concurrency scenario, this should also use the Pipeline (Buffer).
	// For simplicity in this MVP, we use direct Patch, but leveraging Server-Side Apply or MergePatch.

	patchMap := map[string]interface{}{
		"status": map[string]interface{}{
			"phase":   status,
			"message": message,
		},
	}

	patchData, err := json.Marshal(patchMap)
	if err != nil {
		return err
	}

	obj := &v1alpha1.VehicleCommand{}
	obj.Name = cmdID
	obj.Namespace = "default" // TODO: Configurable namespace

	// Use MergePatchType to avoid conflicts
	patch := client.RawPatch(types.MergePatchType, patchData)
	return r.client.Status().Patch(ctx, obj, patch)
}
