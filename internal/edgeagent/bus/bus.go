package bus

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"cloupeer.io/cloupeer/internal/edgeagent/core"
	"cloupeer.io/cloupeer/pkg/log"
	"cloupeer.io/cloupeer/pkg/mqtt"
	mqtttopic "cloupeer.io/cloupeer/pkg/mqtt/topic"
)

type Bus struct {
	vehicleID string

	mc     mqtt.Client
	topics *mqtttopic.Builder
}

var _ core.Sender = (*Bus)(nil)

func New(client mqtt.Client, builder *mqtttopic.Builder, vid string) *Bus {
	return &Bus{
		mc:        client,
		topics:    builder,
		vehicleID: vid,
	}
}

func (b *Bus) Send(ctx context.Context, event core.EventType, payload []byte) error {
	segment, ok := events[event]
	if !ok {
		return fmt.Errorf("unmapped event: %s", event)
	}
	fullTopic := b.topics.Build(segment, b.vehicleID)
	return b.mc.Publish(ctx, fullTopic, 1, true, payload)
}

func (b *Bus) SendProto(ctx context.Context, event core.EventType, msg proto.Message) error {
	payload, err := protojson.Marshal(msg)
	if err != nil {
		return err
	}
	return b.Send(ctx, event, payload)
}

func (b *Bus) Start(ctx context.Context) error {
	if err := b.mc.Start(ctx); err != nil {
		return err
	}

	if err := b.mc.AwaitConnection(ctx); err != nil {
		return err
	}

	for topic, handler := range routes {
		err := b.mc.Subscribe(ctx, topic, 1, func(c context.Context, _ string, p []byte) {
			if handleErr := handler(c, p); handleErr != nil {
				log.Error(handleErr, "Handler execution failed", "topic", topic)
			}
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Bus) Stop() {
	log.Info("Disconnecting MQTT client...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	b.mc.Disconnect(ctx)
}
