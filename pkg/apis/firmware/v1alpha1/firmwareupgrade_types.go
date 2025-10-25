package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpgradePhase defines the observed high-level state of the firmware upgrade task.
type UpgradePhase string

// Constants representing the possible phases of a firmware upgrade task.
const (
	// UpgradePhasePending indicates the task has been created but not yet processed by the controller.
	UpgradePhasePending UpgradePhase = "Pending"
	// UpgradePhaseUpgrading indicates the controller is actively managing the upgrade process for the targeted devices.
	UpgradePhaseUpgrading UpgradePhase = "Upgrading"
	// UpgradePhaseSucceeded indicates all targeted devices have successfully completed the firmware upgrade.
	UpgradePhaseSucceeded UpgradePhase = "Succeeded"
	// UpgradePhaseFailed indicates the upgrade task failed for one or more devices, or the task itself encountered an error.
	UpgradePhaseFailed UpgradePhase = "Failed"
)

// FirmwareUpgradeSpec defines the desired state of FirmwareUpgrade.
// It specifies the parameters for initiating and targeting a firmware upgrade task.
type FirmwareUpgradeSpec struct {
	// Version specifies the target firmware version to upgrade the devices to.
	// This version string should be meaningful to the device agent.
	Version string `json:"version"`

	// ImageUrl specifies the URL from which the device agent should download the firmware image.
	// The accessibility of this URL from the device agent's perspective is crucial.
	ImageUrl string `json:"imageUrl"`

	// DeviceSelector is a standard Kubernetes label selector used to identify the target Device CRs
	// that this upgrade task should apply to. The FirmwareUpgrade controller will find devices
	// matching this selector.
	DeviceSelector *metav1.LabelSelector `json:"deviceSelector"`
}

// FirmwareUpgradeStatus defines the observed state of FirmwareUpgrade
// It reflects the progress and outcome of the upgrade task across all targeted devices.
type FirmwareUpgradeStatus struct {
	// Phase represents the current high-level phase of the overall upgrade task.
	// +optional
	Phase UpgradePhase `json:"phase,omitempty"`

	// Total indicates the total number of devices identified by the DeviceSelector
	// at the time the task started processing (or was last reconciled).
	// +optional
	Total int32 `json:"total,omitempty"`

	// Upgrading indicates the number of targeted devices currently in the process of upgrading.
	// This count is typically derived from the status of individual Device CRs.
	// +optional
	Upgrading int32 `json:"upgrading,omitempty"`

	// Succeeded indicates the number of targeted devices that have successfully upgraded
	// to the target version specified in the Spec.
	// +optional
	Succeeded int32 `json:"succeeded,omitempty"`

	// Failed indicates the number of targeted devices for which the upgrade attempt failed.
	// +optional
	Failed int32 `json:"failed,omitempty"`

	// Conditions provide detailed observations of the upgrade task's current state,
	// following the standard Kubernetes Conditions pattern. They can offer more granular
	// information about progress, errors, or specific stages.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// FirmwareUpgrade is the Schema for the firmwareupgrades API.
// It represents a task to upgrade the firmware of one or more devices matching a selector.
// The controller managing this resource is responsible for orchestrating the upgrade process.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="The current phase of the upgrade"
// +kubebuilder:printcolumn:name="Total",type="integer",JSONPath=".status.total",description="Total number of devices targeted"
// +kubebuilder:printcolumn:name="Succeeded",type="integer",JSONPath=".status.succeeded",description="Number of devices successfully upgraded"
// +kubebuilder:printcolumn:name="Failed",type="integer",JSONPath=".status.failed",description="Number of devices that failed to upgrade"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type FirmwareUpgrade struct {
	metav1.TypeMeta `json:",inline"`
	// Standard Kubernetes object metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired parameters of the firmware upgrade task.
	Spec FirmwareUpgradeSpec `json:"spec"`
	// Status reflects the observed progress and outcome of the firmware upgrade task.
	Status FirmwareUpgradeStatus `json:"status,omitempty"`
}

// FirmwareUpgradeList contains a list of FirmwareUpgrade.
// +kubebuilder:object:root=true
type FirmwareUpgradeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of FirmwareUpgrade resources.
	Items []FirmwareUpgrade `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FirmwareUpgrade{}, &FirmwareUpgradeList{})
}
