package cloudhub

import (
	"fmt"
	"os"

	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	iovv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
	"cloupeer.io/cloupeer/pkg/log"
	"cloupeer.io/cloupeer/pkg/mqtt"
	"cloupeer.io/cloupeer/pkg/options"
)

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

func InitializeMQTTClient(opts *options.MqttOptions) (mqtt.Client, error) {
	cfg := opts.ToClientConfig()

	if cfg.ClientID == "" {
		hostname, _ := os.Hostname()
		cfg.ClientID = fmt.Sprintf("cpeer-hub-%s", hostname)
	}

	mqttclient, err := mqtt.NewClient(cfg)
	if err != nil {
		log.Error(err, "failed to new mqtt client")
		return nil, err
	}

	return mqttclient, nil
}
