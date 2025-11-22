package options

import (
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	cliflag "k8s.io/component-base/cli/flag"

	"cloupeer.io/cloupeer/internal/edgeagent"
	"cloupeer.io/cloupeer/pkg/app"
	"cloupeer.io/cloupeer/pkg/log"
	"cloupeer.io/cloupeer/pkg/options"
)

type AgentOptions struct {
	VehicleID   string
	MqttOptions *options.MqttOptions `json:"mqtt" mapstructure:"mqtt"`
	Log         *log.Options         `json:"log" mapstructure:"log"`
}

var _ app.NamedFlagSetOptions = (*AgentOptions)(nil)

func NewAgentOptions() *AgentOptions {
	o := &AgentOptions{
		VehicleID:   "vh-001",
		MqttOptions: options.NewMqttOptions(),
		Log:         log.NewOptions(),
	}

	return o
}

func (o *AgentOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	fs := fss.FlagSet("Agent")
	fs.StringVar(&o.VehicleID, "vehicle-id", o.VehicleID, "The unique ID of this vehicle.")

	o.MqttOptions.AddFlags(fss.FlagSet("mqtt"))
	o.Log.AddFlags(fss.FlagSet("Log"))
	return fss
}

func (o *AgentOptions) Complete() error {
	return nil
}

func (o *AgentOptions) Validate() error {
	errs := []error{}
	errs = append(errs, o.MqttOptions.Validate()...)
	errs = append(errs, o.Log.Validate()...)
	return utilerrors.NewAggregate(errs)
}

func (o *AgentOptions) Config() (*edgeagent.Config, error) {
	return &edgeagent.Config{
		VehicleID:   o.VehicleID,
		MqttOptions: o.MqttOptions,
	}, nil
}
