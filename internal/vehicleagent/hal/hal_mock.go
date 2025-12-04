//go:build !linux

package hal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloupeer.io/cloupeer/internal/vehicleagent/core"
	"cloupeer.io/cloupeer/pkg/log"
)

// MockHAL 是开发环境下的模拟实现
type MockHAL struct {
	baseDir string
}

func NewHAL() core.HAL {
	// 使用临时目录模拟车机分区
	tmpDir := filepath.Join(os.TempDir(), "cloupeer-mock-hal")
	_ = os.MkdirAll(tmpDir, 0755)
	return &MockHAL{baseDir: tmpDir}
}

func (h *MockHAL) GetVehicleID() string {
	// 模拟：优先读环境变量，否则返回模拟 ID
	if envID := os.Getenv("CPEER_VEHICLE_ID"); envID != "" {
		return envID
	}
	return "vh-mock-mac-001"
}

func (h *MockHAL) GetFirmwareVersion() string {
	// 模拟：从临时文件读取版本号，模拟持久化存储
	verFile := filepath.Join(h.baseDir, "current_version")
	data, err := os.ReadFile(verFile)
	if err != nil {
		return "v1.0.0" // 默认出厂版本
	}
	return strings.TrimSpace(string(data))
}

func (h *MockHAL) CheckSafety() error {
	// 模拟：随机或通过配置文件控制安全状态
	// 这里我们假设永远安全，方便测试流程
	log.Info("[HAL-Mock] Checking safety gates... (Gear=P, Speed=0, Battery=80%)")
	time.Sleep(500 * time.Millisecond) // 模拟传感器读取耗时
	return nil
}

func (h *MockHAL) MarkBootSuccessful() error {
	log.Info("[HAL-Mock] Bootloader marked as SUCCESSFUL. Rollback counter cleared.")
	return nil
}

func (h *MockHAL) InstallFirmware(imagePath string) error {
	log.Info("[HAL-Mock] Writing firmware to inactive slot (Slot B)...", "path", imagePath)
	// 模拟写入耗时
	for i := 0; i < 5; i++ {
		log.Info(fmt.Sprintf("[HAL-Mock] Flashing... %d%%", (i+1)*20))
		time.Sleep(1 * time.Second)
	}
	return nil
}

func (h *MockHAL) SwitchBootSlot() error {
	log.Info("[HAL-Mock] Switching active slot to Slot B.")
	return nil
}

func (h *MockHAL) Reboot() error {
	log.Warn("[HAL-Mock] >>> REBOOT REQUESTED <<<")
	log.Warn("[HAL-Mock] System will 'restart' in 3 seconds...")
	time.Sleep(3 * time.Second)

	// 模拟：既然不能真重启 Mac，我们在这里更新一下 Mock 的版本号文件
	// 这样下次 Agent 启动（或者如果你手动重启程序）就能看到新版本
	// 这里假设升级目标是 v2.0.0，实际工程中应该从 InstallFirmware 的元数据里取
	// 为了演示简单，我们简单地把版本号 +Patch
	_ = os.WriteFile(filepath.Join(h.baseDir, "current_version"), []byte("v1.0.1-upgraded"), 0644)

	log.Info("[HAL-Mock] >>> MOCK REBOOT COMPLETE. Please restart the agent manually to see new version. <<<")
	return nil
}
