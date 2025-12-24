package hub

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/autopeer-io/autopeer/internal/vehicleagent/core"
	"github.com/autopeer-io/autopeer/pkg/log"
	"github.com/autopeer-io/autopeer/pkg/mqtt"
	mqtttopic "github.com/autopeer-io/autopeer/pkg/mqtt/topic"
)

type Hub struct {
	vid string

	mc     mqtt.Client
	topics *mqtttopic.Builder
}

var _ core.Sender = (*Hub)(nil)

func New(vid string, client mqtt.Client, topicbuilder *mqtttopic.Builder) *Hub {
	return &Hub{
		mc:     client,
		topics: topicbuilder,
		vid:    vid,
	}
}

func (b *Hub) Send(ctx context.Context, event core.EventType, payload []byte) error {
	segment, ok := events[event]
	if !ok {
		return fmt.Errorf("unmapped event: %s", event)
	}
	fullTopic := b.topics.Build(segment, b.vid)
	return b.mc.Publish(ctx, fullTopic, 1, true, payload)
}

func (b *Hub) SendProto(ctx context.Context, event core.EventType, msg proto.Message) error {
	payload, err := protojson.Marshal(msg)
	if err != nil {
		return err
	}
	return b.Send(ctx, event, payload)
}

func (b *Hub) IsConnected() bool {
	return b.mc.IsConnected()
}

func (b *Hub) Start(ctx context.Context) error {
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

func (b *Hub) Stop() {
	log.Info("Disconnecting MQTT client...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	b.mc.Disconnect(ctx)
}
