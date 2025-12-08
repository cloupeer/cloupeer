package k8s

import (
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	iovv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
	"cloupeer.io/cloupeer/pkg/log"
	"cloupeer.io/cloupeer/pkg/options"
)

// NewClient creates a generic K8s client for CRD operations.
func NewClient(kubeconfigPath string) (controllerclient.Client, error) {
	var cfg *rest.Config
	var err error

	if kubeconfigPath == "" {
		// In-cluster config
		cfg, err = rest.InClusterConfig()
	} else {
		// Local development config
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	if err != nil {
		return nil, err
	}

	// Register our API types (Vehicle, VehicleCommand) into the scheme
	// Scheme is required by controller-runtime client to understand our CRDs.
	// scheme := v1alpha1.SchemeBuilder.GetScheme()
	cloupeerscheme := runtime.NewScheme()
	utilruntime.Must(scheme.AddToScheme(cloupeerscheme)) // Add standard schemes like v1.Pod, etc.
	utilruntime.Must(iovv1alpha1.AddToScheme(cloupeerscheme))

	c, err := controllerclient.New(cfg, controllerclient.Options{Scheme: cloupeerscheme})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func InitializeK8sClient(opts *options.KubeOptions) (controllerclient.Client, error) {
	k8sconfig, err := config.GetConfig()
	if err != nil {
		log.Error(err, "failed to get kubernetes config")
		return nil, err
	}

	// Create a new scheme and add all our API types and standard types
	cloupeerscheme := runtime.NewScheme()
	utilruntime.Must(scheme.AddToScheme(cloupeerscheme)) // Add standard schemes like v1.Pod, etc.
	utilruntime.Must(iovv1alpha1.AddToScheme(cloupeerscheme))

	k8sclient, err := controllerclient.New(k8sconfig, controllerclient.Options{Scheme: cloupeerscheme})
	if err != nil {
		log.Error(err, "failed to create kubernetes client")
		return nil, err
	}

	return k8sclient, nil
}
