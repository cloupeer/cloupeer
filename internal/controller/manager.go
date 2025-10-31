package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"cloupeer.io/cloupeer/pkg/log"
)

var cloupeerScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(scheme.AddToScheme(cloupeerScheme))
}

type Controller interface {
	SetupWithManager(ctx context.Context, mgr controllerruntime.Manager) error
}

func NewControllerManager(ctx context.Context, kubeconfig *rest.Config, healthProbe string) (manager.Manager, error) {
	mgr, err := controllerruntime.NewManager(kubeconfig, controllerruntime.Options{
		Scheme:                 cloupeerScheme,
		Metrics:                server.Options{BindAddress: "0"},
		HealthProbeBindAddress: healthProbe,
	})
	if err != nil {
		log.Error(err, "failed to create controller manager")
		return nil, err
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up health check")
		return nil, err
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up ready check")
		return nil, err
	}

	if err := setupControllers(ctx, mgr); err != nil {
		return nil, err
	}

	return mgr, nil
}

func setupControllers(ctx context.Context, mgr manager.Manager) error {
	// cli := mgr.GetClient()
	// sche := mgr.GetScheme()

	controllers := []Controller{}

	for _, ctl := range controllers {
		if err := ctl.SetupWithManager(ctx, mgr); err != nil {
			log.Error(err, "failed to setup controller", "ctl", ctl)
			return err
		}
	}

	return nil
}
