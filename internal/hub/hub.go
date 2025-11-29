package hub

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protojson"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"

	pb "cloupeer.io/cloupeer/api/proto/v1"
	"cloupeer.io/cloupeer/internal/hub/storage"
	iovv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
	"cloupeer.io/cloupeer/pkg/log"
	"cloupeer.io/cloupeer/pkg/mqtt"
	mqtttopic "cloupeer.io/cloupeer/pkg/mqtt/topic"
)

type HubServer struct {
	namespace    string
	httpserver   *http.Server
	grpcserver   *GrpcServer
	k8sclient    controllerclient.Client
	mqttclient   mqtt.Client
	topicbuilder *mqtttopic.TopicBuilder
	storage      storage.Provider
}

// Run starts the HubServer and blocks until it stops.
func (s *HubServer) Run(ctx context.Context) error {
	log.Info("Starting cpeer-hub server...")

	// 0. Check Storage Connectivity
	if err := s.storage.CheckBucket(ctx); err != nil {
		return fmt.Errorf("failed to connect to object storage: %w", err)
	}
	log.Info("Object Storage Connected")

	// 1. Start MQTT Client (Critical Dependency)
	// We start this synchronously because other components might depend on connectivity.
	if err := s.mqttclient.Start(ctx); err != nil {
		return fmt.Errorf("failed to start mqtt client: %w", err)
	}

	// Ensure MQTT disconnects when Run exits (LIFO order)
	defer func() {
		log.Info("Disconnecting MQTT client...")
		// Use a fresh context with timeout to ensure disconnect packet sends
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.mqttclient.Disconnect(shutdownCtx)
		log.Info("MQTT client disconnected")
	}()

	// Optional: Wait for initial connection if strict consistency is required.
	// In many cases, async reconnect logic is enough.
	if err := s.mqttclient.AwaitConnection(ctx); err != nil {
		// If we can't connect at boot, fail fast.
		return fmt.Errorf("mqtt initial connection failed: %w", err)
	}
	log.Info("MQTT Connected")

	if err := s.setupMQTTSubscriptions(ctx); err != nil {
		return err
	}

	// 2. Setup ErrorGroup for managing HTTP & gRPC servers
	// The group context 'gCtx' will be canceled if any goroutine returns a non-nil error,
	// or if the parent 'ctx' is canceled.
	g, gCtx := errgroup.WithContext(ctx)

	// --- HTTP Server (Health/Metrics) ---
	g.Go(func() error {
		// s.runHTTPServer is blocking. It should monitor gCtx.Done() for shutdown.
		// When gCtx is canceled, runHTTPServer should exit gracefully.
		if err := s.runHTTPServer(gCtx); err != nil {
			log.Error(err, "HTTP server failed")
			return err
		}
		return nil
	})

	// --- gRPC Server (Business Logic) ---
	g.Go(func() error {
		// Similarly, s.runGRPCServer handles its own graceful stop on context cancellation.
		if err := s.runGRPCServer(gCtx); err != nil {
			log.Error(err, "gRPC server failed")
			return err
		}
		return nil
	})

	// 3. Wait for all servers to stop
	// This blocks until:
	// a) The parent 'ctx' is canceled (OS signal) -> trigger shutdown -> wait for servers to exit
	// b) One of the servers returns an error -> cancel gCtx -> trigger shutdown for others -> return error
	log.Info("All servers started, waiting for shutdown signal...")
	if err := g.Wait(); err != nil {
		log.Error(err, "HubServer exited with error")
		return err
	}

	log.Info("HubServer stopped gracefully")
	return nil
}

func (s *HubServer) setupMQTTSubscriptions(ctx context.Context) error {
	ackTopic := s.topicbuilder.CommandAckWildcard()
	if err := s.mqttclient.Subscribe(ctx, ackTopic, 1, s.handleStatusReport); err != nil {
		return fmt.Errorf("failed to subscribe to ack topic %s: %w", ackTopic, err)
	}

	reqUrlTopic := s.topicbuilder.FirmwareURLReqWildcard()
	if err := s.mqttclient.Subscribe(ctx, reqUrlTopic, 1, s.handleFirmwareRequest); err != nil {
		return fmt.Errorf("failed to subscribe to req-url topic %s: %w", reqUrlTopic, err)
	}

	// Subscribe to registration requests
	registerTopic := s.topicbuilder.RegisterWildcard()
	if err := s.mqttclient.Subscribe(ctx, registerTopic, 1, s.handleRegistration); err != nil {
		return fmt.Errorf("failed to subscribe to register topic %s: %w", registerTopic, err)
	}

	return nil
}

