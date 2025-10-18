package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpgradePhase defines the observed state of the firmware upgrade.
type UpgradePhase string

// Constants for upgrade phases.
const (
	UpgradePhasePending   UpgradePhase = "Pending"
	UpgradePhaseUpgrading UpgradePhase = "Upgrading"
	UpgradePhaseSucceeded UpgradePhase = "Succeeded"
	UpgradePhaseFailed    UpgradePhase = "Failed"
)

// FirmwareUpgradeSpec defines the desired state of FirmwareUpgrade
type FirmwareUpgradeSpec struct {
	// The version of the firmware to upgrade to.
	Version string `json:"version"`

	// The URL where the firmware image can be downloaded.
	ImageUrl string `json:"imageUrl"`

	// Selector to target which devices this upgrade should apply to.
	// This uses standard Kubernetes label selection.
	DeviceSelector *metav1.LabelSelector `json:"deviceSelector"`
}

// FirmwareUpgradeStatus defines the observed state of FirmwareUpgrade
type FirmwareUpgradeStatus struct {
	// The current phase of the upgrade.
	// +optional
	Phase UpgradePhase `json:"phase,omitempty"`

	// Total number of devices targeted by this upgrade.
	// +optional
	Total int32 `json:"total,omitempty"`

	// Number of devices that are currently upgrading.
	// +optional
	Upgrading int32 `json:"upgrading,omitempty"`

	// Number of devices that have successfully upgraded.
	// +optional
	Succeeded int32 `json:"succeeded,omitempty"`

	// Number of devices that failed to upgrade.
	// +optional
	Failed int32 `json:"failed,omitempty"`

	// A list of conditions for the upgrade.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="The current phase of the upgrade"
//+kubebuilder:printcolumn:name="Total",type="integer",JSONPath=".status.total",description="Total number of devices targeted"
//+kubebuilder:printcolumn:name="Succeeded",type="integer",JSONPath=".status.succeeded",description="Number of devices successfully upgraded"
//+kubebuilder:printcolumn:name="Failed",type="integer",JSONPath=".status.failed",description="Number of devices that failed to upgrade"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// FirmwareUpgrade is the Schema for the firmwareupgrades API
type FirmwareUpgrade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FirmwareUpgradeSpec   `json:"spec,omitempty"`
	Status FirmwareUpgradeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FirmwareUpgradeList contains a list of FirmwareUpgrade
type FirmwareUpgradeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FirmwareUpgrade `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FirmwareUpgrade{}, &FirmwareUpgradeList{})
}
