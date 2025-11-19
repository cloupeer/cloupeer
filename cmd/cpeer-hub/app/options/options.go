package options

import (
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	cliflag "k8s.io/component-base/cli/flag"

	"cloupeer.io/cloupeer/internal/hub"
	"cloupeer.io/cloupeer/pkg/app"
	"cloupeer.io/cloupeer/pkg/log"
)

type HubOptions struct {
	Namespace       string
	HttpAddr        string
	GrpcAddr        string
	MqttBroker      string
	MqttUsername    string
	MqttPassword    string
	MqttTopicPrefix string
	Log             *log.Options
}

var _ app.NamedFlagSetOptions = (*HubOptions)(nil)

func NewHubOptions() *HubOptions {
	o := &HubOptions{
		Namespace:       "cloupeer-system",
		HttpAddr:        ":8080",
		GrpcAddr:        ":8081",
		MqttBroker:      "tcp://emqx.cloupeer-system.svc:1883",
		MqttUsername:    "admin",
		MqttPassword:    "public",
		MqttTopicPrefix: "iov/cmd",
		Log:             log.NewOptions(),
	}

	return o
}

func (o *HubOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	// Add flags for Hub specific options
	fs := fss.FlagSet("Hub")
	fs.StringVar(&o.Namespace, "namespace", o.Namespace, "The Kubernetes namespace to watch for Cloupeer resources.")
	fs.StringVar(&o.HttpAddr, "http-addr", o.HttpAddr, "The address the cpeer-hub HTTP server should listen on.")
	fs.StringVar(&o.GrpcAddr, "grpc-addr", o.GrpcAddr, "The address the cpeer-hub gRPC server should listen on.")

	fs.StringVar(&o.MqttBroker, "mqtt-broker", o.MqttBroker, "The URL of the MQTT broker (e.g., tcp://emqx:1883).")
	fs.StringVar(&o.MqttUsername, "mqtt-username", o.MqttUsername, "The username for MQTT authentication.")
	fs.StringVar(&o.MqttPassword, "mqtt-password", o.MqttPassword, "The password for MQTT authentication.")
	fs.StringVar(&o.MqttTopicPrefix, "mqtt-topic-prefix", o.MqttTopicPrefix, "The topic prefix for command publishing.")

	// Add flags for logging
	o.Log.AddFlags(fss.FlagSet("Log"))
	return fss
}

func (o *HubOptions) Complete() error {
	return nil
}

func (o *HubOptions) Validate() error {
	errs := []error{}
	errs = append(errs, o.Log.Validate()...)
	return utilerrors.NewAggregate(errs)
}

func (o *HubOptions) Config() (*hub.Config, error) {
	return &hub.Config{
		Namespace:       o.Namespace,
		HttpAddr:        o.HttpAddr,
		GrpcAddr:        o.GrpcAddr,
		MqttBroker:      o.MqttBroker,
		MqttUsername:    o.MqttUsername,
		MqttPassword:    o.MqttPassword,
		MqttTopicPrefix: o.MqttTopicPrefix,
	}, nil
}
