package bridge

import (
	"fmt"

	"github.com/autopeer-io/autopeer/internal/bridge/core/service"
	"github.com/autopeer-io/autopeer/internal/bridge/k8s"
	"github.com/autopeer-io/autopeer/internal/bridge/notifier"
	"github.com/autopeer-io/autopeer/internal/bridge/server"
	"github.com/autopeer-io/autopeer/internal/bridge/server/grpc"
	"github.com/autopeer-io/autopeer/internal/bridge/server/http"
	"github.com/autopeer-io/autopeer/internal/bridge/server/mqtt"
	"github.com/autopeer-io/autopeer/internal/bridge/storage"
	pkgmqtt "github.com/autopeer-io/autopeer/pkg/mqtt"
	"github.com/autopeer-io/autopeer/pkg/mqtt/topic"
	"github.com/autopeer-io/autopeer/pkg/options"
)

type Config struct {
	KubeOptions *options.KubeOptions
	HttpOptions *options.HttpOptions
	GrpcOptions *options.GrpcOptions
	MqttOptions *options.MqttOptions
	S3Options   *options.S3Options
}

func (cfg *Config) NewHubServer() (*CloudHubServer, error) {
	k8sClient, err := k8s.InitializeK8sClient()
	if err != nil {
		return nil, err
	}

	pipeline := k8s.NewPipeline(cfg.KubeOptions.Namespace, k8sClient)
	// k8sRepo implements both VehicleRepository and CommandRepository
	k8sRepo := k8s.NewRepository(cfg.KubeOptions.Namespace, k8sClient, pipeline)

	// Use the shared MQTT client factory from pkg/mqtt
	mqttClient, err := pkgmqtt.NewClient(cfg.MqttOptions.ToClientConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to init mqtt client: %w", err)
	}

	topicBuilder := topic.NewBuilder(cfg.MqttOptions.TopicRoot)

	// Infrastructure: Storage (Secondary Adapter)
	storageAdapter, err := storage.NewMinIO(cfg.S3Options)
	if err != nil {
		return nil, err
	}

	// Infrastructure: Notifier (Secondary Adapter)
	notifierAdapter, err := notifier.NewMQTTNotifier(mqttClient, topicBuilder)
	if err != nil {
		return nil, fmt.Errorf("failed to init notifier: %w", err)
	}

	// Core Domain Service (The Business Logic)
	// Injecting all Secondary Adapters into the Core
	svc := service.New(k8sRepo, notifierAdapter, storageAdapter)

	// Ingress Servers (Primary Adapters)
	// Injecting the Core Service into the Servers
	grpcServer, err := grpc.NewServer(cfg.GrpcOptions, svc)
	if err != nil {
		return nil, fmt.Errorf("failed to init grpc server: %w", err)
	}
	mqttServer := mqtt.NewServer(mqttClient, topicBuilder, svc)
	httpServer := http.NewServer(cfg.HttpOptions)
	srvManager := server.NewManager(mqttServer, grpcServer, httpServer)

	return &CloudHubServer{
		serverManager: srvManager,
		k8sPipeline:   pipeline,
	}, nil
}
