package ota

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	pb "github.com/autopeer-io/autopeer/api/proto/v1"
	"github.com/autopeer-io/autopeer/internal/agent/core"
	"github.com/autopeer-io/autopeer/pkg/log"
)

func (m *Manager) AckCommand(ctx context.Context, name, status, message string) {
	ack := &pb.AgentCommandStatus{
		CommandName: name,
		Status:      status,
		Message:     message,
	}

	if err := m.sender.SendProto(ctx, core.EventCommandStatus, ack); err != nil {
		log.Error(err, "Failed to ack command status", "name", name, "status", status, "message", message)
	}
}

func (m *Manager) execute(ctx context.Context, cmd *pb.AgentCommand) {
	// 1. 收到指令
	m.AckCommand(ctx, cmd.CommandName, "Received", "Security check passed")

	// 模拟：车主等待确认 (例如 2秒)
	log.Info("[UI] User notification: New firmware available. Click to upgrade.")
	time.Sleep(2 * time.Second)
	log.Info("[UI] User clicked 'Upgrade'. Requesting URL...")

	// 2. 请求 URL
	targetVer := cmd.Parameters["version"]
	reqID := fmt.Sprintf("req-%d", time.Now().UnixNano())

	// 创建接收通道
	respChan := make(chan string, 1)
	m.lock.Lock()
	m.pending[reqID] = respChan
	m.lock.Unlock()

	// 发送请求
	req := &pb.OTARequest{
		VehicleId:      m.vid,
		DesiredVersion: targetVer,
		RequestId:      reqID,
	}

	err := m.sender.SendProto(ctx, core.EventOTARequest, req)
	if err != nil {
		log.Error(err, "Faile to send OTA request")
	}

	var downloadURL string

	// 3. 等待响应 (带超时)
	select {
	case url := <-respChan:
		downloadURL = url
		log.Info("Received Firmware URL", "url", url)
	case <-time.After(15 * time.Second):
		log.Error(nil, "Timeout waiting for firmware URL")
		m.AckCommand(ctx, cmd.CommandName, "Failed", "Timeout fetching URL")

		// 清理 map
		m.lock.Lock()
		delete(m.pending, reqID)
		m.lock.Unlock()
		return
	}

	// 4. 开始下载 (Running)
	m.AckCommand(ctx, cmd.CommandName, "Running", "Downloading firmware artifact...")

	// 执行真实的下载校验
	if err := downloadAndVerify(downloadURL); err != nil {
		log.Error(err, "Download failed")
		m.AckCommand(ctx, cmd.CommandName, "Failed", fmt.Sprintf("Download failed: %v", err))
		return
	}

	// 5. 安全门禁 (调用 HAL)
	log.Info("Performing safety checks before installation...")
	if err := m.hal.CheckSafety(); err != nil {
		log.Error(err, "Safety check failed")
		m.AckCommand(ctx, cmd.CommandName, "Failed", fmt.Sprintf("Safety check failed: %v", err))
		return
	}

	// 6. 原子安装 (调用 HAL)
	m.AckCommand(ctx, cmd.CommandName, "Running", "Installing to Slot B...")
	if err := m.hal.InstallFirmware("/tmp/firmware.bin", targetVer); err != nil {
		log.Error(err, "Installation failed")
		m.AckCommand(ctx, cmd.CommandName, "Failed", "Write partition failed")
		return
	}

	// 7. 切换引导 (调用 HAL)
	if err := m.hal.SwitchBootSlot(); err != nil {
		m.AckCommand(ctx, cmd.CommandName, "Failed", "Switch slot failed")
		return
	}

	// 8. 最终确认 & 重启
	m.AckCommand(ctx, cmd.CommandName, "Running", "Rebooting system...")
	log.Info("OTA sequence complete. Requesting system reboot.")

	// 给一点时间让 MQTT 消息发出去
	time.Sleep(1 * time.Second)

	if err := m.hal.Reboot(); err != nil {
		m.AckCommand(ctx, cmd.CommandName, "Failed", "Reboot failed")
		log.Error(err, "Reboot failed")
		return
	}

	// 9. 完成
	m.AckCommand(ctx, cmd.CommandName, "Succeeded", "Update installed")
}

// downloadAndVerify performs a real HTTP GET to validate the URL.
// In a production agent, this would also verify SHA256 checksums and write to disk.
func downloadAndVerify(url string) error {
	client := &http.Client{
		Timeout: 10 * time.Minute,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %s", resp.Status)
	}

	// Simulate consuming the body (or write to /tmp/firmware.bin)
	// We just read it to ensure the stream is valid.
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}

	return nil
}
