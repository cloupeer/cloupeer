package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CommandType defines the type of command being sent to the vehicle.
type CommandType string

const (
	// CommandTypeOTA represents an Over-The-Air firmware update command.
	CommandTypeOTA CommandType = "OTA"
	// CommandTypeReboot represents a request to reboot the vehicle's edge system.
	CommandTypeReboot CommandType = "Reboot"
)

// CommandPhase defines the lifecycle phase of the command.
type CommandPhase string

const (
	// CommandPhasePending means the command has been created but not yet processed by the controller.
	CommandPhasePending CommandPhase = "Pending"

	// CommandPhaseSent means the controller has forwarded the command to the Hub (and presumably MQTT).
	CommandPhaseSent CommandPhase = "Sent"

	// CommandPhaseReceived means the vehicle agent has acknowledged receipt of the command.
	// This corresponds to the "trigger message to remind owner" stage.
	CommandPhaseReceived CommandPhase = "Received"

	// CommandPhaseRunning means the actual operation (e.g., downloading/installing) is in progress.
	// This happens after the owner clicks "Upgrade".
	CommandPhaseRunning CommandPhase = "Running"

	// CommandPhaseSucceeded means the command executed successfully.
	CommandPhaseSucceeded CommandPhase = "Succeeded"

	// CommandPhaseFailed means the command failed or timed out.
	CommandPhaseFailed CommandPhase = "Failed"
)

// VehicleCommandSpec defines the desired state of VehicleCommand
type VehicleCommandSpec struct {
	// VehicleName is the name of the target Vehicle resource in the same namespace.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	VehicleName string `json:"vehicleName"`

	// Command is the type of operation to perform.
	// +kubebuilder:validation:Enum=OTA;Reboot
	Command CommandType `json:"command"`

	// Parameters contains optional arguments for the command.
	// For OTA, this might include "TargetVersion".
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`
}

// VehicleCommandStatus defines the observed state of VehicleCommand
type VehicleCommandStatus struct {
	// Phase represents the current stage of the command processing.
	// +optional
	Phase CommandPhase `json:"phase,omitempty"`

	// Message provides human-readable details about the current status or error.
	// +optional
	Message string `json:"message,omitempty"`

	// LastUpdateTime is the timestamp of the last status update.
	// +optional
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// AcknowledgeTime is the timestamp when the device confirmed receipt.
	// +optional
	AcknowledgeTime *metav1.Time `json:"acknowledgeTime,omitempty"`

	// StartTime is the timestamp when the actual execution started (e.g. user clicked upgrade).
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the timestamp when the command finished (success or fail).
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Vehicle",type="string",JSONPath=".spec.vehicleName",description="Target Vehicle Name"
//+kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.command",description="Command Type"
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Command Phase"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// VehicleCommand is the Schema for the vehiclecommands API
type VehicleCommand struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VehicleCommandSpec   `json:"spec,omitempty"`
	Status VehicleCommandStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VehicleCommandList contains a list of VehicleCommand
type VehicleCommandList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VehicleCommand `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VehicleCommand{}, &VehicleCommandList{})
}
