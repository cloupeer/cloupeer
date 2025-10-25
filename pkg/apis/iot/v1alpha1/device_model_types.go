package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// DeviceModelSpec defines the desired state of DeviceModel.
// It serves as a template that describes the capabilities and protocol
// used by a class of devices.
type DeviceModelSpec struct {
	// Properties defines the common properties possessed by this type of device.
	// +optional
	Properties []ModelProperty `json:"properties,omitempty"`

	// Protocol indicates the general protocol name used by devices of this model.
	// Specific connection details are in the Device CR.
	// +optional
	Protocol string `json:"protocol,omitempty"`
}

// ModelProperty describes an individual device property defined within a DeviceModel.
// It specifies the metadata and capabilities of a property, like its name, type, and access mode.
type ModelProperty struct {
	// Name of the property. It should be unique within the DeviceModel.
	// This name is used to associate DeviceProperty instances in Device CRs.
	Name string `json:"name,omitempty"`

	// Description is a human-readable description of the property.
	// +optional
	Description string `json:"description,omitempty"`

	// Type specifies the data type of the property.
	Type PropertyType `json:"type,omitempty"`

	// AccessMode defines if the property is ReadOnly or ReadWrite (control).
	AccessMode PropertyAccessMode `json:"accessMode,omitempty"`

	// Minimum value constraint for the property (interpreted based on Type).
	// +optional
	Minimum string `json:"minimum,omitempty"`

	// Maximum value constraint for the property (interpreted based on Type).
	// +optional
	Maximum string `json:"maximum,omitempty"`

	// Unit of the property value (e.g., "°C", "%", "ppm").
	// +optional
	Unit string `json:"unit,omitempty"`
}

// PropertyType defines the data type of a device property.
// +kubebuilder:validation:Enum=INT;FLOAT;DOUBLE;STRING;BOOLEAN;BYTES;STREAM
type PropertyType string

// Constants representing supported property types.
const (
	INT     PropertyType = "INT"
	FLOAT   PropertyType = "FLOAT"
	DOUBLE  PropertyType = "DOUBLE"
	STRING  PropertyType = "STRING"
	BOOLEAN PropertyType = "BOOLEAN"
	BYTES   PropertyType = "BYTES"
	STREAM  PropertyType = "STREAM"
)

// PropertyAccessMode defines whether a device property is read-only or read-write.
// +kubebuilder:validation:Enum=ReadWrite;ReadOnly
type PropertyAccessMode string

// Constants representing property access modes.
const (
	ReadWrite PropertyAccessMode = "ReadWrite"
	ReadOnly  PropertyAccessMode = "ReadOnly"
)

// DeviceModel is the Schema for the devicemodels API.
// It represents a template or blueprint for a class of devices sharing common properties and protocol.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DeviceModel struct {
	metav1.TypeMeta `json:",inline"`
	// Standard Kubernetes object metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the DeviceModel.
	Spec DeviceModelSpec `json:"spec"`
}

// DeviceModelList contains a list of DeviceModel.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DeviceModelList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard Kubernetes list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of DeviceModel resources.
	Items []DeviceModel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeviceModel{}, &DeviceModelList{})
}
