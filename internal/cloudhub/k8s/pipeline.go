package k8s

import (
	"context"
	"encoding/json"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"cloupeer.io/cloupeer/internal/cloudhub/core/model"
	iovv1alpha2 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha2"
	"cloupeer.io/cloupeer/pkg/log"
)

// StatusPipeline implements a write-merging buffer for K8s status updates.
// It protects the K8s API server from being overwhelmed by high-frequency heartbeat events.
type StatusPipeline struct {
	namespace string
	client    client.Client

	// inputCh is the channel where high-velocity updates are pushed.
	inputCh chan *model.VehicleStatusUpdate

	// buffer stores the "latest" state for each vehicle ID.
	// Map Key: Vehicle ID
	buffer map[string]*model.VehicleStatusUpdate

	// flushInterval determines how often we flush the aggregated state to K8s.
	flushInterval time.Duration
}

// NewPipeline creates a new write-merging pipeline.
func NewPipeline(ns string, c client.Client) *StatusPipeline {
	return &StatusPipeline{
		namespace:     ns,
		client:        c,
		inputCh:       make(chan *model.VehicleStatusUpdate, 5000), // Large buffer
		buffer:        make(map[string]*model.VehicleStatusUpdate),
		flushInterval: 1 * time.Second, // Aggregate 1s worth of data
	}
}

// Start begins the background worker that processes the pipeline.
// This should be run in a goroutine.
func (p *StatusPipeline) Start(ctx context.Context) {
	ticker := time.NewTicker(p.flushInterval)
	defer ticker.Stop()

	log.Info("K8s Status Pipeline started", "Interval", p.flushInterval)

	for {
		select {
		case update := <-p.inputCh:
			// MERGE STRATEGY: Last Write Wins (in memory)
			// We only keep the latest update for each vehicle in the buffer map.
			p.buffer[update.VIN] = update

			// Optimization: If buffer gets too large, force flush immediately
			if len(p.buffer) >= 1000 {
				p.flush(ctx)
			}

		case <-ticker.C:
			// Time to sync to K8s
			if len(p.buffer) > 0 {
				p.flush(ctx)
			}

		case <-ctx.Done():
			// Flush remaining data before exit
			p.flush(context.Background())
			return
		}
	}
}

// Push adds an update to the pipeline. It is non-blocking.
func (p *StatusPipeline) Push(update *model.VehicleStatusUpdate) {
	select {
	case p.inputCh <- update:
		// Success
	default:
		// Buffer full: Drop the heartbeat to protect the system (Load Shedding).
		// For status updates, dropping a frame is better than crashing OOM.
		log.Warn("Status pipeline full! Dropping update for vehicle: %s", update.VIN)
	}
}

// flush sends all buffered updates to K8s.
// Note: K8s currently doesn't support bulk updates for different resources.
// We still have to make N requests, BUT we saved M (M >> N) redundant requests via merging.
func (p *StatusPipeline) flush(ctx context.Context) {
	count := 0
	for vin, update := range p.buffer {
		if err := p.patchStatus(ctx, vin, update); err != nil {
			log.Error(err, "Failed to patch vehicle status", "vin", vin)
		}
		count++
	}

	// Reset buffer after flush
	p.buffer = make(map[string]*model.VehicleStatusUpdate)

	log.Debug("Pipeline flushed %d updates to K8s", count)
}

// patchStatus performs a lightweight MergePatch on the Status subresource.
func (p *StatusPipeline) patchStatus(ctx context.Context, vin string, update *model.VehicleStatusUpdate) error {
	// Construct a raw JSON patch for efficiency.
	// We only want to touch specific fields in .status
	// structure: {"status": {"online": true, "lastSeenTime": "..."}}
	patchMap := map[string]any{
		"apiVersion": "iov.cloupeer.io/v1alpha2",
		"kind":       "Vehicle",
		"metadata": map[string]any{
			"name":      vinToMetaName(vin),
			"namespace": p.namespace,
		},
		"status": map[string]any{
			"online":            update.Online,
			"lastHeartbeatTime": update.LastHeartbeatTime, // 确保这里序列化符合 RFC3339
			// "reportedFirmwareVersion": update.FirmwareVersion,
		},
	}

	patchData, err := json.Marshal(patchMap)
	if err != nil {
		return err
	}

	// Use generic client to Patch.
	// Note: We use MergePatchType on the Status subresource.
	obj := &iovv1alpha2.Vehicle{}
	obj.SetName(vinToMetaName(vin))
	obj.SetNamespace(p.namespace)

	patch := client.RawPatch(types.ApplyPatchType, patchData)
	owner := client.FieldOwner("cpeer-cloudhub")
	return p.client.Status().Patch(ctx, obj, patch, owner, client.ForceOwnership)
}
