package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DevicePhase defines the observed state of the device.
type DevicePhase string

// Constants for device phases.
const (
	DevicePhaseOnline    DevicePhase = "Online"
	DevicePhaseOffline   DevicePhase = "Offline"
	DevicePhaseUnknown   DevicePhase = "Unknown"
	DevicePhaseUnhealthy DevicePhase = "Unhealthy"
)

// DeviceSpec defines the desired state of Device
type DeviceSpec struct {
	// DeviceID is the unique identifier of the physical device in the real world,
	// such as a serial number or MAC address. This field is mandatory.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	DeviceID string `json:"deviceID"`

	// A human-readable description of the device.
	// +optional
	Description string `json:"description,omitempty"`

	// The desired firmware version for this device. The controller
	// will attempt to trigger an upgrade if this differs from the
	// version reported in the status.
	// +optional
	FirmwareVersion string `json:"firmwareVersion,omitempty"`

	// Key-value pairs for device-specific configuration.
	// These will be pushed to the device.
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

// DeviceStatus defines the observed state of Device
type DeviceStatus struct {
	// The last reported phase of the device.
	// +optional
	Phase DevicePhase `json:"phase,omitempty"`

	// The last time the device was seen by the control plane.
	// +optional
	LastSeenTime *metav1.Time `json:"lastSeenTime,omitempty"`

	// The current firmware version reported by the device.
	// +optional
	ReportedFirmwareVersion string `json:"reportedFirmwareVersion,omitempty"`

	// A list of conditions for the device.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="The current status of the device"
//+kubebuilder:printcolumn:name="Firmware",type="string",JSONPath=".status.reportedFirmwareVersion",description="The current firmware version of the device"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Device is the Schema for the devices API
type Device struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeviceSpec   `json:"spec,omitempty"`
	Status DeviceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DeviceList contains a list of Device
type DeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Device `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Device{}, &DeviceList{})
}
