package device

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	firmwarev1alpha1 "cloupeer.io/cloupeer/pkg/apis/firmware/v1alpha1"
	iotv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iot/v1alpha1"
)

type DeviceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewDeviceReconciler(cli client.Client, sche *runtime.Scheme) *DeviceReconciler {
	return &DeviceReconciler{Client: cli, Scheme: sche}
}

//+kubebuilder:rbac:groups=iot.cloupeer.io,resources=devices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=iot.cloupeer.io,resources=devices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=iot.cloupeer.io,resources=devices/finalizers,verbs=update
//+kubebuilder:rbac:groups=firmware.cloupeer.io,resources=firmwareupgrades,verbs=get;list;watch;create

// Reconcile is the core logic that gets called for every change to a Device object.
func (r *DeviceReconciler) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Starting reconciliation for device")

	var device iotv1alpha1.Device
	if err := r.Get(ctx, req.NamespacedName, &device); err != nil {
		if errors.IsNotFound(err) {
			// 如果错误是 IsNotFound，意味着对象已经被删除了。
			// 这种情况通常发生在用户执行了 `kubectl delete` 之后。
			// 对于删除事件，我们通常不需要做任何事（因为 OwnerReference 会自动清理子资源），
			// 所以记录一条日志然后正常返回即可。
			log.Info("Device resource not found. Ignoring since object must be deleted.")
			return controllerruntime.Result{}, nil
		}

		// 如果是其他类型的错误（比如网络问题、权限问题），
		// 我们应该记录错误并返回 err，这样 controller-runtime 会稍后自动重试。
		log.Error(err, "unable to fetch device")
		return controllerruntime.Result{}, err
	}

	var desiredFirmwareVersion string
	var firmwareProperty *iotv1alpha1.DeviceProperty // Find the firmware property in Spec
	for i := range device.Spec.Properties {
		if device.Spec.Properties[i].Name == "firmwareVersion" { // Assuming "firmwareVersion" is the standard name
			firmwareProperty = &device.Spec.Properties[i]
			desiredFirmwareVersion = firmwareProperty.Desired.Value
			break
		}
	}

	var reportedFirmwareVersion string
	for _, twin := range device.Status.Twins {
		if twin.PropertyName == "firmwareVersion" {
			reportedFirmwareVersion = twin.Reported.Value
			break
		}
	}

	requeueAfter := 2 * time.Minute
	// If no desired version is set, or if versions match, no action is needed.
	if desiredFirmwareVersion == "" || desiredFirmwareVersion == reportedFirmwareVersion {
		// You might want to add logic here to clean up any "Upgrading" conditions if they exist.
		// For now, we'll just log and finish.
		log.Info("Firmware version is up to date or not specified.", "desired", desiredFirmwareVersion, "reported", reportedFirmwareVersion)
		if meta.IsStatusConditionTrue(device.Status.Conditions, "UpgradingFirmware") {
			log.Info("Removing UpgradingFirmware condition as version matches or desired is empty.")
			meta.RemoveStatusCondition(&device.Status.Conditions, "UpgradingFirmware")
			if err := r.Status().Update(ctx, &device); err != nil {
				log.Error(err, "Failed to remove UpgradingFirmware condition")
				// Don't requeue immediately on status update failure, let reconcile retry
			}
		}
		return controllerruntime.Result{}, nil
	}

	log.Info("Firmware mismatch detected, upgrade required.", "desired", desiredFirmwareVersion, "reported", reportedFirmwareVersion)

	firmwareUpgrade := &firmwarev1alpha1.FirmwareUpgrade{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-to-%s", device.Name, desiredFirmwareVersion),
			Namespace: device.Namespace,
			Labels: map[string]string{
				"iot.cloupeer.io/device-name": device.Name,
			},
		},
		Spec: firmwarev1alpha1.FirmwareUpgradeSpec{
			Version: desiredFirmwareVersion,
			// ARCHITECTURAL NOTE: The ImageUrl is a required field in the FirmwareUpgrade CRD.
			// A robust system would have a central registry (perhaps another CRD or a database)
			// where the FirmwareUpgrade controller can look up this URL based on the version.
			// For this example, we will construct a placeholder URL.
			ImageUrl: fmt.Sprintf("https://firmware.cloupeer.io/images/%s/firmware.bin", desiredFirmwareVersion),
			DeviceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					// This selector ensures the upgrade targets ONLY this specific device.
					"iot.cloupeer.io/device-name": device.Name,
				},
			},
		},
	}

	// Ensure the Device object has the label we need for the selector.
	// This makes the system self-correcting.
	if device.Labels == nil {
		device.Labels = make(map[string]string)
	}
	if device.Labels["iot.cloupeer.io/device-name"] != device.Name {
		device.Labels["iot.cloupeer.io/device-name"] = device.Name
		if err := r.Update(ctx, &device); err != nil {
			log.Error(err, "Failed to add required label to device")
			return controllerruntime.Result{}, err
		}
		log.Info("Added missing device label for selection. Requeuing.")
		// Requeue because the object was modified.
		return controllerruntime.Result{RequeueAfter: requeueAfter}, nil
	}

	// Set the Device as the owner of the FirmwareUpgrade resource.
	// This is crucial for garbage collection: if the Device is deleted, Kubernetes
	// will automatically delete this FirmwareUpgrade object.
	if err := controllerutil.SetControllerReference(&device, firmwareUpgrade, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference on FirmwareUpgrade")
		return controllerruntime.Result{}, err
	}

	// Check if the FirmwareUpgrade already exists. If not, create it.
	var existingFU firmwarev1alpha1.FirmwareUpgrade
	err := r.Get(ctx, types.NamespacedName{Name: firmwareUpgrade.Name, Namespace: firmwareUpgrade.Namespace}, &existingFU)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new FirmwareUpgrade resource", "FirmwareUpgrade.Name", firmwareUpgrade.Name)
		if createErr := r.Create(ctx, firmwareUpgrade); createErr != nil {
			log.Error(createErr, "Failed to create new FirmwareUpgrade resource")
			return controllerruntime.Result{}, createErr
		}

		log.Info("Setting device condition to 'UpgradingFirmware'")
		upgradingCondition := metav1.Condition{
			Type:               "UpgradingFirmware",
			Status:             metav1.ConditionTrue,
			ObservedGeneration: device.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             "UpgradeTriggered",
			Message:            fmt.Sprintf("Firmware upgrade to version %s initiated by FirmwareUpgrade %s", desiredFirmwareVersion, firmwareUpgrade.Name),
		}
		meta.SetStatusCondition(&device.Status.Conditions, upgradingCondition)
		// Optionally update Phase as well, derived from conditions
		// device.Status.Phase = iotv1alpha1.DevicePhaseUnhealthy // Or a specific "Upgrading" phase if defined

		if err := r.Status().Update(ctx, &device); err != nil {
			log.Error(err, "Failed to update device status with UpgradingFirmware condition")
			// Don't return error immediately, allow reconcile to retry creation check
		}

		return controllerruntime.Result{RequeueAfter: requeueAfter}, nil
	} else if err != nil {
		log.Error(err, "Failed to get FirmwareUpgrade resource")
		return controllerruntime.Result{}, err
	}

	log.Info("FirmwareUpgrade resource already exists. No action needed.", "FirmwareUpgrade.Name", existingFU.Name)
	// Optionally, you can check existingFU.Status here and update the Device status accordingly.

	return controllerruntime.Result{RequeueAfter: requeueAfter}, nil
}

func (r *DeviceReconciler) SetupWithManager(ctx context.Context, mgr controllerruntime.Manager) error {
	return controllerruntime.NewControllerManagedBy(mgr).
		For(&iotv1alpha1.Device{}).
		// Also watch FirmwareUpgrade objects that are owned by a Device.
		// This allows us to react if an upgrade succeeds or fails.
		Owns(&firmwarev1alpha1.FirmwareUpgrade{}).
		Complete(r)
}
