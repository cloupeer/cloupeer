package hub

import (
	"context"

	pb "cloupeer.io/cloupeer/api/proto/v1"
	"cloupeer.io/cloupeer/pkg/log"
)

// grpcHandler implements pb.HubServiceServer
type grpcHandler struct {
	pb.UnimplementedHubServiceServer
	// parent *HubServer // 如果需要访问 HubServer 的其他资源（如 K8s client），可以持有引用
}

// SendCommand implements the gRPC method defined in hub.proto
func (h *grpcHandler) SendCommand(ctx context.Context, req *pb.SendCommandRequest) (*pb.SendCommandResponse, error) {
	log.Info("Hub received gRPC Command",
		"vehicleID", req.VehicleId,
		"type", req.CommandType,
		"params", req.Parameters)

	// TODO: Step 2 - Forward this command to EMQX via MQTT

	return &pb.SendCommandResponse{
		Accepted: true,
		Message:  "Command queued in Hub",
	}, nil
}
