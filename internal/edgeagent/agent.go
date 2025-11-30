package edgeagent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	pb "cloupeer.io/cloupeer/api/proto/v1"
	"cloupeer.io/cloupeer/internal/pkg/mqtt/paths"
	"cloupeer.io/cloupeer/pkg/log"
	"cloupeer.io/cloupeer/pkg/mqtt"
	mqtttopic "cloupeer.io/cloupeer/pkg/mqtt/topic"
)

// Agent is the core struct for the edge agent business logic.
type Agent struct {
	vehicleID string

	mqttclient   mqtt.Client
	topicbuilder *mqtttopic.Builder

	httpClient *http.Client

	// 用于接收固件 URL 响应的通道
	// Key: RequestID, Value: Response
	pendingRequests map[string]chan string
	reqMu           sync.Mutex // 保护 map
}

// Run starts the main loop of the agent and handles graceful shutdown.
func (a *Agent) Run(ctx context.Context) error {
	log.Info("Starting cpeer-edge-agent", "vehicleID", a.vehicleID)

	// 初始化 MQTT
	if err := a.mqttclient.Start(ctx); err != nil {
		return err
	}

	defer func() {
		log.Info("Disconnecting MQTT client...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		a.mqttclient.Disconnect(shutdownCtx)
	}()

	if err := a.mqttclient.AwaitConnection(ctx); err != nil {
		return err
	}

	a.setupMQTTSubscriptions(ctx)

	// Send Registration/Online Message
	go a.registerIdentity(ctx)

	// 等待信号或上下文取消
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sig:
		log.Info("OS signal received, shutting down...")
	case <-ctx.Done():
		log.Info("Context cancelled, shutting down...")
	}

	return nil
}

func (a *Agent) setupMQTTSubscriptions(ctx context.Context) error {
	cmdTopic := a.topicbuilder.Build(paths.Command, a.vehicleID)
	if err := a.mqttclient.Subscribe(ctx, cmdTopic, 1, a.handleMessage); err != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", cmdTopic, err)
	}

	respTopic := a.topicbuilder.Build(paths.OTAResponse, a.vehicleID)
	if err := a.mqttclient.Subscribe(ctx, respTopic, 1, a.handleMessage); err != nil {
		return fmt.Errorf("failed to subscribe to req-url topic %s: %w", respTopic, err)
	}

	return nil
}

func (a *Agent) handleMessage(ctx context.Context, topic string, payload []byte) {
	log.Info("Received message", "topic", topic)

	// 使用 protojson 进行反序列化
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true, // 兼容性设计：忽略未知的字段
	}

	// 尝试解析为 URL Response
	var resp pb.GetFirmwareURLResponse
	if err := unmarshaler.Unmarshal(payload, &resp); err == nil && resp.RequestId != "" {
		// 这是一个 URL 响应
		a.reqMu.Lock()
		if ch, ok := a.pendingRequests[resp.RequestId]; ok {
			ch <- resp.DownloadUrl
			delete(a.pendingRequests, resp.RequestId) // 清理
		}
		a.reqMu.Unlock()
		return
	}

	// 如果不是 Response，尝试解析为 Command
	// 使用生成的 Protobuf 结构体
	var cmd pb.AgentCommand
	if err := unmarshaler.Unmarshal(payload, &cmd); err != nil {
		log.Error(err, "Failed to unmarshal agent command proto", "raw", string(payload))
		return
	}

	log.Info(">>> PROCESSING COMMAND <<<",
		"Type", cmd.CommandType,
		"ID", cmd.CommandName,
		"Params", cmd.Parameters,
		"Time", time.Unix(cmd.Timestamp, 0).Format(time.RFC3339))

	// 这里是根据架构设计的后续步骤：
	// 1. "触发一条消息提醒车主" -> Log / UI Event
	// 2. "车主点击升级" -> 模拟等待或直接调用
	// 修改 OTA 处理逻辑：
	if cmd.CommandType == "OTA" {
		go a.executeOTAProcess(&cmd)
	}
}

