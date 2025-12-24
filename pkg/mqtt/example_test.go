package mqtt_test

import (
	"context"
	"fmt"
	"time"

	"github.com/autopeer-io/autopeer/pkg/log"
	"github.com/autopeer-io/autopeer/pkg/mqtt"
)

// ExampleClient 展示了 Autopeer MQTT 组件的标准使用流程。
// 这个示例模拟了一个组件（如 Hub 或 Agent）如何初始化 MQTT 客户端、订阅主题并发送消息。
func ExampleClient() {
	// 1. 准备配置
	// 在实际应用中，这些值通常来自 pkg/options 或 CLI 参数
	cfg := &mqtt.ClientConfig{
		BrokerURL:      "tcp://localhost:1883",
		ClientID:       "example-component-001",
		Username:       "admin",
		Password:       "public",
		KeepAlive:      60,
		ConnectTimeout: 5 * time.Second,
		// 核心规范：开发环境使用自签名证书，必须跳过验证
		InsecureSkipVerify: true,
		// 对于需要接收离线消息的 Agent，CleanStart 通常设为 false
		CleanStart: false,
	}

	// 2. 创建客户端实例
	// 此时尚未建立连接
	client, err := mqtt.NewClient(cfg)
	if err != nil {
		log.Error(err, "Failed to create MQTT client")
		return
	}

	// 3. 启动客户端 (非阻塞)
	// Start 会立即返回，连接过程在后台进行（包含自动重连机制）
	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Error(err, "Failed to start MQTT client")
		return
	}

	// 4. 定义消息处理函数 (Handler)
	// 这是业务逻辑的入口，处理收到的 payload
	myHandler := func(ctx context.Context, topic string, payload []byte) {
		// 注意：Handler 在独立的 goroutine 中运行，不要在此执行耗时过长的阻塞操作
		fmt.Printf("Received message on topic %s: %s\n", topic, string(payload))
	}

	// 5. 订阅主题
	// 我们的组件封装了路由分发。这里注册的 topic 支持通配符 (如 "iov/cmd/+")。
	// 关键特性：如果连接断开重连，组件会自动重新发送 SUBSCRIBE 包，业务层无需感知。
	subTopic := "iov/cmd/+"
	if err := client.Subscribe(ctx, subTopic, 1, myHandler); err != nil {
		log.Error(err, "Failed to subscribe", "topic", subTopic)
	}

	// 6. 等待连接就绪 (可选)
	// 如果你的业务逻辑必须在 MQTT 连接建立后才能继续（例如 Readiness Probe），
	// 可以使用 AwaitConnection 进行阻塞等待。
	fmt.Println("Waiting for connection...")
	if err := client.AwaitConnection(ctx); err != nil {
		log.Error(err, "Connection timed out")
		return
	}
	fmt.Println("MQTT Connected!")

	// 7. 发布消息
	// 使用 QoS 1 确保至少送达一次
	pubTopic := "iov/status/vh-001"
	payload := []byte(`{"status": "online", "version": "v1.0.0"}`)
	if err := client.Publish(ctx, pubTopic, 1, false, payload); err != nil {
		log.Error(err, "Failed to publish message", "topic", pubTopic)
	}

	// 8. 优雅关闭
	// 在应用退出时调用
	client.Disconnect(ctx)
}
