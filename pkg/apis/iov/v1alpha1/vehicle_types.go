package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VehicleFinalizer is the finalizer string used by the Vehicle controller.
const VehicleFinalizer = "iov.cloupeer.io/vehicle-finalizer"

// VehiclePhase defines the observed phase of the Vehicle OTA process.
type VehiclePhase string

// These are the valid phases of a Vehicle OTA process.
const (
	// VehiclePhaseIdle means the vehicle is stable and no update is in progress.
	VehiclePhaseIdle VehiclePhase = "Idle"

	// VehiclePhasePending means an update is required.
	VehiclePhasePending VehiclePhase = "Pending"

	// VehiclePhaseSucceeded means the update finished successfully.
	VehiclePhaseSucceeded VehiclePhase = "Succeeded"

	// VehiclePhaseFailed means the update failed at some point.
	VehiclePhaseFailed VehiclePhase = "Failed"
)

// VehicleSpec defines the desired state of Vehicle
type VehicleSpec struct {
	// A human-readable description of the vehicle.
	// +optional
	Description string `json:"description,omitempty"`

	// The desired firmware version for this vehicle.
	// The controller will attempt to update the vehicle to this version.
	// +optional
	FirmwareVersion string `json:"firmwareVersion,omitempty"`
}

// VehicleStatus defines the observed state of Vehicle
type VehicleStatus struct {
	// Online indicates whether the vehicle agent is currently connected to the hub.
	// +optional
	Online bool `json:"online"`

	// The last reported phase of the vehicle's OTA status.
	// +optional
	Phase VehiclePhase `json:"phase,omitempty"`

	// The firmware version last reported by the vehicle.
	// +optional
	ReportedFirmwareVersion string `json:"reportedFirmwareVersion,omitempty"`

	// Any message during the OTA process.
	// +optional
	Message string `json:"message,omitempty"`

	// RetryCount tracks the number of automated retries for the current update.
	// +optional
	RetryCount int32 `json:"retryCount"`

	// The last time the vehicle was seen by the control plane.
	// +optional
	LastSeenTime *metav1.Time `json:"lastSeenTime,omitempty"`

	// Conditions represent the latest available observations of the Vehicle's state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// Condition Types
const (
	// ConditionTypeReady indicates whether the vehicle is ready to accept new commands.
	//   True: The vehicle is idle, healthy, and ready for operations.
	//   False: The vehicle is currently processing a command (busy) or is in a failed state.
	ConditionTypeReady = "Ready"

	// ConditionTypeSynced indicates whether the observed state matches the desired state.
	//   True: The vehicle's reported firmware version matches the Spec (system is consistent).
	//   False: The vehicle is currently updating or pending an update (Spec and Status are diverged).
	ConditionTypeSynced = "Synced"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Online",type="boolean",JSONPath=".status.online",description="Vehicle Connection Status"
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Vehicle OTA Phase"
//+kubebuilder:printcolumn:name="Desired",type="string",JSONPath=".spec.firmwareVersion",description="Desired firmware version"
//+kubebuilder:printcolumn:name="Reported",type="string",JSONPath=".status.reportedFirmwareVersion",description="Reported firmware version"
//+kubebuilder:printcolumn:name="Retry",type="integer",JSONPath=".status.retryCount",description="Retry count"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message",description="Real-time status message",priority=1

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
