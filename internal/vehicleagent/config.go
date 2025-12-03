package vehicleagent

import (
	"fmt"
	"os"

	"cloupeer.io/cloupeer/internal/vehicleagent/hub"
	"cloupeer.io/cloupeer/internal/vehicleagent/ota"
	"cloupeer.io/cloupeer/pkg/mqtt"
	mqtttopic "cloupeer.io/cloupeer/pkg/mqtt/topic"
	"cloupeer.io/cloupeer/pkg/options"
)

type Config struct {
	VehicleID   string
	MqttOptions *options.MqttOptions
}

func (cfg *Config) NewAgent() (*Agent, error) {
	if cfg.VehicleID == "" {
		return nil, fmt.Errorf("vehicle-id is required")
	}

	clientConfig := cfg.MqttOptions.ToClientConfig()
	if clientConfig.ClientID == "" {
		hostname, _ := os.Hostname()
		clientConfig.ClientID = fmt.Sprintf("cpeer-agent-%s", hostname)
	}

	mqttClient, err := mqtt.NewClient(clientConfig)
	if err != nil {
		return nil, err
	}

	topicBuilder := mqtttopic.NewBuilder(cfg.MqttOptions.TopicRoot)

	msgHub := hub.New(mqttClient, topicBuilder, cfg.VehicleID)

	ota.Register(cfg.VehicleID)

	return NewAgent(cfg.VehicleID, msgHub), nil
}
