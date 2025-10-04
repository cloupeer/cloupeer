/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PhysicalDeviceSpec defines the desired state of PhysicalDevice
// 这就是 "Telos" (目标/终极目的)
type PhysicalDeviceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// MAC 地址，设备的唯一物理标识
	// +kubebuilder:validation:Required
	MACAddress string `json:"macAddress"`

	// 设备的期望配置，可以是 JSON 字符串或者一个更复杂的结构
	// 例如：{"led": "on", "sensor_interval": "5s"}
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Optional
	DesiredConfig *runtime.RawExtension `json:"desiredConfig,omitempty"`

	// 期望的固件版本
	// +kubebuilder:validation:Optional
	FirmwareVersion string `json:"firmwareVersion,omitempty"`

	// 期望设备所处的状态，例如 "active", "maintenance", "offline"
	// +kubebuilder:validation:Enum=active;maintenance;offline
	// +kubebuilder:default:=active
	State string `json:"state,omitempty"`
}

// PhysicalDeviceStatus defines the observed state of PhysicalDevice
// 这是系统持续观测到的真实状态
type PhysicalDeviceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// 设备的连接状态，例如 "Connected", "Disconnected"
	ConnectivityStatus string `json:"connectivityStatus,omitempty"`

	// 设备上报的当前固件版本
	ReportedFirmwareVersion string `json:"reportedFirmwareVersion,omitempty"`

	// 设备最后一次心跳的时间戳
	LastHeartbeatTime *metav1.Time `json:"lastHeartbeatTime,omitempty"`

	// 设备的当前状态条件，这是一种更 Kubernetes-native 的状态表达方式
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PhysicalDevice is the Schema for the physicaldevices API
type PhysicalDevice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PhysicalDeviceSpec   `json:"spec,omitempty"`
	Status PhysicalDeviceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PhysicalDeviceList contains a list of PhysicalDevice
type PhysicalDeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PhysicalDevice `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PhysicalDevice{}, &PhysicalDeviceList{})
}
