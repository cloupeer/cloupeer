package ota

import (
	"context"
	"sync"

	"cloupeer.io/cloupeer/internal/vehicleagent/core"
)

type Manager struct {
	vehicleID string

	sender core.Sender

	lock    sync.Mutex
	pending map[string]chan string
}

var _ core.Module = (*Manager)(nil)

func newManager(vid string) *Manager {
	return &Manager{
		vehicleID: vid,
		pending:   make(map[string]chan string),
	}
}

func Register(vid string) {
	core.Register(newManager(vid))
}

func (m *Manager) Name() string {
	return "OTA"
}

func (m *Manager) Setup(ctx context.Context, sender core.Sender) error {
	m.sender = sender
	return nil
}

func (m *Manager) Routes() map[core.EventType]core.HandlerFunc {
	return map[core.EventType]core.HandlerFunc{
		core.EventOTACommand:  core.ProtoAdapter(m.HandleCommand),
		core.EventOTAResponse: core.ProtoAdapter(m.HandleResponse),
	}
}
