package grpc

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/autopeer-io/autopeer/api/proto/v1"
	"github.com/autopeer-io/autopeer/internal/bridge/core/model"
	"github.com/autopeer-io/autopeer/internal/bridge/core/service"
	"github.com/autopeer-io/autopeer/pkg/log"
	"github.com/autopeer-io/autopeer/pkg/options"
)

type Server struct {
	server                           *grpc.Server
	svc                              *service.Service
	options                          *options.GrpcOptions
	pb.UnimplementedHubServiceServer // Embed for forward compatibility
}

func NewServer(opts *options.GrpcOptions, svc *service.Service) (*Server, error) {
	s := grpc.NewServer()
	srv := &Server{
		server:  s,
		svc:     svc,
		options: opts,
	}
	pb.RegisterHubServiceServer(s, srv)
	reflection.Register(s) // Enable grpc_cli support
	return srv, nil
}

func (s *Server) Start(ctx context.Context) error {
	lis, err := net.Listen(s.options.Network, s.options.Addr)
	if err != nil {
		return err
	}

	log.Info("Starting gRPC Server", "addr", s.options.Addr)

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.Serve(lis); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.server.GracefulStop()
		return nil
	}
}

// SendCommand implements v1.HubServiceServer.
// It receives a command from the Controller and dispatches it via MQTT.
func (s *Server) SendCommand(ctx context.Context, req *pb.SendCommandRequest) (*pb.SendCommandResponse, error) {
	log.Info("Received gRPC Command", "id", req.CommandName, "vehicle", req.VehicleId)

	cmd := &model.Command{
		ID:         req.CommandName,
		VehicleID:  req.VehicleId,
		Type:       model.CommandType(req.CommandType),
		Parameters: req.Parameters,
		Status:     model.CommandStatusPending,
	}

	// WARNING: You need to ensure DispatchCommand exists in core/service/command.go
	// Logic: return s.notifier.Notify(ctx, cmd)
	// If it doesn't exist yet, you can temporarily call s.svc.Notifier.Notify(ctx, cmd)
	// if you expose Notifier, but adding the method to Service is cleaner.
	err := s.svc.DispatchCommand(ctx, cmd)

	if err != nil {
		log.Error(err, "Failed to dispatch command", "id", req.CommandName)
		return &pb.SendCommandResponse{
			Accepted: false,
			Message:  err.Error(),
		}, nil
	}

	return &pb.SendCommandResponse{
		Accepted: true,
		Message:  "Command queued for delivery",
	}, nil
}

// SendCommand implements the gRPC method defined in hub.proto
// func (h *grpcHandler) SendCommand(ctx context.Context, req *pb.SendCommandRequest) (*pb.SendCommandResponse, error) {
// 	log.Info("Hub received gRPC Command",
// 		"vehicleID", req.VehicleId,
// 		"type", req.CommandType,
// 		"params", req.Parameters)

// 	// 1. 构造 Topic
// 	topic := h.topicbuilder.Build(paths.Command, req.VehicleId)

// 	// 2. 构造 Payload (使用生成的 PB 结构体)
// 	pbPayload := &pb.AgentCommand{
// 		CommandName: req.CommandName,
// 		CommandType: req.CommandType,
// 		Parameters:  req.Parameters,
// 		Timestamp:   time.Now().Unix(),
// 	}

// 	// 使用 protojson 进行序列化，它会遵循 proto 文件中的 [json_name] 定义
// 	marshaler := protojson.MarshalOptions{
// 		UseProtoNames:   false, // 使用 camelCase (json_name)
// 		EmitUnpopulated: true,  // 输出空字段（可选）
// 	}

// 	payloadBytes, err := marshaler.Marshal(pbPayload)
// 	if err != nil {
// 		log.Error(err, "Failed to marshal agent command proto")
// 		return nil, err
// 	}

// 	// 3. 发布 MQTT 消息
// 	// 使用 QoS 1 (At Least Once) 确保送达
// 	err = h.mqttclient.Publish(ctx, topic, 1, false, payloadBytes)
// 	if err != nil {
// 		log.Error(err, "Failed to publish MQTT message", "topic", topic)
// 		return &pb.SendCommandResponse{
// 			Accepted: false,
// 			Message:  fmt.Sprintf("MQTT Publish Failed: %v", err),
// 		}, nil
// 	}

// 	log.Info("Command forwarded to MQTT", "topic", topic, "payloadSize", len(payloadBytes))

// 	return &pb.SendCommandResponse{
// 		Accepted: true,
// 		Message:  "Command forwarded to MQTT broker",
// 	}, nil
// }
