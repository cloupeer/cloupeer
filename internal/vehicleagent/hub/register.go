package hub

import (
	"fmt"

	"cloupeer.io/cloupeer/internal/pkg/mqtt/paths"
	"cloupeer.io/cloupeer/internal/vehicleagent/core"
)

var (
	events = make(map[core.EventType]string)
	routes = make(map[string]core.HandlerFunc)
)

func (b *Hub) Register(event core.EventType, handler core.HandlerFunc) error {
	segment, ok := events[event]
	if !ok {
		return fmt.Errorf("unmapped event: %s", event)
	}
	fullTopic := b.topics.Build(segment, b.vehicleID)
	routes[fullTopic] = handler
	return nil
}

func init() {
	events[core.EventOTACommand] = paths.Command
	events[core.EventOTARequest] = paths.OTARequest
	events[core.EventOTAResponse] = paths.OTAResponse
	events[core.EventCommandStatus] = paths.CommandAck
	events[core.EventRegister] = paths.Register
}
