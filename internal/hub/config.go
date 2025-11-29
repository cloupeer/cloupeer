package hub

import (
	"cloupeer.io/cloupeer/internal/hub/storage"
	mqtttopic "cloupeer.io/cloupeer/pkg/mqtt/topic"
	"cloupeer.io/cloupeer/pkg/options"
)

type Config struct {
	KubeOptions *options.KubeOptions
	HttpOptions *options.HttpOptions
	GrpcOptions *options.GrpcOptions
	MqttOptions *options.MqttOptions
	S3Options   *options.S3Options
}

func (cfg *Config) NewHubServer() (*HubServer, error) {
	k8sclient, err := InitializeK8sClient(cfg.KubeOptions)
	if err != nil {
		return nil, err
	}

	mqttclient, err := InitializeMQTTClient(cfg.MqttOptions)
	if err != nil {
		return nil, err
	}

	topicbuilder := mqtttopic.NewTopicBuilder(cfg.MqttOptions.TopicRoot)

	// 初始化存储 Provider
	storageProvider, err := storage.NewMinIOProvider(cfg.S3Options)
	if err != nil {
		return nil, err
	}

	grpcserver, err := cfg.NewGrpcServer(mqttclient, topicbuilder)
	if err != nil {
		return nil, err
	}

	return &HubServer{
		namespace:    cfg.KubeOptions.Namespace,
		httpserver:   cfg.NewHttpServer(),
		grpcserver:   grpcserver,
		k8sclient:    k8sclient,
		mqttclient:   mqttclient,
		topicbuilder: topicbuilder,
		storage:      storageProvider,
	}, nil
}
