package mqtt

import (
	"context"
	"fmt"
	"time"

	pb "cloupeer.io/cloupeer/api/proto/v1"
	"cloupeer.io/cloupeer/internal/cloudhub/core/model"
	"cloupeer.io/cloupeer/internal/cloudhub/core/service"
	"cloupeer.io/cloupeer/internal/pkg/mqtt/paths"
	"cloupeer.io/cloupeer/pkg/log"
	pkgmqtt "cloupeer.io/cloupeer/pkg/mqtt"
	"cloupeer.io/cloupeer/pkg/mqtt/topic"
	"cloupeer.io/cloupeer/pkg/options"
)

// Server implements the MQTT ingress layer.
type Server struct {
	client  pkgmqtt.Client
	svc     *service.Service
	options *options.MqttOptions
}

// NewServer creates a new MQTT server (client).
func NewServer(opts *options.MqttOptions, svc *service.Service) (*Server, error) {
	// Use the shared MQTT client factory from pkg/mqtt
	client, err := pkgmqtt.NewClient(opts.ToClientConfig())
	if err != nil {
		return nil, err
	}

	return &Server{
		client:  client,
		svc:     svc,
		options: opts,
	}, nil
}

// Start connects to the broker and subscribes to topics.
func (s *Server) Start(ctx context.Context) error {
	log.Info("Starting MQTT Server", "broker", s.options.Broker)

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

	if err := s.setupMQTTSubscriptions(ctx); err != nil {
		return err
	}

	<-ctx.Done()

	return nil
}

func (s *Server) setupMQTTSubscriptions(ctx context.Context) error {
	// Define shared subscription group prefix
	const groupName = "cpeer-hub"
	const qos = 1

	builder := topic.NewBuilder(s.options.TopicRoot)

	subscriptions := map[string]HandlerFunc{
		paths.Register:   ProtoAdapter(s.handleRegister),
		paths.Online:     ProtoAdapter(s.handleOnline),
		paths.CommandAck: ProtoAdapter(s.handleCommandAck),
		paths.OTARequest: ProtoAdapter(s.handleOTARequest),
	}

	for segment, handler := range subscriptions {
		fullTopic := builder.Shared(groupName).BuildWildcard(segment)
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

func (s *Server) handleRegister(ctx context.Context, req *pb.RegisterVehicleRequest) error {
	if req.VehicleId == "" {
		log.Warn("Received register request without vehicleID")
		return nil
	}

	log.Info("Received register request", "vehicleID", req.VehicleId, "ver", req.FirmwareVersion)

	v := &model.Vehicle{
		ID:              req.VehicleId,
		FirmwareVersion: req.FirmwareVersion,
		Description:     req.Description,
		IsRegister:      true,
	}

	if err := s.svc.RegisterVehicle(ctx, v); err != nil {
		log.Error(err, "Failed to register vehicle", "id", v.ID)
	} else {
		log.Info("Vehicle registered successfully", "id", v.ID)
	}

	return nil
}

func (s *Server) handleOnline(ctx context.Context, req *pb.OnlineStatus) error {
	if err := s.svc.UpdateOnlineStatus(ctx, req.VehicleId, req.Online); err != nil {
		log.Error(err, "Failed to update online status", "id", req.VehicleId, "online", req.Online)
	}

	return nil
}

func (s *Server) handleCommandAck(ctx context.Context, req *pb.AgentCommandStatus) error {
	log.Info("Received Status Report",
		"commandName", req.CommandName,
		"status", req.Status,
		"msg", req.Message)

	return nil
}

func (s *Server) handleOTARequest(ctx context.Context, req *pb.OTARequest) error {
	// 如果关键字段为空，说明可能解析错了消息类型
	if req.VehicleId == "" || req.RequestId == "" {
		return fmt.Errorf("either VehicleId[%s] or RequestId[%s] is empty", req.VehicleId, req.RequestId)
	}

	log.Info("Received Firmware URL Request", "vehicleID", req.VehicleId, "ver", req.DesiredVersion)
	return nil
}
