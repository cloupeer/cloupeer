package hub

import (
	"context"
	"fmt"
	"time"

	"github.com/eclipse/paho.golang/paho"
	"google.golang.org/protobuf/encoding/protojson"

	pb "cloupeer.io/cloupeer/api/proto/v1"
	"cloupeer.io/cloupeer/pkg/log"
)

// grpcHandler implements pb.HubServiceServer
type grpcHandler struct {
	pb.UnimplementedHubServiceServer
	parent *HubServer
}

// SendCommand implements the gRPC method defined in hub.proto
func (h *grpcHandler) SendCommand(ctx context.Context, req *pb.SendCommandRequest) (*pb.SendCommandResponse, error) {
	log.Info("Hub received gRPC Command",
		"vehicleID", req.VehicleId,
		"type", req.CommandType,
		"params", req.Parameters)

	// 1. 构造 Topic: iov/cmd/{vehicleID}
	topic := fmt.Sprintf("%s/%s", h.parent.mqttTopicPrefix, req.VehicleId)

	// 2. 构造 Payload (使用生成的 PB 结构体)
	// CommandID 暂时由 Hub 生成一个简单的 Trace ID
	cmdID := fmt.Sprintf("cmd-%s-%s-%d", req.VehicleId, req.CommandType, time.Now().UnixNano())

	pbPayload := &pb.AgentCommand{
		CommandId:   cmdID,
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
	_, err = h.parent.mqttMgr.Publish(ctx, &paho.Publish{
		Topic:   topic,
		QoS:     1,
		Payload: payloadBytes,
	})

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
