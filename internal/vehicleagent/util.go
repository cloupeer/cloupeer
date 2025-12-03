package vehicleagent

import (
	"os"
	"strings"

	"cloupeer.io/cloupeer/pkg/log"
)

// DiscoverVehicleID 尝试从系统环境中自动获取车辆 ID
// 最佳实践：应该有一个特权级的初始化进程（Init Container 或 Systemd Service）负责调用底层驱动读取 VIN，
// 然后将其写入文件 /etc/cloupeer/vin 或注入环境变量 CPEER_VEHICLE_ID。
// 好处： 这样你的 Agent 可以在任何硬件上运行（只要配置对了），而不需要把复杂的 CAN 协议解析代码耦合进 Agent 核心逻辑里。
func DiscoverVehicleID() string {
	// 1. 尝试读取环境变量 (生产环境容器注入)
	if envID := os.Getenv("CPEER_VEHICLE_ID"); envID != "" {
		log.Info("VehicleID detected from env", "id", envID)
		return envID
	}

	// 2. 尝试读取硬件文件 (模拟)
	// 在真实 Linux 车机中，这可能是 /sys/class/dmi/id/product_serial
	// 或者 /etc/machine-id，或者调用专门的 CAN Bus SDK
	// 这里我们模拟读取一个文件
	if content, err := os.ReadFile("/etc/cloupeer/vin"); err == nil {
		id := strings.TrimSpace(string(content))
		if id != "" {
			log.Info("VehicleID detected from file", "id", id)
			return id
		}
	}

	// 3. 如果都失败了，返回空
	return ""
}
