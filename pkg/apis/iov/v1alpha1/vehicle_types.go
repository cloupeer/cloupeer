package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VehicleSpec defines the desired state of Vehicle
type VehicleSpec struct {
	// A human-readable description of the vehicle.
	// +optional
	Description string `json:"description,omitempty"`

	// The desired firmware version for this vehicle.
	// +optional
	FirmwareVersion string `json:"firmwareVersion,omitempty"`
}

// VehicleStatus defines the observed state of Vehicle
type VehicleStatus struct {
	// The last reported phase of the vehicle.
	// +optional
	Phase string `json:"phase,omitempty"`

	// The last time the vehicle was seen by the control plane.
	// +optional
	LastSeenTime *metav1.Time `json:"lastSeenTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Vehicle is the Schema for the vehicles API
type Vehicle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VehicleSpec   `json:"spec,omitempty"`
	Status VehicleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VehicleList contains a list of Vehicle
type VehicleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Vehicle `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Vehicle{}, &VehicleList{})
}
