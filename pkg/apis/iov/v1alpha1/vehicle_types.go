package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	// The last reported phase of the vehicle's OTA status.
	// +optional
	Phase VehiclePhase `json:"phase,omitempty"`

	// The firmware version last reported by the vehicle.
	// +optional
	ReportedFirmwareVersion string `json:"reportedFirmwareVersion,omitempty"`

	// Any error message during the OTA process.
	// +optional
	ErrorMessage string `json:"errorMessage,omitempty"`

	// The last time the vehicle was seen by the control plane.
	// +optional
	LastSeenTime *metav1.Time `json:"lastSeenTime,omitempty"`

	// Conditions represent the latest available observations of the Vehicle's state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// --- 定义 Condition Types ---
const (
	// ConditionTypeReady 表示车辆是否准备好接收新指令（即处于 Idle）
	ConditionTypeReady = "Ready"
	// ConditionTypeDownloaded 表示固件是否已下载
	ConditionTypeDownloaded = "Downloaded"
	// ConditionTypeInstalled 表示固件是否已安装
	ConditionTypeInstalled = "Installed"
	// ConditionTypeRebooted 表示车辆是否已重启
	ConditionTypeRebooted = "Rebooted"
	// ConditionTypeFailed 表示OTA升级失败
	ConditionTypeFailed = "Failed"
)

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
