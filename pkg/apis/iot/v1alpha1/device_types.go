package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DevicePhase defines the observed high-level state of the device.
type DevicePhase string

// Constants representing the possible phases of a device.
const (
	DevicePhaseOnline    DevicePhase = "Online"    // Device is connected and reporting.
	DevicePhaseOffline   DevicePhase = "Offline"   // Device is disconnected.
	DevicePhaseUnknown   DevicePhase = "Unknown"   // Device state cannot be determined.
	DevicePhaseUnhealthy DevicePhase = "Unhealthy" // Device is connected but reporting issues.
)

// DeviceSpec defines the desired state or configuration of a specific Device instance.
type DeviceSpec struct {
	// Required: DeviceModelRef is a reference to the DeviceModel used as a template
	// for this device instance. The referenced DeviceModel must exist in the same namespace.
	DeviceModelRef *v1.LocalObjectReference `json:"deviceModelRef,omitempty"`

	// Required: Protocol configuration specifies how to connect to this specific device instance.
	// It includes the protocol name (which should generally match the one in DeviceModelRef)
	// and instance-specific connection parameters (e.g., IP address, port, credentials).
	Protocol ProtocolConfig `json:"protocol,omitempty"`

	// List of properties specific to this device instance.
	// Each property here must correspond to a ModelProperty defined in the referenced DeviceModel.
	// It defines the desired state (for controllable properties) and reporting configuration for each property.
	// +optional
	Properties []DeviceProperty `json:"properties,omitempty"`

	// Optional device attributes like serial number, MAC address, location, etc.
	// Stored as key-value pairs. Consider using metadata.annotations as an alternative.
	// +optional
	// Attributes map[string]string `json:"attributes,omitempty"`
}

// DeviceStatus reports the observed state of the Device instance.
// It includes the device's phase, reported property values (twins), connection status, and detailed conditions.
type DeviceStatus struct {
	// Phase represents the high-level summary of the device's status.
	// It should be derived from the more detailed Conditions.
	// +optional
	Phase DevicePhase `json:"phase,omitempty"`

	// Twins represents the digital twin state for device properties.
	// Each twin tracks the reported value (actual state) of a property.
	// +optional
	Twins []Twin `json:"twins,omitempty"`

	// LastOnlineTime is the timestamp when the device last successfully communicated with the hub.
	// +optional
	LastOnlineTime *metav1.Time `json:"lastOnlineTime,omitempty"`

	// Conditions provide detailed observations of the resource's current state.
	// They follow the standard Kubernetes Conditions pattern.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Twin represents the digital twin state for a single device property.
// It primarily holds the last reported value from the device.
type Twin struct {
	// PropertyName is the name of the property this twin corresponds to.
	// It must match the Name field of a ModelProperty in the DeviceModel and a DeviceProperty in the Device Spec.
	PropertyName string `json:"propertyName,omitempty"`

	// Reported represents the last known actual value of the property reported by the device agent.
	Reported TwinProperty `json:"reported,omitempty"`

	// ObservedDesired represents the last desired value that the device agent acknowledged receiving and is attempting to apply.
	// Useful for tracking command acknowledgement in the control loop.
	// +optional
	ObservedDesired TwinProperty `json:"observedDesired,omitempty"`
}

// TwinProperty represents a specific value (either desired or reported) for a device property twin,
// potentially including associated metadata like timestamps.
type TwinProperty struct {
	// Value is the actual value of the property, represented as a string.
	// The interpretation of this string depends on the 'Type' defined in the corresponding ModelProperty.
	Value string `json:"value,omitempty"`

	// Metadata contains additional information about the value, such as the timestamp when it was reported.
	// Keys could include "timestamp", "source", etc.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`
}

// DeviceProperty defines instance-specific configuration for a property declared in the DeviceModel.
// It includes the desired state for controllable properties and reporting settings.
type DeviceProperty struct {
	// Name of the property. Must match a Name in the referenced DeviceModel's properties.
	Name string `json:"name,omitempty"`

	// Desired represents the desired value for a controllable property (ReadWrite access mode).
	// The cloud platform sets this value, and the device agent attempts to apply it.
	// For ReadOnly properties, this field is ignored.
	Desired TwinProperty `json:"desired,omitempty"`

	// ReportIntervalSeconds specifies the desired interval (in seconds) for reporting this property's value to the cloud.
	// Overrides the default reporting interval if specified.
	// +optional
	ReportIntervalSeconds *int64 `json:"reportIntervalSeconds,omitempty"`

	// ReportToCloud indicates whether the value of this property should be reported to the cloud platform.
	// Defaults to true if not specified. Allows filtering out unnecessary data.
	// Use pointer for explicit false vs unset
	// +optional
	ReportToCloud *bool `json:"reportToCloud,omitempty"`
}

// ProtocolConfig defines the specific protocol and connection parameters for a device instance.
type ProtocolConfig struct {
	// ProtocolName specifies the name of the protocol to use (e.g., "modbus", "opcua").
	// Should generally match the Protocol field in the referenced DeviceModel.
	ProtocolName string `json:"protocolName,omitempty"`

	// ConfigData holds protocol-specific configuration parameters (e.g., IP address, port, device address, security settings).
	// The structure is flexible (map[string]any) and interpreted by the corresponding protocol adapter (Agent logic).
	// +optional
	ConfigData *CustomizedValue `json:"configData,omitempty"`
}

// Device is the Schema for the devices API
// Represents a single physical device instance managed by the Cloupeer platform.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="The current status of the device"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Device struct {
	metav1.TypeMeta `json:",inline"`
	// Standard Kubernetes object metadata. metadata.name is the unique identifier for this Device CR within the namespace.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state and configuration of the Device.
	Spec DeviceSpec `json:"spec,omitempty"`
	// Status reflects the observed state of the Device as reported by the system.
	Status DeviceStatus `json:"status,omitempty"`
}

// DeviceList contains a list of Device.
// +kubebuilder:object:root=true
type DeviceList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard Kubernetes list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of Device resources.
	Items []Device `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Device{}, &DeviceList{})
}
