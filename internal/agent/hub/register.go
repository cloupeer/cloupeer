package hub

import (
	"fmt"

	"github.com/autopeer-io/autopeer/internal/agent/core"
	"github.com/autopeer-io/autopeer/internal/pkg/mqtt/adapter"
	"github.com/autopeer-io/autopeer/internal/pkg/mqtt/paths"
)

var (
	events = make(map[core.EventType]string)
	routes = make(map[string]adapter.HandlerFunc)
)

func (b *Hub) Register(event core.EventType, handler adapter.HandlerFunc) error {
	segment, ok := events[event]
	if !ok {
		return fmt.Errorf("unmapped event: %s", event)
	}
	fullTopic := b.topics.Build(segment, b.vid)
	routes[fullTopic] = handler
	return nil
}

func init() {
	events[core.EventRegister] = paths.Register
	events[core.EventOnline] = paths.Online
	events[core.EventOTACommand] = paths.Command
	events[core.EventOTARequest] = paths.OTARequest
	events[core.EventOTAResponse] = paths.OTAResponse
	events[core.EventCommandStatus] = paths.CommandAck
}