func (s *HubServer) runHTTPServer(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpserver.Shutdown(shutdownCtx)
	}()

	log.Info("cpeer-hub HTTP listening", "addr", s.httpserver.Addr)
	if err := s.httpserver.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start cpeer-hub server: %w", err)
	}

	return nil
}

func (s *HubServer) runGRPCServer(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		s.grpcserver.srv.GracefulStop()
	}()

	log.Info("cpeer-hub gRPC listening", "addr", s.grpcserver.lis.Addr().String())
	return s.grpcserver.srv.Serve(s.grpcserver.lis)
}

// handleStatusReport 处理 Agent 上报的状态
func (s *HubServer) handleStatusReport(ctx context.Context, topic string, payload []byte) {
	if !strings.Contains(topic, mqtttopic.SuffixCommandAck) {
		return
	}

	var statusMsg pb.AgentCommandStatus
	if err := protojson.Unmarshal(payload, &statusMsg); err != nil {
		log.Error(err, "Failed to unmarshal status report")
		return
	}

	log.Info("Received Status Report",
		"commandName", statusMsg.CommandName,
		"status", statusMsg.Status,
		"msg", statusMsg.Message)

	cmd := &iovv1alpha1.VehicleCommand{}
	cmd.Name = statusMsg.CommandName
	cmd.Namespace = s.namespace // 假设所有 Command 都在 Hub 所在的 Namespace

	// 2. 获取当前对象 (可选，为了更安全的 Patch，或者直接使用 MergePatch)
	// 这里我们使用 MergeFrom 进行 Patch
	patch := controllerclient.MergeFrom(cmd.DeepCopy())

	// 3. 设置新状态
	cmd.Status.Phase = iovv1alpha1.CommandPhase(statusMsg.Status)
	cmd.Status.Message = statusMsg.Message
	now := metav1.Now()
	cmd.Status.LastUpdateTime = &now

	// 根据状态设置特定的时间戳
	if statusMsg.Status == string(iovv1alpha1.CommandPhaseReceived) {
		cmd.Status.AcknowledgeTime = &now
	} else if statusMsg.Status == string(iovv1alpha1.CommandPhaseSucceeded) ||
		statusMsg.Status == string(iovv1alpha1.CommandPhaseFailed) {
		cmd.Status.CompletionTime = &now
	}

	// 4. 执行 Patch
	if err := s.k8sclient.Status().Patch(ctx, cmd, patch); err != nil {
		log.Error(err, "Failed to patch VehicleCommand status", "name", cmd.Name)
		return
	}

	log.Info("Successfully patched VehicleCommand", "name", cmd.Name, "phase", statusMsg.Status)
}

func (s *HubServer) handleFirmwareRequest(ctx context.Context, topic string, payload []byte) {
	if !strings.Contains(topic, mqtttopic.SuffixFirmwareReq) {
		return
	}

	var req pb.GetFirmwareURLRequest
	uo := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := uo.Unmarshal(payload, &req); err != nil {
		return
	}

	// 如果关键字段为空，说明可能解析错了消息类型
	if req.VehicleId == "" || req.RequestId == "" {
		return
	}

	log.Info("Received Firmware URL Request", "vehicleID", req.VehicleId, "ver", req.DesiredVersion)
	resp := &pb.GetFirmwareURLResponse{RequestId: req.RequestId}

	// 假设固件文件在存储桶中的路径格式为: {version}/vehicle.bin
	// 在真实场景中，这里应该查询数据库或 K8s 获取该版本对应的真实 ObjectKey
	objectKey := fmt.Sprintf("%s/vehicle.bin", req.DesiredVersion)

	// 生成 1 小时有效期的链接
	downloadURL, err := s.storage.GeneratePresignedURL(ctx, objectKey, 1*time.Hour)
	if err != nil {
		log.Error(err, "Failed to generate presigned URL")
		resp.ErrorMessage = "Internal Server Error: Storage unavailable"
	} else {
		resp.DownloadUrl = downloadURL
	}

	// 发送响应
	respBytes, _ := protojson.Marshal(resp)
	err = s.mqttclient.Publish(ctx, s.topicbuilder.FirmwareURLResp(req.VehicleId), 1, false, respBytes)
	if err != nil {
		log.Error(err, "Failed to publish firmware URL response")
	} else {
		log.Info("Sent Firmware URL", "url", resp.DownloadUrl)
	}
}

