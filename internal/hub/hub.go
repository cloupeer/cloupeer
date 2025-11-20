package hub

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	pb "cloupeer.io/cloupeer/api/proto/v1"
	iovv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
	"cloupeer.io/cloupeer/pkg/log"
)

type HubServer struct {
	namespace       string
	httpAddr        string
	grpcAddr        string
	mqttBroker      string
	mqttUsername    string
	mqttPassword    string
	mqttTopicPrefix string
	k8sclient       controllerclient.Client
	httpClient      *http.Client
	mqttMgr         *autopaho.ConnectionManager
}

func (s *HubServer) Run(ctx context.Context) error {
	// Initialize Kubernetes client
	if err := s.initK8sClient(); err != nil {
		return err
	}

	if err := s.initMQTTClient(ctx); err != nil {
		return err
	}
	defer s.mqttMgr.Disconnect(context.Background())

	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// 1. Start HTTP Server (Health/Metrics)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.runHTTPServer(ctx); err != nil {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// 2. Start gRPC Server (Business Logic)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.runGRPCServer(ctx); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	// 3. MQTT Connection Monitor (Optional but good for logging)
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-s.mqttMgr.Done() // 等待 MQTT 连接彻底关闭（通常发生在 Disconnect 调用后）
		log.Info("MQTT connection manager stopped")
	}()

	// Wait for context cancellation (shutdown signal)
	<-ctx.Done()
	log.Info("Shutdown signal received, waiting for servers to stop...")

	// In a real implementation, we would call Shutdown() on both servers here.
	// For simplicity, we rely on the context cancellation propagation inside the run functions (if implemented)
	// or simply exit as the main process terminates.
	//
	// Create a new context for shutdown with a timeout
	// shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()
	// if err := server.Shutdown(shutdownCtx); err != nil {
	// 	log.Error(err, "cpeer-hub server shutdown failed")
	// }

	log.Info("cpeer-hub server stopped gracefully.")
	return nil
}

func (s *HubServer) runHTTPServer(ctx context.Context) error {
	mux := http.NewServeMux()
	// Add healthz handler
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	// Add readyz handler
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:    s.httpAddr,
		Handler: mux,
	}

	log.Info("cpeer-hub HTTP listening", "address", s.httpAddr, "namespace", s.namespace)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start cpeer-hub server: %w", err)
	}

	return nil
}

func (s *HubServer) runGRPCServer(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on grpc addr %s: %w", s.grpcAddr, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterHubServiceServer(grpcServer, &grpcHandler{parent: s})

	log.Info("cpeer-hub gRPC listening", "address", s.grpcAddr, "namespace", s.namespace)

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	return grpcServer.Serve(lis)
}

func (s *HubServer) initMQTTClient(ctx context.Context) error {
	brokerURL, err := url.Parse(s.mqttBroker)
	if err != nil {
		return fmt.Errorf("invalid mqtt broker url: %w", err)
	}

	clientID := fmt.Sprintf("cpeer-hub-%s", s.namespace)

	cfg := autopaho.ClientConfig{
		ServerUrls: []*url.URL{brokerURL},
		TlsCfg:     &tls.Config{InsecureSkipVerify: true},
		KeepAlive:  20,
		// 自动重连退避策略
		ReconnectBackoff:              autopaho.NewConstantBackoff(3 * time.Second),
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval:         60,
		ConnectUsername:               s.mqttUsername,
		ConnectPassword:               []byte(s.mqttPassword),
		OnConnectionUp: func(cm *autopaho.ConnectionManager, c *paho.Connack) {
			log.Info("Connected to MQTT Broker", "server", s.mqttBroker)

			// 订阅反馈 Topic: iov/cmd-ack/+
			// 假设 mqttTopicPrefix 是 "iov/cmd"，我们需要构造对应的 ack topic
			// 这里简单起见，我们硬编码或约定 ack topic 为 "iov/cmd-ack/+"
			statusTopic := "iov/cmd-ack/+"
			if _, err := cm.Subscribe(context.Background(), &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: statusTopic, QoS: 1},
				},
			}); err != nil {
				log.Error(err, "Failed to subscribe to status topic", "topic", statusTopic)
			}

			// 订阅 URL 请求 Topic: iov/req-url/+
			reqUrlTopic := "iov/req-url/+"
			if _, err := cm.Subscribe(context.Background(), &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: reqUrlTopic, QoS: 1},
				},
			}); err != nil {
				log.Error(err, "Failed to subscribe to req-url topic")
			}
		},
		OnConnectError: func(err error) {
			log.Error(err, "Failed to connect to MQTT Broker", "server", s.mqttBroker)
		},
		ClientConfig: paho.ClientConfig{
			ClientID: clientID,
			OnClientError: func(err error) {
				log.Error(err, "MQTT Client Error")
			},
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					log.Info("Server requested disconnect", "reason", d.Properties.ReasonString)
				} else {
					log.Info("Server requested disconnect", "reasonCode", d.ReasonCode)
				}
			},
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				s.handleStatusReport,
				s.handleFirmwareRequest,
			},
		},
	}

	log.Info("Connecting to MQTT Broker...", "url", s.mqttBroker, "clientID", clientID)

	// NewConnection 会立即尝试连接，并启动后台 goroutine 进行维护
	s.mqttMgr, err = autopaho.NewConnection(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create mqtt connection manager: %w", err)
	}

	// 等待第一次连接成功（可选，为了确保启动时服务可用）
	if err := s.mqttMgr.AwaitConnection(ctx); err != nil {
		return fmt.Errorf("failed to establish initial mqtt connection: %w", err)
	}

	return nil
}

