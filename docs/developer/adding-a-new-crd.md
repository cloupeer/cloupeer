# 如何向 Cloupeer 添加一个新的 CRD

本文档详细介绍了向 Cloupeer 项目添加一个新的 Custom Resource Definition (CRD) 的标准流程。我们将以 `Vehicle` CRD 为例进行说明。

这个流程遵循 "API-First" 的原则：开发者首先定义 API 的数据结构，然后利用项目内置的工具链自动生成所需的模板代码和 Kubernetes 清单文件。

## 先决条件

- 你已经成功配置了本地开发环境。
- 你已经安装了项目所需的所有工具（通过 `make` 命令会自动安装）。

## 流程步骤

### 第 1 步：定义 API 的 Go 类型

所有的 API 类型定义都存放在 `pkg/apis/` 目录下。你需要为你的新 API 创建对应的 Group 和 Version 目录，并添加两个 Go 文件。

假设我们要创建的 CRD 是：
- **Group:** `iov.cloupeer.io`
- **Version:** `v1alpha2`
- **Kind:** `Vehicle`

#### 1.1 创建 groupversion_info.go

这个文件用于向 `controller-runtime` 注册你的 API Group 和 Version。

创建文件：`pkg/apis/iov/v1alpha2/groupversion_info.go`

```go
// +k8s:deepcopy-gen=package
// +groupName=iov.cloupeer.io
package v1alpha2

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "iov.cloupeer.io", Version: "v1alpha2"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
```

#### 1.2 创建 *_types.go

这个文件是核心，用于定义 CRD 的 `Spec`（期望状态）和 `Status`（实际状态），以及 `Kind` 和 `List` 结构体。

创建文件：`pkg/apis/iov/v1alpha2/vehicle_types.go`

```go
package v1alpha2

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
```

**重要提示：**

  - 文件顶部的 `// +groupName` 注释是必需的。
  - 结构体上方的 `//+kubebuilder:` 注释是“魔法注释”，`controller-gen` 会读取它们来生成代码和 YAML。请参考 [Kubebuilder Markers](https://book.kubebuilder.io/reference/markers.html) 官方文档了解更多可用选项。

### 第 2 步：自动生成代码和清单文件

在你完成了 Go 类型的定义后，只需在项目根目录运行一个命令：

```bash
make manifests generate
```

这个命令会触发 `hack/make-rules/generate.sh` 脚本，并完成以下所有工作：

1.  **生成 DeepCopy 方法：** 在你的 `v1alpha2` 目录下创建一个 `zz_generated.deepcopy.go` 文件。你的 API 类型必须实现 `runtime.Object` 接口，这些方法是必需的。
2.  **生成 CRD 清单：** 在 `manifests/base/crd/bases/` 目录下生成 `xxx.cloupeer.io_xxxxxxx.yaml` 文件。
3.  **更新 Kustomization：** 自动将新生成的 CRD 文件名添加到 `manifests/base/crd/bases/kustomization.yaml` 中。


### 第 3 步：实现 Controller 逻辑

代码生成后，你需要为新的 CRD 编写业务逻辑，即 `Controller`。

1. 在 `internal/controller/` 目录下为你的 CRD 创建一个新目录，例如 `internal/controller/vehicle/`。
2. 在该目录下创建一个 `vehiclecontroller.go` 文件，并编写你的 `Reconcile` 循环。
3. 在 Reconcile 方法上添加 `//+kubebuilder:rbac:groups=iov.cloupeer.io,resources=vehicles,...` 标记。
4. 打开 `internal/controller/manager.go`，在 `setupControllers` 函数中初始化并注册你的新 Controller。

然后，再次执行：

```bash
make manifests generate
```

这次会**更新 RBAC 权限：** 更新 `manifests/components/cpeer-controller-manager/base/generated.manager-role.yaml` 文件，为 controller-manager 添加操作新 `Vehicle` 资源的 ClusterRole 权限。


### 第 4 步：验证

完成以上步骤后，你可以部署到本地集群来验证你的工作。

```bash
# Install the new CRD into your cluster
make install

# Deploy the controller manager with the new controller logic
make deploy ENV=dev COMPONENT=cpeer-controller-manager
```

至此，你已经成功地向 Cloupeer 项目添加了一个新的 CRD。