// handleRegistration handles the auto-discovery of vehicles.
func (s *HubServer) handleRegistration(ctx context.Context, topic string, payload []byte) {
	// 1. Unmarshal payload
	var req pb.RegisterVehicleRequest
	if err := protojson.Unmarshal(payload, &req); err != nil {
		log.Error(err, "Failed to unmarshal registration request")
		return
	}

	if req.VehicleId == "" {
		log.Warn("Received registration request without vehicleID")
		return
	}

	log.Info("Received Registration Request", "vehicleID", req.VehicleId, "ver", req.FirmwareVersion)

	// 2. Check if Vehicle exists
	var vehicle iovv1alpha1.Vehicle
	err := s.k8sclient.Get(ctx, types.NamespacedName{Name: req.VehicleId, Namespace: s.namespace}, &vehicle)

	if err != nil {
		if apierrors.IsNotFound(err) {
			// --- Scenario A: New Vehicle (Auto-Create) ---

			// Step 1: Create the CR skeleton (Spec only)
			// [关键点] 我们特意将 Spec.FirmwareVersion 留空。
			// 这表示控制平面此时对该车辆没有特定的版本要求（无意图）。
			// 这样可以防止 Controller 误判 从而触发升级。
			newVehicle := &iovv1alpha1.Vehicle{
				ObjectMeta: metav1.ObjectMeta{
					Name:      req.VehicleId,
					Namespace: s.namespace,
					Labels: map[string]string{
						"app.kubernetes.io/managed-by":    "cpeer-hub",
						"iov.cloupeer.io/auto-discovered": "true",
					},
				},
				Spec: iovv1alpha1.VehicleSpec{
					Description: req.Description,
					// FirmwareVersion: "", // Explicitly left empty
				},
			}

			if createErr := s.k8sclient.Create(ctx, newVehicle); createErr != nil {
				log.Error(createErr, "Failed to create auto-discovered Vehicle", "vehicleID", req.VehicleId)
				return
			}
			log.Info("Auto-discovered new Vehicle (Spec created)", "vehicleID", req.VehicleId)

			// Step 2: Initialize Status
			// 由于 K8s API Server 在 Create 时会丢弃 status 字段，我们必须发起第二次调用来专门更新它。
			now := metav1.Now()
			newVehicle.Status.LastSeenTime = &now
			newVehicle.Status.ReportedFirmwareVersion = req.FirmwareVersion
			newVehicle.Status.Phase = iovv1alpha1.VehiclePhaseIdle

			// 初始化 Conditions，标记为 Ready 和 Synced
			newVehicle.Status.Conditions = []metav1.Condition{
				{
					Type:               iovv1alpha1.ConditionTypeReady,
					Status:             metav1.ConditionTrue,
					Reason:             "AutoRegistered",
					Message:            "Vehicle auto-registered successfully",
					LastTransitionTime: now,
				},
				{
					Type:               iovv1alpha1.ConditionTypeSynced,
					Status:             metav1.ConditionTrue,
					Reason:             "AutoRegistered",
					Message:            fmt.Sprintf("Initial version %s", req.FirmwareVersion),
					LastTransitionTime: now,
				},
			}

			// 使用 Update 更新 Status 子资源 (因为这是一个全新的对象，Update 比 Patch 更直接且开销略小)
			if statusErr := s.k8sclient.Status().Update(ctx, newVehicle); statusErr != nil {
				log.Error(statusErr, "Failed to init Vehicle status", "vehicleID", req.VehicleId)
			} else {
				log.Info("Initialized Vehicle status", "vehicleID", req.VehicleId)
			}
			return
		}
		// Other API errors
		log.Error(err, "Failed to query Vehicle", "vehicleID", req.VehicleId)
		return
	}

	// --- Scenario B: Existing Vehicle (Update Heartbeat) ---
	// Use Patch to minimize conflict
	patch := controllerclient.MergeFrom(vehicle.DeepCopy())

	now := metav1.Now()
	vehicle.Status.LastSeenTime = &now

	// Also update reported version if changed (e.g. manual update via USB)
	if req.FirmwareVersion != "" {
		vehicle.Status.ReportedFirmwareVersion = req.FirmwareVersion
	}

	if err := s.k8sclient.Status().Patch(ctx, &vehicle, patch); err != nil {
		log.Error(err, "Failed to update Vehicle heartbeat", "vehicleID", req.VehicleId)
	} else {
		log.Debug("Updated Vehicle heartbeat", "vehicleID", req.VehicleId)
	}
}
