package mqtt

import (
	"context"
	"fmt"

	pb "cloupeer.io/cloupeer/api/proto/v1"
	"cloupeer.io/cloupeer/internal/cloudhub/core/model"
	"cloupeer.io/cloupeer/internal/pkg/mqtt/paths"
	"cloupeer.io/cloupeer/pkg/log"
	"google.golang.org/protobuf/encoding/protojson"
)

func (s *Server) handleRegister(ctx context.Context, req *pb.RegisterVehicleRequest) error {
	if req.VehicleId == "" {
		log.Warn("Received register request without vehicleID")
		return nil
	}

	log.Info("Received register request", "vehicleID", req.VehicleId, "version", req.FirmwareVersion)

	v := &model.Vehicle{
		ID:              req.VehicleId,
		ReportedVersion: req.FirmwareVersion,
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

	return s.svc.UpdateCommandStatus(ctx, req.CommandName, model.CommandStatus(req.Status), req.Message)
}

func (s *Server) handleOTARequest(ctx context.Context, req *pb.OTARequest) error {
	// 如果关键字段为空，说明可能解析错了消息类型
	if req.VehicleId == "" || req.RequestId == "" {
		return fmt.Errorf("either VehicleId[%s] or RequestId[%s] is empty", req.VehicleId, req.RequestId)
	}

	resp := &pb.OTAResponse{RequestId: req.RequestId}

	// 假设固件文件在存储桶中的路径格式为: {version}/vehicle.bin
	// 在真实场景中，这里应该查询数据库或 K8s 获取该版本对应的真实 ObjectKey
	objectKey := fmt.Sprintf("%s/vehicle.bin", req.DesiredVersion)

	downloadURL, err := s.svc.GetFirmwareDownloadURL(ctx, objectKey)
	if err != nil {
		log.Error(err, "Failed to get firmware download URL")
		resp.ErrorMessage = "Internal Server Error: DownloadUrl unavailable"
	} else {
		resp.DownloadUrl = downloadURL
	}

	// 发送响应
	respBytes, _ := protojson.Marshal(resp)
	topicPath := s.topics.Build(paths.OTAResponse, req.VehicleId)
	qos := 1
	retain := true
	if err = s.client.Publish(ctx, topicPath, qos, retain, respBytes); err != nil {
		log.Error(err, "Failed to publish firmware URL response")
		return err
	}

	log.Info("Sent Firmware URL", "url", resp.DownloadUrl)
	return nil
}
