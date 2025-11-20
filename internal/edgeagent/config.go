package edgeagent

import (
	"fmt"
	"net/http"
	"time"
)

type Config struct {
	// Identity
	VehicleID string

	// MQTT Config
	MqttBroker      string
	MqttUsername    string
	MqttPassword    string
	MqttTopicPrefix string
}

func (cfg *Config) NewAgent() (*Agent, error) {
	if cfg.VehicleID == "" {
		return nil, fmt.Errorf("vehicle-id is required")
	}

	return &Agent{
		vehicleID:       cfg.VehicleID,
		mqttBroker:      cfg.MqttBroker,
		mqttUsername:    cfg.MqttUsername,
		mqttPassword:    cfg.MqttPassword,
		mqttTopicPrefix: cfg.MqttTopicPrefix,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
		pendingRequests: make(map[string]chan string),
	}, nil
}
