package options

import (
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	cliflag "k8s.io/component-base/cli/flag"

	"cloupeer.io/cloupeer/internal/vehicleagent"
	"cloupeer.io/cloupeer/pkg/app"
	"cloupeer.io/cloupeer/pkg/log"
	"cloupeer.io/cloupeer/pkg/options"
)

type AgentOptions struct {
	VehicleID   string               `json:"id" mapstructure:"id"`
	MqttOptions *options.MqttOptions `json:"mqtt" mapstructure:"mqtt"`
	Log         *log.Options         `json:"log" mapstructure:"log"`
}

var _ app.NamedFlagSetOptions = (*AgentOptions)(nil)

func NewAgentOptions() *AgentOptions {
	o := &AgentOptions{
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

func (o *AgentOptions) Config() (*vehicleagent.Config, error) {
	if o.VehicleID == "" {
		o.VehicleID = vehicleagent.DiscoverVehicleID()
	}

	return &vehicleagent.Config{
		VehicleID:   o.VehicleID,
		MqttOptions: o.MqttOptions,
	}, nil
}
