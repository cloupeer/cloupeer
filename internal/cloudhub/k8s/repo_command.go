package k8s

import (
	"context"
	"encoding/json"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/autopeer-io/autopeer/internal/cloudhub/core/model"
	iovv1alpha2 "github.com/autopeer-io/autopeer/pkg/apis/iov/v1alpha2"
)

type commandRepository struct {
	namespace string
	client    client.Client
}

func newCommandRepository(ns string, c client.Client) *commandRepository {
	return &commandRepository{namespace: ns, client: c}
}

// UpdateStatus implements core.CommandRepository.
// It maps the model status to the K8s CRD status.
func (r *commandRepository) UpdateStatus(ctx context.Context, cmdID string, status model.CommandStatus, message string) error {
	// In a real high-concurrency scenario, this should also use the Pipeline (Buffer).
	// For simplicity in this MVP, we use direct Patch, but leveraging Server-Side Apply or MergePatch.

	patchMap := map[string]any{
		"status": map[string]any{
			"phase":   status,
			"message": message,

			// "lastUpdateTime": "",
			// TODO: AcknowledgeTime, CompletionTime
		},
	}

	patchData, err := json.Marshal(patchMap)
	if err != nil {
		return err
	}

	obj := &iovv1alpha2.VehicleCommand{}
	obj.SetName(cmdID)
	obj.SetNamespace(r.namespace)

	// Use MergePatchType to avoid conflicts
	patch := client.RawPatch(types.MergePatchType, patchData)
	return r.client.Status().Patch(ctx, obj, patch)
}
