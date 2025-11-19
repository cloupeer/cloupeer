package hub

import (
	"net/http"
	"time"
)

type Config struct {
	Namespace string
	HttpAddr  string // HTTP Address (e.g., :8080)
	GrpcAddr  string // gRPC Address (e.g., :8081)

	// MQTT Configuration
	MqttBroker      string
	MqttUsername    string
	MqttPassword    string
	MqttTopicPrefix string
}

func (cfg *Config) NewHubServer() (*HubServer, error) {
	return &HubServer{
		namespace:       cfg.Namespace,
		httpAddr:        cfg.HttpAddr,
		grpcAddr:        cfg.GrpcAddr,
		mqttBroker:      cfg.MqttBroker,
		mqttUsername:    cfg.MqttUsername,
		mqttPassword:    cfg.MqttPassword,
		mqttTopicPrefix: cfg.MqttTopicPrefix,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
	}, nil
}
