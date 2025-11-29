package edgeagent

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"cloupeer.io/cloupeer/pkg/log"
	"cloupeer.io/cloupeer/pkg/mqtt"
	mqtttopic "cloupeer.io/cloupeer/pkg/mqtt/topic"
	"cloupeer.io/cloupeer/pkg/options"
)

type Config struct {
	// Identity
	VehicleID string

	MqttOptions *options.MqttOptions
}

func (cfg *Config) NewAgent() (*Agent, error) {
	if cfg.VehicleID == "" {
		return nil, fmt.Errorf("vehicle-id is required")
	}

	clientConfig := cfg.MqttOptions.ToClientConfig()

	if clientConfig.ClientID == "" {
		hostname, _ := os.Hostname()
		clientConfig.ClientID = fmt.Sprintf("cpeer-edge-agent-%s", hostname)
	}

	mqttclient, err := mqtt.NewClient(clientConfig)
	if err != nil {
		log.Error(err, "failed to new mqtt client")
		return nil, err
	}

	topicbuilder := mqtttopic.NewTopicBuilder(cfg.MqttOptions.TopicRoot)

	httpClient := &http.Client{
		Timeout: 10 * time.Minute,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return &Agent{
		vehicleID:       cfg.VehicleID,
		mqttclient:      mqttclient,
		topicbuilder:    topicbuilder,
		httpClient:      httpClient,
		pendingRequests: make(map[string]chan string),
	}, nil
}
