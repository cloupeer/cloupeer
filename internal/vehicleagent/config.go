package vehicleagent

import (
	"encoding/json"
	"fmt"

	pb "github.com/autopeer-io/autopeer/api/proto/v1"
	"github.com/autopeer-io/autopeer/internal/pkg/mqtt/paths"
	"github.com/autopeer-io/autopeer/internal/vehicleagent/hal"
	"github.com/autopeer-io/autopeer/internal/vehicleagent/hub"
	"github.com/autopeer-io/autopeer/internal/vehicleagent/ota"
	"github.com/autopeer-io/autopeer/pkg/mqtt"
	mqtttopic "github.com/autopeer-io/autopeer/pkg/mqtt/topic"
	"github.com/autopeer-io/autopeer/pkg/options"
)

type Config struct {
	MqttOptions *options.MqttOptions
}

func (cfg *Config) NewAgent() (*Agent, error) {
	var vid string
	systemHAL := hal.NewHAL()

	if vid = systemHAL.GetVehicleID(); vid == "" {
		return nil, fmt.Errorf("FATAL: unable to retrieve VehicleID from HAL")
	}

	mqttClient, topicBuilder, err := cfg.initMqttClientAndTopicBuilder(vid)
	if err != nil {
		return nil, fmt.Errorf("failed to init mqtt client")
	}

	return NewAgent(
		systemHAL,
		hub.New(vid, mqttClient, topicBuilder),
		ota.NewManager(vid),
	), nil
}

func (cfg *Config) initMqttClientAndTopicBuilder(vid string) (mqtt.Client, *mqtttopic.Builder, error) {
	topicBuilder := mqtttopic.NewBuilder(cfg.MqttOptions.TopicRoot)

	mqttConfig := cfg.MqttOptions.ToClientConfig()
	if mqttConfig.ClientID == "" {
		mqttConfig.ClientID = fmt.Sprintf("cpeer-agent-%s", vid)
	}

	// We rely on Hub's reception time, so no timestamp in payload to avoid LWT staleness.
	offlinePayload, _ := json.Marshal(pb.OnlineStatus{
		VehicleId: vid,
		Online:    false,
		Reason:    "UnexpectedDisconnect",
	})

	mqttConfig.WillTopic = topicBuilder.Build(paths.Online, vid)
	mqttConfig.WillPayload = offlinePayload
	mqttConfig.WillQoS = 1
	mqttConfig.WillRetain = true

	mqttClient, err := mqtt.NewClient(mqttConfig)
	if err != nil {
		return nil, nil, err
	}

	return mqttClient, topicBuilder, nil
}
