package firmwareupgrade

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	firmwarev1alpha1 "cloupeer.io/cloupeer/pkg/apis/firmware/v1alpha1"
	iotv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iot/v1alpha1"
)

const firmwareUpgradeFinalizer = "firmware.cloupeer.io/finalizer"

type FirmwareUpgradeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewFirmwareUpgradeReconciler(cli client.Client, sche *runtime.Scheme) *FirmwareUpgradeReconciler {
	return &FirmwareUpgradeReconciler{Client: cli, Scheme: sche}
}

//+kubebuilder:rbac:groups=firmware.cloupeer.io,resources=firmwareupgrades,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=firmware.cloupeer.io,resources=firmwareupgrades/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=firmware.cloupeer.io,resources=firmwareupgrades/finalizers,verbs=update
//+kubebuilder:rbac:groups=iot.cloupeer.io,resources=devices,verbs=get;list;watch;update;patch

func (r *FirmwareUpgradeReconciler) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the FirmwareUpgrade instance
	var upgradeTask firmwarev1alpha1.FirmwareUpgrade
	if err := r.Get(ctx, req.NamespacedName, &upgradeTask); err != nil {
		return controllerruntime.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion: check if the object is being deleted
	if !upgradeTask.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&upgradeTask, firmwareUpgradeFinalizer) {
			// Our finalizer is present, so let's handle any external cleanup.
			// In our case, we might want to notify devices to cancel the upgrade.
			// For this example, we'll just log it.
			log.Info("Performing cleanup for FirmwareUpgrade before deletion.")

			// Remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&upgradeTask, firmwareUpgradeFinalizer)
			if err := r.Update(ctx, &upgradeTask); err != nil {
				return controllerruntime.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return controllerruntime.Result{}, nil
	}

	// Add finalizer for this CR if it doesn't exist
	if !controllerutil.ContainsFinalizer(&upgradeTask, firmwareUpgradeFinalizer) {
		controllerutil.AddFinalizer(&upgradeTask, firmwareUpgradeFinalizer)
		if err := r.Update(ctx, &upgradeTask); err != nil {
			return controllerruntime.Result{}, err
		}
		// Requeue because we've updated the object, the new reconcile will have the finalizer
		return controllerruntime.Result{}, nil
	}

	if upgradeTask.Status.Phase == "" {
		log.Info("New FirmwareUpgrade task detected, initializing to Pending state.")

		// Find all target devices to calculate the 'Total' count.
		var deviceList iotv1alpha1.DeviceList
		selector, err := metav1.LabelSelectorAsSelector(upgradeTask.Spec.DeviceSelector)
		if err != nil {
			log.Error(err, "Invalid DeviceSelector, marking task as Failed")
			upgradeTask.Status.Phase = firmwarev1alpha1.UpgradePhaseFailed
			// You could add a Condition here to record the error message.
			return controllerruntime.Result{}, r.Status().Update(ctx, &upgradeTask)
		}

		if err := r.List(ctx, &deviceList, &client.ListOptions{LabelSelector: selector}); err != nil {
			log.Error(err, "Failed to list target devices")
			return controllerruntime.Result{}, err
		}

		// Use Patch to initialize the status.
		patch := client.MergeFrom(upgradeTask.DeepCopy())
		upgradeTask.Status.Phase = firmwarev1alpha1.UpgradePhasePending // Set to Pending, not Upgrading!
		upgradeTask.Status.Total = int32(len(deviceList.Items))
		upgradeTask.Status.Succeeded = 0
		upgradeTask.Status.Failed = 0
		upgradeTask.Status.Upgrading = 0
		if err := r.Status().Patch(ctx, &upgradeTask, patch); err != nil {
			log.Error(err, "Failed to initialize FirmwareUpgrade status to Pending")
			return controllerruntime.Result{}, err
		}

		// Requeue immediately to re-process with the new status.
		return controllerruntime.Result{Requeue: true}, nil
	}

	// If the task is already completed, do nothing.
	if upgradeTask.Status.Phase == firmwarev1alpha1.UpgradePhaseSucceeded || upgradeTask.Status.Phase == firmwarev1alpha1.UpgradePhaseFailed {
		log.Info("Upgrade task is already in a terminal state.", "phase", upgradeTask.Status.Phase)
		return controllerruntime.Result{}, nil
	}

	log.Info("Performing status aggregation for FirmwareUpgrade task.", "phase", upgradeTask.Status.Phase)

	var deviceList iotv1alpha1.DeviceList
	selector, _ := metav1.LabelSelectorAsSelector(upgradeTask.Spec.DeviceSelector)
	if err := r.List(ctx, &deviceList, &client.ListOptions{LabelSelector: selector}); err != nil {
		log.Error(err, "Failed to list target devices for status aggregation")
		return controllerruntime.Result{}, err
	}

	// 7. Core upgrade logic: Iterate through devices and "command" them to upgrade
	// In a real system, this would involve calling a service, sending a message (MQTT), etc.
	// Here, we simulate this by updating the Device's Spec to our desired version.
	// The DeviceReconciler will then see this change and act upon it.
	var successful, failed, inProgress int32

	for _, device := range deviceList.Items {
		// Check if device is already on the target version
		if device.Status.ReportedFirmwareVersion == upgradeTask.Spec.Version {
			successful++
		} else {
			inProgress++
		}
	}

	patch := client.MergeFrom(upgradeTask.DeepCopy())

	// 8. Update the FirmwareUpgrade status with the current progress
	upgradeTask.Status.Succeeded = successful
	upgradeTask.Status.Failed = failed
	upgradeTask.Status.Upgrading = inProgress

	// This happens when no devices are left in the "upgrading" state.
	isTaskComplete := (successful + failed) == upgradeTask.Status.Total
	if isTaskComplete {
		// If the task is complete, THEN determine the final phase.
		if failed > 0 {
			upgradeTask.Status.Phase = firmwarev1alpha1.UpgradePhaseFailed
			log.Info("Firmware upgrade task completed with one or more failures.")
		} else {
			upgradeTask.Status.Phase = firmwarev1alpha1.UpgradePhaseSucceeded
			log.Info("All devices have been successfully upgraded.")
		}
	}

	// If isTaskComplete is false, we don't change the Phase. It remains "Upgrading"
	// while the counts are being updated in each cycle.
	if err := r.Status().Patch(ctx, &upgradeTask, patch); err != nil {
		log.Error(err, "Failed to patch FirmwareUpgrade aggregate status")
		return controllerruntime.Result{}, err
	}

	return controllerruntime.Result{RequeueAfter: 2 * time.Minute}, nil
}