func (s *HubServer) initK8sClient() error {
	k8sconfig, err := config.GetConfig()
	if err != nil {
		log.Error(err, "failed to get kubernetes config")
		return err
	}

	// Create a new scheme and add all our API types and standard types
	cloupeerscheme := runtime.NewScheme()
	utilruntime.Must(scheme.AddToScheme(cloupeerscheme)) // Add standard schemes like v1.Pod, etc.
	utilruntime.Must(iovv1alpha1.AddToScheme(cloupeerscheme))

	k8sclient, err := controllerclient.New(k8sconfig, controllerclient.Options{Scheme: cloupeerscheme})
	if err != nil {
		log.Error(err, "failed to create kubernetes client")
		return err
	}
	s.k8sclient = k8sclient
	return nil
}

// handleStatusReport 处理 Agent 上报的状态
func (s *HubServer) handleStatusReport(pr paho.PublishReceived) (bool, error) {
	// 简单过滤：确保是我们关心的 Topic
	// 如果不是 ack 消息，直接跳过，交给下一个 Handler
	if !strings.Contains(pr.Packet.Topic, "cmd-ack") {
		return false, nil
	}

	// 实际生产中可以使用 paho 的 Router 进行更精准匹配
	var statusMsg pb.AgentCommandStatus
	if err := protojson.Unmarshal(pr.Packet.Payload, &statusMsg); err != nil {
		log.Error(err, "Failed to unmarshal status report")
		return true, nil // 格式错误不重试
	}

	log.Info("Received Status Report",
		"commandName", statusMsg.CommandName,
		"status", statusMsg.Status,
		"msg", statusMsg.Message)

	// 更新 K8s CRD
	ctx := context.Background()

	// 1. 构造 Patch 对象
	// 我们只更新 Status 部分。CommandName 就是 CR Name。
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
		return false, nil // 如果是 K8s 错误，也不要让 MQTT 重发了，避免阻塞
	}

	log.Info("Successfully patched VehicleCommand", "name", cmd.Name, "phase", statusMsg.Status)
	return true, nil
}

// handleFirmwareRequest 处理 Agent 的 URL 请求
func (s *HubServer) handleFirmwareRequest(pr paho.PublishReceived) (bool, error) {
	// 简单判断 Topic 是否匹配 (实际应使用 paho Router)
	// 假设 Topic 格式: iov/req-url/{vehicleID}
	if !strings.Contains(pr.Packet.Topic, "req-url") {
		return false, nil
	}

	// 这里我们直接尝试反序列化，如果成功就是这个消息
	var req pb.GetFirmwareURLRequest
	// 使用 DiscardUnknown 避免解析错误
	uo := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := uo.Unmarshal(pr.Packet.Payload, &req); err != nil {
		// 如果解析失败，可能不是这个类型的消息，返回 false 让下一个 Handler 处理 (如果有)
		// 或者如果是唯一 Handler，就返回 nil 忽略
		return false, nil
	}

	// 如果关键字段为空，说明可能解析错了消息类型
	if req.VehicleId == "" || req.RequestId == "" {
		return false, nil
	}

	log.Info("Received Firmware URL Request", "vehicleID", req.VehicleId, "ver", req.DesiredVersion)

	// 模拟生成 URL
	resp := &pb.GetFirmwareURLResponse{
		RequestId:   req.RequestId,
		DownloadUrl: fmt.Sprintf("https://firmware.cloupeer.io/%s/%s.bin", req.VehicleId, req.DesiredVersion),
	}

	// 发送响应
	respTopic := fmt.Sprintf("iov/resp-url/%s", req.VehicleId)
	respBytes, _ := protojson.Marshal(resp)

	_, err := s.mqttMgr.Publish(context.Background(), &paho.Publish{
		Topic:   respTopic,
		QoS:     1,
		Payload: respBytes,
	})

	if err != nil {
		log.Error(err, "Failed to publish firmware URL response")
	} else {
		log.Info("Sent Firmware URL", "url", resp.DownloadUrl)
	}

	return true, nil
}
