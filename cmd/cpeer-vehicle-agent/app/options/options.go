package options

import (
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	cliflag "k8s.io/component-base/cli/flag"

	"github.com/autopeer-io/autopeer/internal/vehicleagent"
	"github.com/autopeer-io/autopeer/pkg/app"
	"github.com/autopeer-io/autopeer/pkg/log"
	"github.com/autopeer-io/autopeer/pkg/options"
)

type AgentOptions struct {
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
	return &vehicleagent.Config{
		MqttOptions: o.MqttOptions,
	}, nil
}
