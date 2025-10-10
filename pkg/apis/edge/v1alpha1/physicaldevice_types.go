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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FirmwareSpec defines the desired firmware state.
type FirmwareSpec struct {
	// Version specifies the target firmware version.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Version string `json:"version"`

	// URL points to the firmware package location.
	// It can be an HTTP/S endpoint or an object storage URL.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^https?://.*`
	URL string `json:"url"`

	// Checksum is the SHA-256 checksum of the firmware package
	// used by the device agent to verify integrity.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=64
	// +kubebuilder:validation:MaxLength=64
	Checksum string `json:"checksum"`
}

// PhysicalDeviceSpec defines the desired state of PhysicalDevice
type PhysicalDeviceSpec struct {
	// DeviceID is a unique identifier for the physical device, typically assigned by manufacturing
	// or provisioning systems. This is the minimal field for Hello World onboarding.
	// +kubebuilder:validation:Required
	DeviceID string `json:"deviceId"`

	// Firmware holds the desired firmware specification for the device.
	// The operator will create a FirmwareUpgradeTask if the reported version
	// in the status does not match this spec.
	// +optional
	Firmware *FirmwareSpec `json:"firmware,omitempty"`
}

// UpgradeStatus defines the status of the ongoing or last completed firmware upgrade.
type UpgradeStatus struct {
	// TaskRef is the name of the FirmwareUpgradeTask resource handling the current upgrade.
	TaskRef string `json:"taskRef,omitempty"`

	// State reflects the current phase of the upgrade process.
	// e.g., "InProgress", "Succeeded", "Failed".
	State string `json:"state,omitempty"`

	// LastFailureMessage provides details on the last failed upgrade attempt.
	LastFailureMessage string `json:"lastFailureMessage,omitempty"`

	// LastSuccessfulVersion is the last version that was successfully installed.
	LastSuccessfulVersion string `json:"lastSuccessfulVersion,omitempty"`

	// LastAttemptTime is the timestamp of the last upgrade attempt.
	LastAttemptTime *metav1.Time `json:"lastAttemptTime,omitempty"`
}

// PhysicalDeviceStatus defines the observed state of PhysicalDevice
type PhysicalDeviceStatus struct {
	// LastHeartbeatTime records the last time a heartbeat was received from the device via gateway
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime,omitempty"`
	// IPAddress is the last known IP address of the device as observed by the gateway
	IPAddress string `json:"ipAddress,omitempty"`

	// CurrentFirmwareVersion is the version reported by the device in its heartbeat.
	CurrentFirmwareVersion string `json:"currentFirmwareVersion,omitempty"`

	// FirmwareUpgradeStatus tracks the state of the firmware upgrade process.
	FirmwareUpgradeStatus *UpgradeStatus `json:"firmwareUpgradeStatus,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DEVICE_ID",type="string",JSONPath=".spec.deviceId"
// +kubebuilder:printcolumn:name="DESIRED_VERSION",type="string",JSONPath=".spec.firmware.version"
// +kubebuilder:printcolumn:name="REPORTED_VERSION",type="string",JSONPath=".status.currentFirmwareVersion"
// +kubebuilder:printcolumn:name="UPGRADE_STATE",type="string",JSONPath=".status.firmwareUpgradeStatus.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// PhysicalDevice is the Schema for the physicaldevices API
type PhysicalDevice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PhysicalDeviceSpec   `json:"spec,omitempty"`
	Status PhysicalDeviceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PhysicalDeviceList contains a list of PhysicalDevice
type PhysicalDeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PhysicalDevice `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PhysicalDevice{}, &PhysicalDeviceList{})
}
