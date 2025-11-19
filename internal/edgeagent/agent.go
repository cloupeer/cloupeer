package edgeagent

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
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

	// 构造 Topic: iov/cmd/{vehicleID}
	topic := fmt.Sprintf("%s/%s", a.mqttTopicPrefix, a.vehicleID)

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
			if _, err := cm.Subscribe(context.Background(), &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: topic, QoS: 1},
				},
			}); err != nil {
				log.Error(err, "Failed to subscribe", "topic", topic)
			} else {
				log.Info("Subscribed to command topic", "topic", topic)
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

	// 使用生成的 Protobuf 结构体
	var cmd pb.AgentCommand

	// 使用 protojson 进行反序列化
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true, // 兼容性设计：忽略未知的字段
	}

	if err := unmarshaler.Unmarshal(pr.Packet.Payload, &cmd); err != nil {
		log.Error(err, "Failed to unmarshal agent command proto", "raw", string(pr.Packet.Payload))
		return true, nil
	}

	log.Info(">>> PROCESSING COMMAND <<<",
		"Type", cmd.CommandType,
		"ID", cmd.CommandId,
		"Params", cmd.Parameters,
		"Time", time.Unix(cmd.Timestamp, 0).Format(time.RFC3339))

	// 这里是根据架构设计的后续步骤：
	// 1. "触发一条消息提醒车主" -> Log / UI Event
	// 2. "车主点击升级" -> 模拟等待或直接调用

	return true, nil
}
