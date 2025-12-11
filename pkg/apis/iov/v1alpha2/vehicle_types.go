package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VehicleFinalizer allows the controller to clean up resources (e.g. remove from EMQX authentication) before deletion.
const VehicleFinalizer = "iov.cloupeer.io/vehicle-finalizer"

// VehicleSpec defines the desired state of Vehicle.
// It focuses on high-level configuration and desired properties (Twins).
type VehicleSpec struct {
	// VIN (Vehicle Identification Number) is the unique business identifier.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-HJ-NPR-Z0-9]{17}$`
	VIN string `json:"vin"`

	// VehicleModelRef references the definition of the vehicle model (e.g., "tesla-model-3-v1").
	// This allows future validation of supported properties.
	// +optional
	VehicleModelRef string `json:"vehicleModelRef,omitempty"`

	// Access contains the connectivity credentials and protocol settings
	// used by CloudHub to manage the device connection.
	// +optional
	Access AccessConfig `json:"access,omitempty"`

	// Profile defines the core operational configuration of the vehicle.
	// This struct is SHARED between Spec (Desired) and Status (Reported).
	// It enables simple diffing logic: if Spec.Profile != Status.Profile, sync is needed.
	// +optional
	Profile VehicleProfile `json:"profile,omitempty"`

	// Properties holds the list of dynamic extension attributes.
	// Used for non-core, model-specific features (e.g., "ambient_light_color": "blue").
	// +optional
	Properties map[string]string `json:"properties,omitempty"`
}

// AccessConfig defines the connectivity parameters.
type AccessConfig struct {
	// ClientID is the MQTT client identifier.
	// If empty, defaults to the Kubernetes metadata.name.
	// +optional
	ClientID string `json:"clientID,omitempty"`

	// AuthSecretRef references the secret containing client certificate or password.
	// Using LocalObjectReference ensures security by restricting to the same namespace.
	// +optional
	AuthSecretRef *corev1.LocalObjectReference `json:"authSecretRef,omitempty"`
}

// VehicleProfile is the SHARED struct for both Desired (Spec) and Reported (Status) states.
// All fields here represent "Stateful Configurations".
type VehicleProfile struct {
	// Firmware describes the software version and download location.
	// In Spec: The version/url we want the vehicle to have.
	// In Status: The version/url the vehicle currently has installed.
	// +optional
	Firmware FirmwareConfig `json:"firmware,omitempty"`

	// OTAPolicy defines the safeguards for Over-The-Air updates.
	// In Spec: The policy we want to enforce.
	// In Status: The policy currently active on the agent.
	// +optional
	OTAPolicy OTAPolicy `json:"otaPolicy,omitempty"`

	// MaxSpeedLimit defines the safety speed cap in km/h.
	// Using pointer to distinguish between "0" (stop) and "unset" (no limit).
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=500
	// +optional
	MaxSpeedLimit *int32 `json:"maxSpeedLimit,omitempty"`

	// EnableEdgeCompute toggles the edge computing capabilities on the vehicle unit.
	// +optional
	EnableEdgeCompute *bool `json:"enableEdgeCompute,omitempty"`
}

// FirmwareConfig defines the details of the software bundle.
type FirmwareConfig struct {
	// Version is the semantic version of the bundle (e.g., "1.2.0-beta.1").
	// +optional
	Version string `json:"version,omitempty"`

	// DownloadURL is the location of the firmware bundle (S3/MinIO link).
	// +optional
	DownloadURL string `json:"downloadURL,omitempty"`

	// Checksum ensures the integrity of the binary (e.g., "sha256:xxxx").
	// +optional
	Checksum string `json:"checksum,omitempty"`
}

// OTAPolicy defines safety constraints for updates.
type OTAPolicy struct {
	// MinBatteryLevel defines the minimum battery percentage (0-100) required to start an OTA.
	// This is a POLICY, not the current battery level.
	// +kubebuilder:validation:Minimum=30
	// +kubebuilder:validation:Maximum=100
	// +optional
	MinBatteryLevel *int32 `json:"minBatteryLevel,omitempty"`

	// RetryLimit defines how many times the agent should retry a failed update.
	// +optional
	RetryLimit *int32 `json:"retryLimit,omitempty"`
}

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

// Condition Types for Vehicle
const (
	// ConditionTypeReady indicates if the vehicle controller is functioning properly for this resource.
	ConditionTypeReady = "Ready"

	// ConditionTypeSynced indicates if the Vehicle's reported state matches the desired Spec.
	ConditionTypeSynced = "Synced"
)

// VehicleStatus defines the observed state of Vehicle.
type VehicleStatus struct {
	// Online status derived from heartbeats.
	// +optional
	Online bool `json:"online"`

	// LastHeartbeatTime (Networking level).
	// +optional
	LastHeartbeatTime *metav1.Time `json:"lastHeartbeatTime,omitempty"`

	// Profile represents the actual configuration reported by the vehicle.
	// The Controller compares Spec.Profile vs Status.Profile to determine 'Synced' condition.
	// +optional
	Profile VehicleProfile `json:"profile,omitempty"`

	// Properties holds the map of dynamic extension attributes (Reported State).
	// Symmetric to Spec.Properties.
	// +optional
	Properties map[string]string `json:"properties,omitempty"`

	// UpgradeStatus tracks the PROGRESS of the current firmware installation.
	// This separates "Configuration" (Profile) from "Execution State" (RetryCount).
	// +optional
	UpgradeStatus UpgradeStatus `json:"upgradeStatus,omitempty"`

	// Conditions represent the latest available observations of the Vehicle's state (e.g., Ready, Synced).
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// UpgradeStatus isolates the transient state of an OTA process.
type UpgradeStatus struct {
	// The last reported phase of the vehicle's OTA status.
	// +optional
	Phase VehiclePhase `json:"phase,omitempty"`

	// RetryCount tracks execution attempts.
	// Compared against Spec.Profile.OTAPolicy.RetryLimit by the Agent/Controller.
	// +optional
	RetryCount int32 `json:"retryCount,omitempty"`

	// LastError stores the last failure reason for debugging.
	// +optional
	LastError string `json:"lastError,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="VIN",type="string",JSONPath=".spec.vin",description="Vehicle Identification Number"
//+kubebuilder:printcolumn:name="Model",type="string",JSONPath=".spec.vehicleModelRef",description="Vehicle Model"
//+kubebuilder:printcolumn:name="Online",type="boolean",JSONPath=".status.online",description="Connection Status"
//+kubebuilder:printcolumn:name="Desired",type="string",JSONPath=".spec.profile.firmware.version",description="Target Firmware Version"
//+kubebuilder:printcolumn:name="Reported",type="string",JSONPath=".status.profile.firmware.version",description="Current Firmware Version"
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.upgradeStatus.phase",description="OTA Phase"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

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
