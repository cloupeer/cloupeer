package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/autopeer-io/autopeer/internal/controller/vehicle"
	"github.com/autopeer-io/autopeer/internal/controller/vehiclecommand"
	iovv1alpha2 "github.com/autopeer-io/autopeer/pkg/apis/iov/v1alpha2"
	"github.com/autopeer-io/autopeer/pkg/log"
)

var autopeerScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(scheme.AddToScheme(autopeerScheme))
	utilruntime.Must(iovv1alpha2.AddToScheme(autopeerScheme))
}

type Controller interface {
	SetupWithManager(ctx context.Context, mgr ctrl.Manager) error
}

func NewControllerManager(ctx context.Context, kubeconfig *rest.Config, healthProbe string, metricsAddr string, hubAddr string) (manager.Manager, error) {
	mgr, err := ctrl.NewManager(kubeconfig, ctrl.Options{
		Scheme:                 autopeerScheme,
		Metrics:                server.Options{BindAddress: metricsAddr},
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

	if err := setupControllers(ctx, mgr, hubAddr); err != nil {
		return nil, err
	}

	return mgr, nil
}

// setupControllers initializes and registers all controllers with the manager.
func setupControllers(ctx context.Context, mgr manager.Manager, hubAddr string) error {
	cli := mgr.GetClient()
	sche := mgr.GetScheme()

	// EventRecorders for the controllers.
	vehicleRecorder := mgr.GetEventRecorderFor("cpeer-vehicle-controller")
	commandRecorder := mgr.GetEventRecorderFor("cpeer-command-controller")

	// Register Controllers
	controllers := []Controller{
		vehicle.NewReconciler(cli, sche, vehicleRecorder),
		vehiclecommand.NewReconciler(cli, sche, commandRecorder, hubAddr),
	}

	for _, ctl := range controllers {
		if err := ctl.SetupWithManager(ctx, mgr); err != nil {
			log.Error(err, "failed to setup controller", "ctl", ctl)
			return err
		}
	}

	return nil
}
