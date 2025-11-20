package edgeagent

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"google.golang.org/protobuf/encoding/protojson"

	pb "cloupeer.io/cloupeer/api/proto/v1"
	"cloupeer.io/cloupeer/pkg/log"
)

// Agent is the core struct for the edge agent business logic.
type Agent struct {
	vehicleID       string
	mqttBroker      string
	mqttUsername    string
	mqttPassword    string
	mqttTopicPrefix string

	httpClient *http.Client
	mqttMgr    *autopaho.ConnectionManager

	// 用于接收固件 URL 响应的通道
	// Key: RequestID, Value: Response
	pendingRequests map[string]chan string
	reqMu           sync.Mutex // 保护 map
}

// Run starts the main loop of the agent and handles graceful shutdown.
func (a *Agent) Run(ctx context.Context) error {
	log.Info("Starting cpeer-edge-agent", "vehicleID", a.vehicleID)

	// 初始化 MQTT
	if err := a.initMQTT(ctx); err != nil {
		return err
	}
	defer a.mqttMgr.Disconnect(context.Background())

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

func (a *Agent) initMQTT(ctx context.Context) error {
	brokerURL, err := url.Parse(a.mqttBroker)
	if err != nil {
		return fmt.Errorf("invalid broker url: %w", err)
	}

	clientID := fmt.Sprintf("agent-%s", a.vehicleID)

	cfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{brokerURL},
		TlsCfg:                        &tls.Config{InsecureSkipVerify: true},
		KeepAlive:                     20,
		ReconnectBackoff:              autopaho.NewConstantBackoff(5 * time.Second),
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval:         60,
		// 认证信息放在顶层
		ConnectUsername: a.mqttUsername,
		ConnectPassword: []byte(a.mqttPassword),

		OnConnectionUp: func(cm *autopaho.ConnectionManager, c *paho.Connack) {
			log.Info("Agent connected to MQTT", "server", a.mqttBroker)

			// *** 关键步骤：连接成功后立即订阅 ***
			// 构造 Topic: iov/cmd/{vehicleID}
			topic := fmt.Sprintf("%s/%s", a.mqttTopicPrefix, a.vehicleID)
			if _, err := cm.Subscribe(context.Background(), &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: topic, QoS: 1},
				},
			}); err != nil {
				log.Error(err, "Failed to subscribe", "topic", topic)
			} else {
				log.Info("Subscribed to command topic", "topic", topic)
			}

			// 订阅: iov/resp-url/{vehicleID}
			respTopic := fmt.Sprintf("iov/resp-url/%s", a.vehicleID)
			if _, err := cm.Subscribe(context.Background(), &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: respTopic, QoS: 1},
				},
			}); err != nil {
				log.Error(err, "Failed to subscribe", "respTopic", respTopic)
			} else {
				log.Info("Subscribed to resp-url topic", "respTopic", respTopic)
			}
		},
		OnConnectError: func(err error) {
			log.Error(err, "Agent failed to connect to MQTT")
		},
		ClientConfig: paho.ClientConfig{
			ClientID: clientID,
			// 处理接收到的消息
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				a.handleMessage,
			},
			OnClientError: func(err error) {
				log.Error(err, "MQTT Client Error")
			},
		},
	}

	log.Info("Connecting to MQTT Broker...", "url", a.mqttBroker, "clientID", clientID)
	a.mqttMgr, err = autopaho.NewConnection(ctx, cfg)
	if err != nil {
		return err
	}

	// 等待首次连接
	if err := a.mqttMgr.AwaitConnection(ctx); err != nil {
		return err
	}

	return nil
}

func (a *Agent) handleMessage(pr paho.PublishReceived) (bool, error) {
	log.Info("Received message", "topic", pr.Packet.Topic)

	// 使用 protojson 进行反序列化
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true, // 兼容性设计：忽略未知的字段
	}

	// 尝试解析为 URL Response
	var resp pb.GetFirmwareURLResponse
	if err := unmarshaler.Unmarshal(pr.Packet.Payload, &resp); err == nil && resp.RequestId != "" {
		// 这是一个 URL 响应
		a.reqMu.Lock()
		if ch, ok := a.pendingRequests[resp.RequestId]; ok {
			ch <- resp.DownloadUrl
			delete(a.pendingRequests, resp.RequestId) // 清理
		}
		a.reqMu.Unlock()
		return true, nil
	}

	// 如果不是 Response，尝试解析为 Command
	// 使用生成的 Protobuf 结构体
	var cmd pb.AgentCommand

	if err := unmarshaler.Unmarshal(pr.Packet.Payload, &cmd); err != nil {
		log.Error(err, "Failed to unmarshal agent command proto", "raw", string(pr.Packet.Payload))
		return true, nil
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

	return true, nil
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

	a.mqttMgr.Publish(context.Background(), &paho.Publish{
		Topic:   fmt.Sprintf("iov/req-url/%s", a.vehicleID),
		QoS:     1,
		Payload: reqBytes,
	})

	// 3. 等待响应 (带超时)
	var downloadURL string
	select {
	case url := <-respChan:
		downloadURL = url
		log.Info("Received Firmware URL", "url", url)
	case <-time.After(5 * time.Second):
		log.Error(nil, "Timeout waiting for firmware URL")
		a.publishStatus(cmd.CommandName, "Failed", "Timeout fetching URL")

		// 清理 map
		a.reqMu.Lock()
		delete(a.pendingRequests, reqID)
		a.reqMu.Unlock()
		return
	}

	// 4. 开始下载 (Running)
	a.publishStatus(cmd.CommandName, "Running", fmt.Sprintf("Downloading from %s...", downloadURL))
	time.Sleep(3 * time.Second) // 模拟下载

	// 5. 完成
	a.publishStatus(cmd.CommandName, "Succeeded", "Update installed")
}

func (a *Agent) publishStatus(cmdName, status, msg string) {
	topic := fmt.Sprintf("iov/cmd-ack/%s", a.vehicleID)

	payload := &pb.AgentCommandStatus{
		CommandName: cmdName,
		Status:      status,
		Message:     msg,
	}

	bytes, _ := protojson.Marshal(payload)

	if _, err := a.mqttMgr.Publish(context.Background(), &paho.Publish{
		Topic:   topic,
		QoS:     1,
		Payload: bytes,
	}); err != nil {
		log.Error(err, "Failed to publish status", "status", status)
	}
}
