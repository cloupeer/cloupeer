package vehicle

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	iovv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha1"
	"cloupeer.io/cloupeer/pkg/log"
)

const (
	// simulateDownloadDuration is the fake time it takes to download the firmware.
	simulateDownloadDuration = 30 * time.Second
	// simulateInstallFailVersion is a specific version we use to simulate a failed installation.
	simulateInstallFailVersion = "v2.1.0"
)

// SimulateOTA is the main entrypoint for the OTA simulation logic.
// It is called by the SubStateMachine when the Vehicle is in the 'Pending' phase.
//
// This function simulates a three-step process in a specific order:
// 1. Download (which is asynchronous and uses RequeueAfter)
// 2. Install (instantaneous)
// 3. Reboot (instantaneous)
//
// It checks the status of each step and only proceeds to the next if the
// current one is complete.
func SimulateOTA(v *iovv1alpha1.Vehicle) (ctrl.Result, error) {

	// --- 1. Downloading Step ---
	// Check if the download is complete.
	downloadCond := meta.FindStatusCondition(v.Status.Conditions, iovv1alpha1.ConditionTypeDownloaded)
	if downloadCond == nil || downloadCond.Status == metav1.ConditionFalse {
		// Download is not complete. Run the download simulation.
		res, err := simulateDownload(v)
		if err != nil {
			// A real failure occurred (e.g., simulated network error)
			// The "Failed" condition was set inside simulateDownload.
			return ctrl.Result{}, nil
		}
		if res.RequeueAfter > 0 {
			// Download is still in progress, requeue and wait.
			return res, nil
		}
	}

	// --- 2. Installing Step ---
	// Check if the install is complete.
	installCond := meta.FindStatusCondition(v.Status.Conditions, iovv1alpha1.ConditionTypeInstalled)
	if installCond == nil || installCond.Status == metav1.ConditionFalse {
		// Install is not complete. Run the install simulation.
		if err := simulateInstall(v); err != nil {
			// A simulated install failure occurred.
			// The "Failed" condition was set inside simulateInstall.
			return ctrl.Result{}, nil
		}
	}

	// --- 3. Rebooting Step ---
	// Check if the reboot is complete.
	rebootCond := meta.FindStatusCondition(v.Status.Conditions, iovv1alpha1.ConditionTypeRebooted)
	if rebootCond == nil || rebootCond.Status == metav1.ConditionFalse {
		// Reboot is not complete. Run the reboot simulation.
		simulateReboot(v)
	}

	// --- 4. All steps done ---
	// All simulation steps are complete.
	// Return an empty result, which will cause a Status.Patch().
	// The SubStateMachine's next reconcile loop will see the
	// "Rebooted" condition (via FindLatestCondition) and transition to the "Succeeded" phase.
	return ctrl.Result{}, nil
}

// simulateDownload handles the logic for the (fake) download step.
// It uses the `ConditionTypeDownloaded`'s LastTransitionTime as its timer.
func simulateDownload(v *iovv1alpha1.Vehicle) (ctrl.Result, error) {

	// --- Simulated Failure Check ---
	// This provides a deterministic way to test failures.
	if strings.HasSuffix(v.Name, "7") {
		log.Warn("Simulating download network failure.")
		SetCondition(v, iovv1alpha1.ConditionTypeFailed, metav1.ConditionTrue, "DownloadFailed", "simulated network failure")
		return ctrl.Result{}, errors.New("simulated network failure")
	}

	// Find the "Downloaded" condition to manage our timer state.
	downloadCond := meta.FindStatusCondition(v.Status.Conditions, iovv1alpha1.ConditionTypeDownloaded)

	if downloadCond == nil {
		// This is the first time. We need to start the timer.
		log.Info("Initializing simulated download timer...")
		// Set the "Downloading" condition. This will set its LastTransitionTime to metav1.Now().
		// This status update will cause a Patch and an immediate requeue.
		SetCondition(v, iovv1alpha1.ConditionTypeDownloaded, metav1.ConditionFalse, "Downloading", "Firmware download in progress")
		// Return the full duration. The next reconcile loop (from the Patch)
		// will enter the 'else' block below.
		return ctrl.Result{RequeueAfter: simulateDownloadDuration}, nil
	}

	// We are in the "Downloading" wait period (Status is "False").
	elapsed := time.Since(downloadCond.LastTransitionTime.Time)

	if elapsed < simulateDownloadDuration {
		// Still downloading.
		remaining := simulateDownloadDuration - elapsed
		log.Info("Download in progress...", "remaining", remaining, "elapsed", elapsed)

		// Return RequeueAfter. CRUCIALLY: do not modify the status object here.
		// This prevents the Patch vs. RequeueAfter conflict.
		return ctrl.Result{RequeueAfter: remaining}, nil
	}

	// Timer is up! Download is complete.
	log.Info("Simulated download complete.")
	SetCondition(v, iovv1alpha1.ConditionTypeDownloaded, metav1.ConditionTrue, "DownloadComplete", "Firmware downloaded")

	return ctrl.Result{}, nil
}

// simulateInstall handles the (fake) instantaneous install step.
func simulateInstall(v *iovv1alpha1.Vehicle) error {
	// Set the "Installing" condition
	SetCondition(v, iovv1alpha1.ConditionTypeInstalled, metav1.ConditionFalse, "Installing", "Firmware install in progress")

	// --- Simulated Failure Check ---
	if v.Spec.FirmwareVersion == simulateInstallFailVersion {
		errMsg := fmt.Sprintf("Firmware is bad in version: %s", simulateInstallFailVersion)
		log.Warn("Simulating bad firmware install failure.", "version", simulateInstallFailVersion)
		SetCondition(v, iovv1alpha1.ConditionTypeFailed, metav1.ConditionTrue, "InstallFailed", errMsg)
		return errors.New(errMsg)
	}

	// --- Success ---
	log.Info("Simulated install complete.")
	SetCondition(v, iovv1alpha1.ConditionTypeInstalled, metav1.ConditionTrue, "InstallComplete", "Firmware installed")

	return nil
}

// simulateReboot handles the (fake) instantaneous reboot step.
func simulateReboot(v *iovv1alpha1.Vehicle) {
	log.Info("Starting simulated vehicle reboot...")
	SetCondition(v, iovv1alpha1.ConditionTypeRebooted, metav1.ConditionFalse, "Rebooting", "Vehicle reboot in progress")

	// This is the most important part of the simulation:
	// The vehicle "reports" its new version.
	v.Status.ReportedFirmwareVersion = v.Spec.FirmwareVersion

	log.Info("Simulated reboot complete.", "newVersion", v.Status.ReportedFirmwareVersion)
	SetCondition(v, iovv1alpha1.ConditionTypeRebooted, metav1.ConditionTrue, "RebootComplete", "Vehicle rebooted")
}
