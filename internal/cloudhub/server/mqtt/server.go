package mqtt

import (
	"context"
	"fmt"
	"time"

	"cloupeer.io/cloupeer/internal/cloudhub/core/service"
	"cloupeer.io/cloupeer/internal/pkg/mqtt/adapter"
	"cloupeer.io/cloupeer/internal/pkg/mqtt/paths"
	"cloupeer.io/cloupeer/pkg/log"
	pkgmqtt "cloupeer.io/cloupeer/pkg/mqtt"
	"cloupeer.io/cloupeer/pkg/mqtt/topic"
)

// Server implements the MQTT ingress layer.
type Server struct {
	client pkgmqtt.Client
	topics *topic.Builder
	svc    *service.Service
}

// NewServer creates a new MQTT server (client).
func NewServer(client pkgmqtt.Client, builder *topic.Builder, svc *service.Service) *Server {
	return &Server{
		client: client,
		topics: builder,
		svc:    svc,
	}
}

// Start connects to the broker and subscribes to topics.
func (s *Server) Start(ctx context.Context) error {
	// 1. Start the connection manager (Non-blocking)
	if err := s.client.Start(ctx); err != nil {
		return err
	}

	// Ensure MQTT disconnects when Run exits (LIFO order)
	defer func() {
		log.Info("Disconnecting MQTT client...")
		// Use a fresh context with timeout to ensure disconnect packet sends
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.client.Disconnect(shutdownCtx)
		log.Info("MQTT client disconnected")
	}()

	// 2. Wait for the initial connection to be established
	// This ensures we don't start serving traffic until we are actually connected.
	// pkgmqtt.AwaitConnection handles the timeout internally or via ctx.
	log.Info("Waiting for MQTT connection...")
	if err := s.client.AwaitConnection(ctx); err != nil {
		return err
	}
	log.Info("MQTT Connected")

	if err := s.initMQTTSubscriptions(ctx); err != nil {
		return err
	}

	<-ctx.Done()

	return nil
}

func (s *Server) initMQTTSubscriptions(ctx context.Context) error {
	// Define shared subscription group prefix
	const groupName = paths.GroupCloudHub
	const qos = 1

	subscriptions := map[string]adapter.HandlerFunc{
		paths.Register:   adapter.ProtoHandler(s.handleRegister),
		paths.Online:     adapter.ProtoHandler(s.handleOnline),
		paths.CommandAck: adapter.ProtoHandler(s.handleCommandAck),
		paths.OTARequest: adapter.ProtoHandler(s.handleOTARequest),
	}

	for segment, handler := range subscriptions {
		fullTopic := s.topics.Shared(groupName).BuildWildcard(segment)
		if err := s.client.Subscribe(ctx, fullTopic, qos, func(c context.Context, _ string, p []byte) {
			if handleErr := handler(c, p); handleErr != nil {
				log.Error(handleErr, "Handler execution failed", "topic", fullTopic)
			}
		}); err != nil {
			return fmt.Errorf("failed to subscribe to topic: %s, err: %w", fullTopic, err)
		}
	}

	return nil
}