func (a *Agent) executeOTAProcess(cmd *pb.AgentCommand) {
	// 1. ACK
	a.publishStatus(cmd.CommandName, "Received", "Waiting for user confirmation...")

	// 模拟：车主等待确认 (例如 2秒)
	log.Info("[UI] User notification: New firmware available. Click to upgrade.")
	time.Sleep(2 * time.Second)
	log.Info("[UI] User clicked 'Upgrade'. Requesting URL...")

	// 2. 请求 URL
	targetVer := cmd.Parameters["version"]
	reqID := fmt.Sprintf("req-%d", time.Now().UnixNano())

	// 创建接收通道
	respChan := make(chan string, 1)
	a.reqMu.Lock()
	a.pendingRequests[reqID] = respChan
	a.reqMu.Unlock()

	// 发送请求
	req := &pb.GetFirmwareURLRequest{
		VehicleId:      a.vehicleID,
		DesiredVersion: targetVer,
		RequestId:      reqID,
	}
	reqBytes, _ := protojson.Marshal(req)

	topic := a.topicbuilder.Build(paths.OTARequest, a.vehicleID)
	a.mqttclient.Publish(context.Background(), topic, 1, false, reqBytes)

	// 3. 等待响应 (带超时)
	var downloadURL string
	select {
	case url := <-respChan:
		downloadURL = url
		log.Info("Received Firmware URL", "url", url)
	case <-time.After(15 * time.Second):
		log.Error(nil, "Timeout waiting for firmware URL")
		a.publishStatus(cmd.CommandName, "Failed", "Timeout fetching URL")

		// 清理 map
		a.reqMu.Lock()
		delete(a.pendingRequests, reqID)
		a.reqMu.Unlock()
		return
	}

	// 4. 开始下载 (Running)
	a.publishStatus(cmd.CommandName, "Running", "Downloading firmware artifact...")

	// 执行真实的下载校验
	if err := a.downloadAndVerify(downloadURL); err != nil {
		log.Error(err, "Download failed")
		a.publishStatus(cmd.CommandName, "Failed", fmt.Sprintf("Download failed: %v", err))
		return
	}

	// 5. 完成
	a.publishStatus(cmd.CommandName, "Succeeded", "Update installed")
}

// downloadAndVerify performs a real HTTP GET to validate the URL.
// In a production agent, this would also verify SHA256 checksums and write to disk.
func (a *Agent) downloadAndVerify(url string) error {
	// 1. Create request
	resp, err := a.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	// 2. Check status code (Expect 200 OK)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %s", resp.Status)
	}

	// 3. Simulate consuming the body (or write to /tmp/firmware.bin)
	// We just read it to ensure the stream is valid.
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}

	return nil
}

func (a *Agent) publishStatus(cmdName, status, msg string) {
	topic := a.topicbuilder.Build(paths.CommandAck, a.vehicleID)

	payload := &pb.AgentCommandStatus{
		CommandName: cmdName,
		Status:      status,
		Message:     msg,
	}

	bytes, _ := protojson.Marshal(payload)

	if err := a.mqttclient.Publish(context.Background(), topic, 1, false, bytes); err != nil {
		log.Error(err, "Failed to publish status", "status", status)
	}
}

// registerIdentity sends the initial registration packet to the Hub.
func (a *Agent) registerIdentity(ctx context.Context) {
	// Simulation: Get current version from local system
	// In production, this comes from a version file or API.
	currentVersion := "v1.0.0"

	req := &pb.RegisterVehicleRequest{
		VehicleId:       a.vehicleID,
		FirmwareVersion: currentVersion,
		Description:     "Edge Agent Auto-Registration",
		Timestamp:       time.Now().Unix(),
	}

	bytes, err := protojson.Marshal(req)
	if err != nil {
		log.Error(err, "Failed to marshal registration request")
		return
	}

	topic := a.topicbuilder.Build(paths.Register, a.vehicleID)

	// Retry logic could be added here, but for now we send once (QoS 1 handles delivery)
	if err := a.mqttclient.Publish(ctx, topic, 1, false, bytes); err != nil {
		log.Error(err, "Failed to send registration request")
	} else {
		log.Info("Sent registration request", "topic", topic, "version", currentVersion)
	}
}
