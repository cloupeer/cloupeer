package hub

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"

	pb "cloupeer.io/cloupeer/api/proto/v1"
	"cloupeer.io/cloupeer/internal/pkg/mqtt/paths"
	"cloupeer.io/cloupeer/pkg/log"
	"cloupeer.io/cloupeer/pkg/mqtt"
	mqtttopic "cloupeer.io/cloupeer/pkg/mqtt/topic"
)

type GrpcServer struct {
	srv *grpc.Server
	lis net.Listener
}

func (cfg *Config) NewGrpcServer(mqttClient mqtt.Client, topicBuilder *mqtttopic.Builder) (*GrpcServer, error) {
	lis, err := net.Listen("tcp", cfg.GrpcOptions.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on grpc addr %s: %w", cfg.GrpcOptions.Addr, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterHubServiceServer(grpcServer, &grpcHandler{
		mqttclient:   mqttClient,
		topicbuilder: topicBuilder,
	})

	return &GrpcServer{srv: grpcServer, lis: lis}, nil
}

// grpcHandler implements pb.HubServiceServer
type grpcHandler struct {
	pb.UnimplementedHubServiceServer

	mqttclient   mqtt.Client
	topicbuilder *mqtttopic.Builder
}

// SendCommand implements the gRPC method defined in hub.proto
func (h *grpcHandler) SendCommand(ctx context.Context, req *pb.SendCommandRequest) (*pb.SendCommandResponse, error) {
	log.Info("Hub received gRPC Command",
		"vehicleID", req.VehicleId,
		"type", req.CommandType,
		"params", req.Parameters)

	// 1. 构造 Topic
	topic := h.topicbuilder.Build(paths.Command, req.VehicleId)

	// 2. 构造 Payload (使用生成的 PB 结构体)
	pbPayload := &pb.AgentCommand{
		CommandName: req.CommandName,
		CommandType: req.CommandType,
		Parameters:  req.Parameters,
		Timestamp:   time.Now().Unix(),
	}

	// 使用 protojson 进行序列化，它会遵循 proto 文件中的 [json_name] 定义
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   false, // 使用 camelCase (json_name)
		EmitUnpopulated: true,  // 输出空字段（可选）
	}

	payloadBytes, err := marshaler.Marshal(pbPayload)
	if err != nil {
		log.Error(err, "Failed to marshal agent command proto")
		return nil, err
	}

	// 3. 发布 MQTT 消息
	// 使用 QoS 1 (At Least Once) 确保送达
	err = h.mqttclient.Publish(ctx, topic, 1, false, payloadBytes)
	if err != nil {
		log.Error(err, "Failed to publish MQTT message", "topic", topic)
		return &pb.SendCommandResponse{
			Accepted: false,
			Message:  fmt.Sprintf("MQTT Publish Failed: %v", err),
		}, nil
	}

	log.Info("Command forwarded to MQTT", "topic", topic, "payloadSize", len(payloadBytes))

	return &pb.SendCommandResponse{
		Accepted: true,
		Message:  "Command forwarded to MQTT broker",
	}, nil
}
