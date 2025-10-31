package options

import (
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	cliflag "k8s.io/component-base/cli/flag"

	"cloupeer.io/cloupeer/internal/edgeagent"
	"cloupeer.io/cloupeer/pkg/app"
	"cloupeer.io/cloupeer/pkg/log"
)

type AgentOptions struct {
	Log *log.Options `json:"log" mapstructure:"log"`
}

var _ app.NamedFlagSetOptions = (*AgentOptions)(nil)

func NewAgentOptions() *AgentOptions {
	o := &AgentOptions{
		Log: log.NewOptions(),
	}

	return o
}

func (o *AgentOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	// fs := fss.FlagSet("Agent")

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
	return &edgeagent.Config{}, nil
}
