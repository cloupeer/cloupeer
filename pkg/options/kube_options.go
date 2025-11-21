package options

import (
	"github.com/spf13/pflag"
)

var _ IOptions = (*KubeOptions)(nil)

// KubeOptions contains configuration for Kubernetes client interactions.
type KubeOptions struct {
	// Namespace is the Kubernetes namespace to watch or operate in.
	// Default is usually "cloupeer-system" or extracted from the pod environment.
	Namespace string `json:"namespace" mapstructure:"namespace"`

	// KubeConfig is the path to the kubeconfig file.
	// If empty, it defaults to in-cluster config or standard KUBECONFIG env.
	KubeConfig string `json:"kubeconfig" mapstructure:"kubeconfig"`

	// Future extensions:
	// QPS     float32
	// Burst   int
}

// NewKubeOptions creates a new KubeOptions with default values.
func NewKubeOptions() *KubeOptions {
	return &KubeOptions{
		Namespace:  "cloupeer-system",
		KubeConfig: "", // Default to empty, letting client-go resolve it automatically
	}
}

// Validate is used to parse and validate the parameters entered by the user at
// the command line when the program starts.
func (o *KubeOptions) Validate() []error {
	if o == nil {
		return nil
	}

	errors := []error{}

	return errors
}

// AddFlags adds flags for KubeOptions to the specified FlagSet.
func (o *KubeOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Namespace, "kube.namespace", o.Namespace, "The Kubernetes namespace to watch or operate in.")
	fs.StringVar(&o.KubeConfig, "kube.kubeconfig", o.KubeConfig, "Path to kubeconfig file with authorization and master location information.")
}
