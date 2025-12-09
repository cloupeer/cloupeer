package k8s

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"cloupeer.io/cloupeer/internal/cloudhub/core"
)

var _ core.Repository = (*repository)(nil)

// repository implements core.Repository using K8s CRDs.
type repository struct {
	namespace string
	client    client.Client
	pipeline  *StatusPipeline
}

func NewRepository(ns string, c client.Client, p *StatusPipeline) *repository {
	return &repository{
		namespace: ns,
		client:    c,
		pipeline:  p,
	}
}

func (r *repository) Vehicle() core.VehicleRepository {
	return newVehicleRepository(r.namespace, r.client, r.pipeline)
}

func (r *repository) Command() core.CommandRepository {
	return newCommandRepository(r.namespace, r.client)
}
