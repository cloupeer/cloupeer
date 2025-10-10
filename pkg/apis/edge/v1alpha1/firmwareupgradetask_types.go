package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TaskPhase defines the state of a task.
type TaskPhase string

const (
	// TaskPhasePending means the task has been created but not yet picked up by an agent.
	TaskPhasePending TaskPhase = "Pending"
	// TaskPhaseDownloading means the agent has picked up the task and is downloading the firmware.
	TaskPhaseDownloading TaskPhase = "Downloading"
	// TaskPhaseInstalling means the agent is installing the firmware.
	TaskPhaseInstalling TaskPhase = "Installing"
	// TaskPhaseSucceeded means the task completed successfully.
	TaskPhaseSucceeded TaskPhase = "Succeeded"
	// TaskPhaseFailed means the task failed.
	TaskPhaseFailed TaskPhase = "Failed"
)

// FirmwareUpgradeTaskSpec defines the desired state of FirmwareUpgradeTask
type FirmwareUpgradeTaskSpec struct {
	// DeviceID is the target device for this upgrade task.
	// It's used by the gateway to look up tasks for a specific device.
	// +kubebuilder:validation:Required
	DeviceID string `json:"deviceId"`

	// Firmware contains the details of the firmware to be installed.
	// +kubebuilder:validation:Required
	Firmware FirmwareSpec `json:"firmware"`

	// RetryPolicy could be added here in the future.
}

// FirmwareUpgradeTaskStatus defines the observed state of FirmwareUpgradeTask
type FirmwareUpgradeTaskStatus struct {
	// Phase is the current phase of the task.
	Phase TaskPhase `json:"phase,omitempty"`

	// Message provides human-readable details about the current state, especially for failures.
	Message string `json:"message,omitempty"`

	// StartTime is the time the task was picked up by the agent.
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time the task was completed.
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="DEVICE_ID",type="string",JSONPath=".spec.deviceId"
//+kubebuilder:printcolumn:name="VERSION",type="string",JSONPath=".spec.firmware.version"
//+kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// FirmwareUpgradeTask is the Schema for the firmwareupgradetasks API
type FirmwareUpgradeTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FirmwareUpgradeTaskSpec   `json:"spec,omitempty"`
	Status FirmwareUpgradeTaskStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FirmwareUpgradeTaskList contains a list of FirmwareUpgradeTask
type FirmwareUpgradeTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FirmwareUpgradeTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FirmwareUpgradeTask{}, &FirmwareUpgradeTaskList{})
}
