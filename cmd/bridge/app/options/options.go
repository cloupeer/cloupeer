package options

import (
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	cliflag "k8s.io/component-base/cli/flag"

	"github.com/autopeer-io/autopeer/internal/bridge"
	"github.com/autopeer-io/autopeer/pkg/app"
	"github.com/autopeer-io/autopeer/pkg/log"
	"github.com/autopeer-io/autopeer/pkg/options"
)

type HubOptions struct {
	KubeOptions *options.KubeOptions `json:"kube" mapstructure:"kube"`
	HttpOptions *options.HttpOptions `json:"http" mapstructure:"http"`
	GrpcOptions *options.GrpcOptions `json:"grpc" mapstructure:"grpc"`
	MqttOptions *options.MqttOptions `json:"mqtt" mapstructure:"mqtt"`
	S3Options   *options.S3Options   `json:"s3" mapstructure:"s3"`
	Log         *log.Options
}

var _ app.NamedFlagSetOptions = (*HubOptions)(nil)

func NewHubOptions() *HubOptions {
	o := &HubOptions{
		KubeOptions: options.NewKubeOptions(),
		HttpOptions: options.NewHttpOptions(),
		GrpcOptions: options.NewGrpcOptions(),
		MqttOptions: options.NewMqttOptions(),
		S3Options:   options.NewS3Options(),
		Log:         log.NewOptions(),
	}

	return o
}

func (o *HubOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}
	o.KubeOptions.AddFlags(fss.FlagSet("kube"))
	o.HttpOptions.AddFlags(fss.FlagSet("http"))
	o.GrpcOptions.AddFlags(fss.FlagSet("grpc"))
	o.MqttOptions.AddFlags(fss.FlagSet("mqtt"))
	o.S3Options.AddFlags(fss.FlagSet("s3"))
	o.Log.AddFlags(fss.FlagSet("log"))
	return fss
}

func (o *HubOptions) Complete() error {
	return nil
}

func (o *HubOptions) Validate() error {
	errs := []error{}
	errs = append(errs, o.KubeOptions.Validate()...)
	errs = append(errs, o.HttpOptions.Validate()...)
	errs = append(errs, o.GrpcOptions.Validate()...)
	errs = append(errs, o.MqttOptions.Validate()...)
	errs = append(errs, o.S3Options.Validate()...)
	errs = append(errs, o.Log.Validate()...)
	return utilerrors.NewAggregate(errs)
}

func (o *HubOptions) Config() (*bridge.Config, error) {
	return &bridge.Config{
		KubeOptions: o.KubeOptions,
		HttpOptions: o.HttpOptions,
		GrpcOptions: o.GrpcOptions,
		MqttOptions: o.MqttOptions,
		S3Options:   o.S3Options,
	}, nil
}
