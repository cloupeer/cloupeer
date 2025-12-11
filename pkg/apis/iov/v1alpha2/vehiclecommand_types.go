package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VehicleCommandSpec defines the desired command execution.
type VehicleCommandSpec struct {
	// VehicleName is the name of the target Vehicle resource in the same namespace.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	VehicleName string `json:"vehicleName"`

	// Method is the name of the operation to execute (e.g., "Reboot", "OTA", "OpenTrunk").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Method string `json:"method"`

	// RequestID allow external systems (like BFF) to trace the command.
	// Ideally maps to OpenTelemetry TraceID or a UUID.
	// +optional
	RequestID string `json:"requestID,omitempty"`

	// Priority defines the urgency of the command.
	// 0: Low (Background, e.g. Log Upload)
	// 1: Normal (Default, e.g. OTA)
	// 2: High (User Interactive, e.g. Remote Unlock)
	// Controller/CloudHub should prioritize processing High priority commands.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=2
	Priority *int32 `json:"priority,omitempty"`

	// Parameters contains the input arguments for the method.
	// We use map[string]string for consistency with Vehicle.Spec.Properties.
	// Complex JSON values should be serialized as strings.
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`

	// TimeoutSeconds defines the maximum time allowed for the command to complete
	// once it reaches the "Sent" phase. If exceeded, the controller marks it as Failed/Timeout.
	// +optional
	// +kubebuilder:validation:Minimum=1
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`
}

// CommandPhase defines the lifecycle stages of the command.
// +kubebuilder:validation:Enum=Pending;Sent;Acknowledged;Running;Succeeded;Failed;Timeout
type CommandPhase string

const (
	// CommandPhasePending means the command has been created but not yet processed.
	CommandPhasePending CommandPhase = "Pending"
	// CommandPhaseSent means the command has been delivered to the Hub/Broker.
	CommandPhaseSent CommandPhase = "Sent"
	// CommandPhaseAcknowledged means the vehicle agent has acknowledged receipt.
	CommandPhaseAcknowledged CommandPhase = "Acknowledged"
	// CommandPhaseRunning means the actual operation is in progress.
	CommandPhaseRunning CommandPhase = "Running"
	// CommandPhaseSucceeded means the command executed successfully.
	CommandPhaseSucceeded CommandPhase = "Succeeded"
	// CommandPhaseFailed means the command failed or timed out.
	CommandPhaseFailed CommandPhase = "Failed"
	// CommandPhaseTimeout means the command expired before completion.
	CommandPhaseTimeout CommandPhase = "Timeout"
)

// VehicleCommandStatus defines the observed state of VehicleCommand.
type VehicleCommandStatus struct {
	// Phase represents the current high-level stage of the command lifecycle.
	// +optional
	Phase CommandPhase `json:"phase,omitempty"`

	// Message provides human-readable details about the current status or error reason.
	// +optional
	Message string `json:"message,omitempty"`

	// Result holds the output data.
	// WARNING: Do NOT store large binaries or logs here.
	// Use strictly for references (e.g., {"status": "ok", "report_url": "s3://bucket/log.txt"}).
	// Etcd limit is 1.5MB, keeping this small is crucial.
	// +optional
	Result map[string]string `json:"result,omitempty"`

	// LastUpdateTime captures the timestamp of the most recent status change.
	// Useful for detecting stalled commands or debugging controller latency.
	// +optional
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// StartTime is when the controller first saw the command.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// SentTime records the exact moment when the command was successfully published to the MQTT broker (EMQX).
	// Comparing (SentTime - StartTime) reveals the internal processing latency of the CloudHub.
	// +optional
	SentTime *metav1.Time `json:"sentTime,omitempty"`

	// AcknowledgeTime marks when the Vehicle Agent confirmed receipt of the command (PUBACK or equivalent).
	// Comparing (AcknowledgeTime - SentTime) reveals the network latency and connectivity health.
	// +optional
	AcknowledgeTime *metav1.Time `json:"acknowledgeTime,omitempty"`

	// CompletionTime marks when the command reached a terminal state (Succeeded, Failed, or Timeout).
	// This timestamp effectively closes the SLA window for the operation.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Conditions provide a detailed history of the command's progress (e.g., Downloaded, Verified).
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="Vehicle",type="string",JSONPath=".spec.vehicleName",description="Target Vehicle"
//+kubebuilder:printcolumn:name="Method",type="string",JSONPath=".spec.method",description="Command Method"
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
