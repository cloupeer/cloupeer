//go:build linux

package hal

import (
	"os"
	"strings"
	"syscall"

	"cloupeer.io/cloupeer/internal/vehicleagent/core"
	"cloupeer.io/cloupeer/pkg/log"
)

// LinuxHAL 是真实车机环境的适配器
type LinuxHAL struct{}

func NewHAL() core.HAL {
	return &LinuxHAL{}
}

func (h *LinuxHAL) GetVehicleID() string {
	// 真实：读取 /etc/machine-id 或专门的 VIN 码文件
	data, _ := os.ReadFile("/etc/cloupeer/vin")
	return strings.TrimSpace(string(data))
}

func (h *LinuxHAL) GetFirmwareVersion() string {
	// 真实：读取 /etc/os-release 中的 VERSION_ID
	// 这里简化为读取一个固定文件
	data, _ := os.ReadFile("/etc/cloupeer/version")
	return strings.TrimSpace(string(data))
}

func (h *LinuxHAL) CheckSafety() error {
	// 真实：读取 /sys/class/... 下的传感器文件
	// 或者调用 Cgo 库
	return nil // 暂略
}

func (h *LinuxHAL) MarkBootSuccessful() error {
	// 真实：调用 bootloader 接口
	// syscall.Exec("/usr/sbin/bootctl", "mark-successful")
	return nil
}

func (h *LinuxHAL) InstallFirmware(path string, version string) error {
	// 真实：调用 swupdate 或 dd 命令
	return nil
}

func (h *LinuxHAL) SwitchBootSlot() error {
	// 真实：修改 u-boot 环境变量
	return nil
}

func (h *LinuxHAL) Reboot() error {
	log.Info("System is rebooting NOW...")
	syscall.Sync()
	return syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
}
