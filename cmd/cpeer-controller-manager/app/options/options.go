package options

import (
	cliflag "k8s.io/component-base/cli/flag"

	"cloupeer.io/cloupeer/pkg/log"
)

type ControllerManagerOptions struct {
	ConcurrentReconciles   int
	HealthProbeBindAddress string
	MetricsBindAddress     string
	HubAddr                string
	FeatureGates           []string
	LogOptions             *log.Options
}

func NewControllerManagerOptions() *ControllerManagerOptions {
	return &ControllerManagerOptions{
		ConcurrentReconciles:   5,
		HealthProbeBindAddress: ":9001",
		MetricsBindAddress:     ":8080",
		HubAddr:                "cpeer-hub.cloupeer-system.svc:8091",
		LogOptions:             log.NewOptions(),
	}
}

func (o *ControllerManagerOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("Controller Manager")
	fs.IntVar(&o.ConcurrentReconciles, "concurrent-reconciles", o.ConcurrentReconciles, "The number of concurrent reconciles.")
	fs.StringVar(&o.HealthProbeBindAddress, "health-probe-bind-address", o.HealthProbeBindAddress, "The TCP address that the controller should bind to for serving health probes.")
	fs.StringVar(&o.MetricsBindAddress, "metrics-bind-address", o.MetricsBindAddress, "The TCP address that the controller should bind to for serving prometheus metrics.")
	fs.StringVar(&o.HubAddr, "hub-addr", o.HubAddr, "The gRPC address of the Cloupeer Hub.")
	fs.StringArrayVar(&o.FeatureGates, "feature-gates", o.FeatureGates, "Used to enable some features.")

	o.LogOptions.AddFlags(fss.FlagSet("Log"))

	return fss
}