func (r *FirmwareUpgradeReconciler) SetupWithManager(ctx context.Context, mgr controllerruntime.Manager) error {
	return controllerruntime.NewControllerManagedBy(mgr).
		For(&firmwarev1alpha1.FirmwareUpgrade{}).
		// Watch for changes to Devices as well. If a Device's status.reportedFirmwareVersion
		// changes, we need to re-evaluate our FirmwareUpgrade tasks.
		Watches(
			&iotv1alpha1.Device{},
			// This part is more advanced and requires a custom handler to map a Device change
			// back to the relevant FirmwareUpgrade task(s). For simplicity, we will rely on
			// the periodic requeue for now. A full implementation would use handler.EnqueueRequestsFromMapFunc.
			handler.EnqueueRequestsFromMapFunc(r.findFirmwareUpgradesForDevice),
		).
		Complete(r)
}

// findFirmwareUpgradesForDevice is our new mapping function.
// It is called when a Device object is changed.
// It returns a list of reconcile.Request for FirmwareUpgrade objects that should be reconciled.
func (r *FirmwareUpgradeReconciler) findFirmwareUpgradesForDevice(ctx context.Context, deviceObject client.Object) []reconcile.Request {
	log := log.FromContext(ctx)

	changedDevice, ok := deviceObject.(*iotv1alpha1.Device)
	if !ok {
		// This should not happen, but it's a good practice to handle it.
		log.Error(fmt.Errorf("unexpected type for device watch"), "expected Device", "got", fmt.Sprintf("%T", deviceObject))
		return []reconcile.Request{}
	}

	// List all FirmwareUpgrade objects that are currently active.
	var upgradeList firmwarev1alpha1.FirmwareUpgradeList
	if err := r.List(ctx, &upgradeList, client.InNamespace(changedDevice.Namespace)); err != nil {
		log.Error(err, "Failed to list FirmwareUpgrades to find matching tasks")
		return []reconcile.Request{}
	}

	requests := []reconcile.Request{}
	for _, upgrade := range upgradeList.Items {
		// We only care about upgrades that are not in a final state.
		if upgrade.Status.Phase == firmwarev1alpha1.UpgradePhaseSucceeded || upgrade.Status.Phase == firmwarev1alpha1.UpgradePhaseFailed {
			continue
		}

		// Convert the label selector from the upgrade's spec into a selector object.
		selector, err := metav1.LabelSelectorAsSelector(upgrade.Spec.DeviceSelector)
		if err != nil {
			log.Error(err, "Failed to parse selector for FirmwareUpgrade", "name", upgrade.Name)
			continue // Skip this one, it has an invalid selector.
		}

		// Check if the changed device's labels match the upgrade's selector.
		if selector.Matches(labels.Set(changedDevice.GetLabels())) {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      upgrade.Name,
					Namespace: upgrade.Namespace,
				},
			})
		}
	}

	if len(requests) > 0 {
		log.Info("Device change triggered reconciliation for FirmwareUpgrades", "device", changedDevice.Name, "upgradeCount", len(requests))
	}
	return requests
}
