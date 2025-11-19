package options

import (
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	cliflag "k8s.io/component-base/cli/flag"

	"cloupeer.io/cloupeer/internal/edgeagent"
	"cloupeer.io/cloupeer/pkg/app"
	"cloupeer.io/cloupeer/pkg/log"
)

type AgentOptions struct {
	VehicleID       string
	MqttBroker      string
	MqttUsername    string
	MqttPassword    string
	MqttTopicPrefix string
	Log             *log.Options `json:"log" mapstructure:"log"`
}

var _ app.NamedFlagSetOptions = (*AgentOptions)(nil)

func NewAgentOptions() *AgentOptions {
	o := &AgentOptions{
		VehicleID:       "vh-001",
		MqttBroker:      "tcp://emqx.cloupeer-system.svc:1883",
		MqttUsername:    "admin",
		MqttPassword:    "public",
		MqttTopicPrefix: "iov/cmd",
		Log:             log.NewOptions(),
	}

	return o
}

func (o *AgentOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	fs := fss.FlagSet("Agent")
	fs.StringVar(&o.VehicleID, "vehicle-id", o.VehicleID, "The unique ID of this vehicle.")
	fs.StringVar(&o.MqttBroker, "mqtt-broker", o.MqttBroker, "MQTT broker address.")
	fs.StringVar(&o.MqttUsername, "mqtt-username", o.MqttUsername, "MQTT username.")
	fs.StringVar(&o.MqttPassword, "mqtt-password", o.MqttPassword, "MQTT password.")
	fs.StringVar(&o.MqttTopicPrefix, "mqtt-topic-prefix", o.MqttTopicPrefix, "Topic prefix for subscribing commands.")

	o.Log.AddFlags(fss.FlagSet("Log"))
	return fss
}

func (o *AgentOptions) Complete() error {
	// ...
	return nil
}

func (o *AgentOptions) Validate() error {
	errs := []error{}

	errs = append(errs, o.Log.Validate()...)

	return utilerrors.NewAggregate(errs)
}

func (o *AgentOptions) Config() (*edgeagent.Config, error) {
	return &edgeagent.Config{
		VehicleID:       o.VehicleID,
		MqttBroker:      o.MqttBroker,
		MqttUsername:    o.MqttUsername,
		MqttPassword:    o.MqttPassword,
		MqttTopicPrefix: o.MqttTopicPrefix,
	}, nil
}
