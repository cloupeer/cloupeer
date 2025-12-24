//go:build !linux

package hal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/autopeer-io/autopeer/internal/agent/core"
	"github.com/autopeer-io/autopeer/pkg/log"
)

const (
	fileCurrentVersion = "current_version"
	filePendingVersion = "pending_version"
)

var (
	count int
	mu    sync.Mutex
)

type MockHAL struct {
	vid     string
	baseDir string
}

func NewHAL() core.HAL {
	vid := os.Getenv("AUTOPEER_VEHICLE_ID")
	if vid == "" {
		mu.Lock()
		count++
		timestampPart := fmt.Sprintf("%08d", time.Now().Unix()%100000000)
		vid = fmt.Sprintf("MVH%s%06d", timestampPart, count)
		mu.Unlock()
	}

	baseDir := filepath.Join(os.TempDir(), "autopeer-devices", vid)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to provision mock storage for %s: %v", vid, err))
	}

	h := &MockHAL{vid: vid, baseDir: baseDir}
	if err := h.ensureFactoryFirmware(); err != nil {
		panic(fmt.Sprintf("failed to ensure factory firware version for %s: %v", vid, err))
	}

	return h
}

func (h *MockHAL) ensureFactoryFirmware() error {
	verFile := filepath.Join(h.baseDir, fileCurrentVersion)
	if _, err := os.Stat(verFile); err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(verFile, []byte("v1.0.0"), 0644)
		}
		return err
	}
	return nil
}

func (h *MockHAL) GetVehicleID() string {
	return h.vid
}

func (h *MockHAL) GetFirmwareVersion() string {
	data, err := os.ReadFile(filepath.Join(h.baseDir, fileCurrentVersion))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (h *MockHAL) CheckSafety() error {
	log.Info("[HAL-Mock] Checking safety gates... (Gear=P, Speed=0, Battery=80%)")
	time.Sleep(500 * time.Millisecond) // 模拟传感器读取耗时
	return nil
}

func (h *MockHAL) MarkBootSuccessful() error {
	log.Info("[HAL-Mock] Bootloader marked as SUCCESSFUL. Rollback counter cleared.")
	return nil
}

func (h *MockHAL) InstallFirmware(path string, version string) error {
	log.Info("[HAL-Mock] Writing firmware to inactive slot (Slot B)...", "vid", h.vid, "path", path, "version", version)
	for i := 0; i < 5; i++ {
		log.Info(fmt.Sprintf("[HAL-Mock] Flashing... %d%%", (i+1)*20))
		time.Sleep(1 * time.Second)
	}

	pendingFile := filepath.Join(h.baseDir, filePendingVersion)
	return os.WriteFile(pendingFile, []byte(version), 0644)
}

func (h *MockHAL) SwitchBootSlot() error {
	log.Info("[HAL-Mock] Switching active slot to Slot B.")
	return nil
}

func (h *MockHAL) Reboot() error {
	log.Warn("[HAL-Mock] >>> REBOOT REQUESTED <<<")
	log.Warn("[HAL-Mock] System will 'restart' in 3 seconds...")
	time.Sleep(3 * time.Second)

	pendingFile := filepath.Join(h.baseDir, filePendingVersion)
	currentFile := filepath.Join(h.baseDir, fileCurrentVersion)

	if data, err := os.ReadFile(pendingFile); err == nil {
		newVer := string(data)

		if err := os.WriteFile(currentFile, data, 0644); err != nil {
			log.Error(err, "Bootloader failed to load new kernel")
			return err
		}

		_ = os.Remove(pendingFile)

		log.Info(fmt.Sprintf(">>> [BOOT] %s started with NEW VERSION: %s <<<", h.vid, newVer))
	} else {
		log.Info(fmt.Sprintf(">>> [BOOT] %s started (Normal Boot) <<<", h.vid))
	}

	return nil
}
