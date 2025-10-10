// Copyright 2025 Anankix.

package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	edgev1alpha1 "github.com/anankix/anankix/pkg/apis/edge/v1alpha1"
)

// FirmwareUpgradeTaskReconciler reconciles a FirmwareUpgradeTask object
type FirmwareUpgradeTaskReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=edge.anankix,resources=firmwareupgradetasks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=edge.anankix,resources=firmwareupgradetasks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=edge.anankix,resources=firmwareupgradetasks/finalizers,verbs=update
//+kubebuilder:rbac:groups=edge.anankix,resources=physicaldevices,verbs=get;list;watch
//+kubebuilder:rbac:groups=edge.anankix,resources=physicaldevices/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *FirmwareUpgradeTaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling FirmwareUpgradeTask")

	// 1. Fetch the FirmwareUpgradeTask instance
	var task edgev1alpha1.FirmwareUpgradeTask
	if err := r.Get(ctx, req.NamespacedName, &task); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("FirmwareUpgradeTask resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get FirmwareUpgradeTask")
		return ctrl.Result{}, err
	}

	// 2. Check if the task is in a terminal phase (Succeeded or Failed)
	// If not, we don't need to do anything yet. The agent/gateway will update it.
	if task.Status.Phase != edgev1alpha1.TaskPhaseSucceeded && task.Status.Phase != edgev1alpha1.TaskPhaseFailed {
		logger.Info("Task is not in a terminal state yet, skipping status update.", "taskPhase", task.Status.Phase)
		// We can add a timeout check here in a more advanced implementation
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	// 3. The task is finished. Find the parent PhysicalDevice to update its status.
	// We find it by its deviceId, which is more robust than OwnerReferences in some edge cases.
	var deviceList edgev1alpha1.PhysicalDeviceList
	if err := r.List(ctx, &deviceList,
		client.InNamespace(task.Namespace),
		client.MatchingFields{"spec.deviceId": task.Spec.DeviceID},
	); err != nil {
		logger.Error(err, "Failed to list PhysicalDevices to find owner of task")
		return ctrl.Result{}, err
	}

	if len(deviceList.Items) == 0 {
		logger.Info("No PhysicalDevice found for this task's deviceId. The device may have been deleted.", "deviceId", task.Spec.DeviceID)
		// No device, nothing to update.
		return ctrl.Result{}, nil
	}

	// Normally there should be only one device with a given ID in a namespace.
	pd := deviceList.Items[0]
	logger.Info("Found parent PhysicalDevice", "device", pd.Name)

	// 4. Update the PhysicalDevice's status if it's still tracking this task.
	if pd.Status.FirmwareUpgradeStatus == nil || pd.Status.FirmwareUpgradeStatus.TaskRef != task.Name {
		logger.Info("PhysicalDevice is not tracking this task anymore. Ignoring.", "trackingTask", pd.Status.FirmwareUpgradeStatus.TaskRef)
		return ctrl.Result{}, nil
	}

	patch := client.MergeFrom(pd.DeepCopy())

	if task.Status.Phase == edgev1alpha1.TaskPhaseSucceeded {
		pd.Status.FirmwareUpgradeStatus.State = "Succeeded"
		pd.Status.FirmwareUpgradeStatus.LastSuccessfulVersion = task.Spec.Firmware.Version
		pd.Status.FirmwareUpgradeStatus.LastFailureMessage = ""
	} else { // TaskPhaseFailed
		pd.Status.FirmwareUpgradeStatus.State = "Failed"
		pd.Status.FirmwareUpgradeStatus.LastFailureMessage = fmt.Sprintf("Task %s failed: %s", task.Name, task.Status.Message)
	}

	if err := r.Status().Patch(ctx, &pd, patch); err != nil {
		logger.Error(err, "Failed to update PhysicalDevice status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully updated PhysicalDevice status based on completed task.", "device", pd.Name, "finalState", pd.Status.FirmwareUpgradeStatus.State)

	// The reconciliation is complete for this task. We don't need to requeue.
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FirmwareUpgradeTaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// We need to add an index on `spec.deviceId` for PhysicalDevice.
	// This allows us to efficiently look up devices by their ID, which we do in the Reconcile loop.
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &edgev1alpha1.PhysicalDevice{}, "spec.deviceId", func(rawObj client.Object) []string {
		pd := rawObj.(*edgev1alpha1.PhysicalDevice)
		return []string{pd.Spec.DeviceID}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&edgev1alpha1.FirmwareUpgradeTask{}).
		Complete(r)
}
