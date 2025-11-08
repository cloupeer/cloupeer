package vehicle

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	iovv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
)

type VehicleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewReconciler(cli client.Client, sche *runtime.Scheme) *VehicleReconciler {
	return &VehicleReconciler{
		Client: cli,
		Scheme: sche,
	}
}

// +kubebuilder:rbac:groups=iov.cloupeer.io,resources=vehicles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=iov.cloupeer.io,resources=vehicles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=iov.cloupeer.io,resources=vehicles/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *VehicleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Hello Vehicle ...")

	return ctrl.Result{}, nil
}

func (r *VehicleReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&iovv1alpha1.Vehicle{}).Complete(r)
}
