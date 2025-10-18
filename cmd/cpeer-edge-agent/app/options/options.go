package options

import (
	"fmt"
	"time"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	cliflag "k8s.io/component-base/cli/flag"

	"cloupeer.io/cloupeer/internal/edgeagent"
	"cloupeer.io/cloupeer/pkg/app"
	"cloupeer.io/cloupeer/pkg/log"
)

type AgentOptions struct {
	DeviceID          string
	GatewayURL        string
	HeartbeatInterval time.Duration
	VersionFile       string
	Log               *log.Options `json:"log" mapstructure:"log"`
}

var _ app.NamedFlagSetOptions = (*AgentOptions)(nil)

func NewAgentOptions() *AgentOptions {
	o := &AgentOptions{
		GatewayURL:        "http://localhost:9090",
		HeartbeatInterval: 15 * time.Second,
		VersionFile:       "/tmp/cloupeer_agent_version.txt",
		Log:               log.NewOptions(),
	}

	return o
}

func (o *AgentOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	fs := fss.FlagSet("Agent")
	fs.StringVar(&o.DeviceID, "device-id", o.DeviceID, "The unique identifier for this device. Must be provided.")
	fs.StringVar(&o.GatewayURL, "gateway-url", o.GatewayURL, "The URL of the cpeer-hub gateway.")
	fs.DurationVar(&o.HeartbeatInterval, "heartbeat-interval", o.HeartbeatInterval, "The interval at which the agent sends heartbeats.")
	fs.StringVar(&o.VersionFile, "version-file", o.VersionFile, "The path to the file that stores the current firmware version.")

	o.Log.AddFlags(fss.FlagSet("Log"))
	return fss
}

func (o *AgentOptions) Complete() error {
	// ...
	return nil
}

func (o *AgentOptions) Validate() error {
	errs := []error{}

	if o.DeviceID == "" {
		errs = append(errs, fmt.Errorf("--device-id is required"))
	}
	if o.GatewayURL == "" {
		errs = append(errs, fmt.Errorf("--gateway-url is required"))
	}

	errs = append(errs, o.Log.Validate()...)

	return utilerrors.NewAggregate(errs)
}

func (o *AgentOptions) Config() (*edgeagent.Config, error) {
	return &edgeagent.Config{
		DeviceID:          o.DeviceID,
		GatewayURL:        o.GatewayURL,
		HeartbeatInterval: o.HeartbeatInterval,
		VersionFile:       o.VersionFile,
	}, nil
}
