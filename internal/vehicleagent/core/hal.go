package core

// HAL (Hardware Abstraction Layer) 定义了 Agent 与底层操作系统/硬件交互的标准接口。
// 在六边形架构中，这是 "Driven Port" (出站端口)。
type HAL interface {
	// Info 接口：获取设备静态信息
	GetVehicleID() string
	GetFirmwareVersion() string

	// Safety 接口：安全门禁检查 (P档, 零速, 电量等)
	// 如果不满足安全条件，返回 error
	CheckSafety() error

	// Action 接口：执行具体操作
	// MarkBootSuccessful 通知 Bootloader 当前启动成功 (防回滚)
	MarkBootSuccessful() error

	// InstallFirmware 将固件写入闲置分区 (Slot B)
	InstallFirmware(imagePath string) error

	// SwitchBootSlot 切换启动分区标志位
	SwitchBootSlot() error

	// Reboot 执行系统重启
	Reboot() error
}
