package ota

import (
	"context"
	"sync"

	"github.com/autopeer-io/autopeer/internal/agent/core"
	"github.com/autopeer-io/autopeer/internal/pkg/mqtt/adapter"
)

type Manager struct {
	vid string

	hal    core.HAL
	sender core.Sender

	lock    sync.Mutex
	pending map[string]chan string
}

var _ core.Module = (*Manager)(nil)

func NewManager(vid string) *Manager {
	return &Manager{
		vid:     vid,
		pending: make(map[string]chan string),
	}
}

func (m *Manager) Name() string {
	return "OTA"
}

func (m *Manager) Setup(ctx context.Context, hal core.HAL, sender core.Sender) error {
	m.hal = hal
	m.sender = sender
	return nil
}

func (m *Manager) Routes() map[core.EventType]adapter.HandlerFunc {
	return map[core.EventType]adapter.HandlerFunc{
		core.EventOTACommand:  adapter.ProtoHandler(m.HandleCommand),
		core.EventOTAResponse: adapter.ProtoHandler(m.HandleResponse),
	}
}
