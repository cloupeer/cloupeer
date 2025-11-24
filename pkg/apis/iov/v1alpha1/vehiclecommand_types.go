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
	// CommandPhasePending means the command has been created but not yet processed.
	CommandPhasePending CommandPhase = "Pending"
	// CommandPhaseSent means the command has been delivered to the Hub/Broker.
	CommandPhaseSent CommandPhase = "Sent"
	// CommandPhaseReceived means the vehicle agent has acknowledged receipt.
	CommandPhaseReceived CommandPhase = "Received"
	// CommandPhaseRunning means the actual operation is in progress.
	CommandPhaseRunning CommandPhase = "Running"
	// CommandPhaseSucceeded means the command executed successfully.
	CommandPhaseSucceeded CommandPhase = "Succeeded"
	// CommandPhaseFailed means the command failed or timed out.
	CommandPhaseFailed CommandPhase = "Failed"
)

// Condition Types for OTA Commands
// These conditions represent the detailed milestones within the 'Running' phase.
const (
	// ConditionTypeDownloaded indicates whether the firmware artifact has been successfully downloaded to the vehicle.
	// This usually implies the file is present in the local storage.
	ConditionTypeDownloaded = "Downloaded"

	// ConditionTypeVerified indicates whether the downloaded artifact has passed integrity and security checks.
	// This includes checksum validation, signature verification (Uptane/TUF), and decryption.
	ConditionTypeVerified = "Verified"

	// ConditionTypeReadyToInstall indicates whether the vehicle meets the safety preconditions for installation.
	// Real-world checks include: Engine off, Gear in Park (P), Battery > 50%, Parking Brake engaged.
	ConditionTypeReadyToInstall = "ReadyToInstall"

	// ConditionTypeInstalled indicates whether the firmware has been written/flashed to the target ECU or partition.
	// For A/B updates, this means the inactive partition has been successfully flashed.
	ConditionTypeInstalled = "Installed"

	// ConditionTypeActivated indicates whether the new firmware has been marked as active/bootable.
	// This typically involves switching the boot slot (A/B switch) or updating the bootloader config.
	ConditionTypeActivated = "Activated"

	// ConditionTypeRebooted indicates whether the vehicle has successfully restarted to apply the update.
	// This is often the final step before the Agent reports 'Succeeded'.
	ConditionTypeRebooted = "Rebooted"
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
	// Phase represents the current lifecycle stage of the command.
	// +optional
	Phase CommandPhase `json:"phase,omitempty"`

	// Conditions allow extensibility for specific command types (e.g. OTA steps).
	// For an OTA command, this might contain "Downloaded", "Installed".
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

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
