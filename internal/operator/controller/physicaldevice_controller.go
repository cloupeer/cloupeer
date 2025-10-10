/*
Copyright 2025 Anankix.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	edgev1alpha1 "github.com/anankix/anankix/pkg/apis/edge/v1alpha1"
)

// PhysicalDeviceReconciler reconciles a PhysicalDevice object
type PhysicalDeviceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=edge.anankix,resources=physicaldevices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=edge.anankix,resources=physicaldevices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=edge.anankix,resources=physicaldevices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PhysicalDevice object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *PhysicalDeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// 1. Get a logger with context, this is a best practice.
	logger := log.FromContext(ctx)
	logger.Info("Reconciling PhysicalDevice", "Request.Namespace", req.Namespace, "Request.Name", req.Name)

	// 1. Fetch the PhysicalDevice object
	var pd edgev1alpha1.PhysicalDevice
	if err := r.Get(ctx, req.NamespacedName, &pd); err != nil {
		if errors.IsNotFound(err) {
			// 如果错误是 IsNotFound，意味着对象已经被删除了。
			// 这种情况通常发生在用户执行了 `kubectl delete` 之后。
			// 对于删除事件，我们通常不需要做任何事（因为 OwnerReference 会自动清理子资源），
			// 所以记录一条日志然后正常返回即可。
			logger.Info("PhysicalDevice resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		// 如果是其他类型的错误（比如网络问题、权限问题），
		// 我们应该记录错误并返回 err，这样 controller-runtime 会稍后自动重试。
		logger.Error(err, "Failed to get PhysicalDevice")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully fetched PhysicalDevice", "DeviceID", pd.Spec.DeviceID, "ReportedVersion", pd.Status.CurrentFirmwareVersion)

	// 2. Check if firmware upgrade is needed
	// If spec.firmware is nil, do nothing and return.
	if pd.Spec.Firmware == nil {
		// Potentially clean up status if needed
		return ctrl.Result{}, nil
	}

	// If reported version matches desired version, the work is done.
	if pd.Status.CurrentFirmwareVersion == pd.Spec.Firmware.Version {
		// Update status to Succeeded if it was InProgress
		if pd.Status.FirmwareUpgradeStatus != nil && pd.Status.FirmwareUpgradeStatus.State == "InProgress" {
			pd.Status.FirmwareUpgradeStatus.State = "Succeeded"
			pd.Status.FirmwareUpgradeStatus.LastSuccessfulVersion = pd.Spec.Firmware.Version
			// Update status and return
		}
		return ctrl.Result{}, nil
	}

	// 3. An upgrade is needed. Find or Create a FirmwareUpgradeTask.
	// We'll name the task predictably, e.g., "pd.Name-pd.Spec.Firmware.Version"
	taskName := fmt.Sprintf("%s-%s", pd.Name, pd.Spec.Firmware.Version)
	var task edgev1alpha1.FirmwareUpgradeTask
	err := r.Get(ctx, types.NamespacedName{Name: taskName, Namespace: pd.Namespace}, &task)
	if err != nil && errors.IsNotFound(err) {
		// 4. Task does not exist, so create it.
		logger.Info("Creating a new FirmwareUpgradeTask", "taskName", taskName)
		newTask := &edgev1alpha1.FirmwareUpgradeTask{
			ObjectMeta: metav1.ObjectMeta{
				Name:      taskName,
				Namespace: pd.Namespace,
				// CRITICAL: Set OwnerReference so the task is garbage collected if the device is deleted.
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(&pd, edgev1alpha1.GroupVersion.WithKind("PhysicalDevice")),
				},
				Labels: map[string]string{
					"edge.anankix/deviceId": pd.Spec.DeviceID, // Label for easy lookup
				},
			},
			Spec: edgev1alpha1.FirmwareUpgradeTaskSpec{
				DeviceID: pd.Spec.DeviceID,
				Firmware: *pd.Spec.Firmware,
			},
		}
		if err := r.Create(ctx, newTask); err != nil {
			return ctrl.Result{}, err
		}

		// 5. Update PhysicalDevice status to reflect that an upgrade is in progress.
		pd.Status.FirmwareUpgradeStatus = &edgev1alpha1.UpgradeStatus{
			TaskRef:         taskName,
			State:           "InProgress",
			LastAttemptTime: &metav1.Time{Time: time.Now()},
		}
		if err := r.Status().Update(ctx, &pd); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil

	} else if err != nil {
		// Handle other errors
		return ctrl.Result{}, err
	}

	// 6. Task already exists. We can inspect its status if needed, but for now, we assume it's being processed.
	// The PhysicalDevice status is already "InProgress", so we just wait.
	// We could add timeout logic here in a more advanced implementation.
	logger.Info("FirmwareUpgradeTask already exists, waiting for completion.", "taskName", taskName)
	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PhysicalDeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&edgev1alpha1.PhysicalDevice{}).
		Complete(r)
}
