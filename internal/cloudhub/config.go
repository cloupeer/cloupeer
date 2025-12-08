package cloudhub

import (
	"fmt"

	"cloupeer.io/cloupeer/internal/cloudhub/core/service"
	"cloupeer.io/cloupeer/internal/cloudhub/k8s"
	"cloupeer.io/cloupeer/internal/cloudhub/notifier"
	"cloupeer.io/cloupeer/internal/cloudhub/server"
	"cloupeer.io/cloupeer/internal/cloudhub/storage"
	"cloupeer.io/cloupeer/pkg/options"
)

type Config struct {
	KubeOptions *options.KubeOptions
	HttpOptions *options.HttpOptions
	GrpcOptions *options.GrpcOptions
	MqttOptions *options.MqttOptions
	S3Options   *options.S3Options
}

func (cfg *Config) NewHubServer() (*CloudHubServer, error) {
	k8sClient, err := k8s.InitializeK8sClient(cfg.KubeOptions)
	if err != nil {
		return nil, err
	}

	pipeline := k8s.NewPipeline(cfg.KubeOptions.Namespace, k8sClient)
	// k8sRepo implements both VehicleRepository and CommandRepository
	k8sRepo := k8s.NewRepository(cfg.KubeOptions.Namespace, k8sClient, pipeline)

	// 2. Infrastructure: Storage (Secondary Adapter)
	// 初始化存储 Provider
	storageAdapter, err := storage.NewMinIO(cfg.S3Options)
	if err != nil {
		return nil, err
	}

	// 3. Infrastructure: Notifier (Secondary Adapter)
	notifierAdapter, err := notifier.NewMQTTNotifier(cfg.MqttOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to init notifier: %w", err)
	}

	// 4. Core Domain Service (The Business Logic)
	// Injecting all Secondary Adapters into the Core
	svc := service.New(k8sRepo, k8sRepo, notifierAdapter, storageAdapter)

	// 5. Ingress Servers (Primary Adapters)
	// Injecting the Core Service into the Servers
	serverConfig := &server.Config{
		HttpOptions: cfg.HttpOptions,
		GrpcOptions: cfg.GrpcOptions,
		MqttOptions: cfg.MqttOptions,
	}
	srvManager, err := server.NewManager(serverConfig, svc)
	if err != nil {
		return nil, fmt.Errorf("failed to init server manager: %w", err)
	}

	return &CloudHubServer{
		serverManager: srvManager,
		k8sPipeline:   pipeline,
	}, nil
}
