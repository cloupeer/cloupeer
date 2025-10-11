// Copyright 2025 Anankix.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	edgev1alpha1 "github.com/anankix/anankix/pkg/apis/edge/v1alpha1"
	"github.com/anankix/anankix/pkg/log"
)

const versionFile = "/tmp/anx_agent_version.txt"

// AgentState holds the runtime state of the agent.
type AgentState struct {
	DeviceID               string
	GatewayURL             string
	CurrentFirmwareVersion string
	Client                 *http.Client
}

// HeartbeatPayload is the data sent in a heartbeat.
type HeartbeatPayload struct {
	DeviceID               string `json:"deviceId"`
	CurrentFirmwareVersion string `json:"currentFirmwareVersion"`
}

// TaskStatusUpdatePayload is the data sent to update a task's status.
type TaskStatusUpdatePayload struct {
	Phase   edgev1alpha1.TaskPhase `json:"phase"`
	Message string                 `json:"message,omitempty"`
}

// readVersionFromFile reads the current version from the local file.
func readVersionFromFile() string {
	data, err := os.ReadFile(versionFile)
	if err != nil {
		// If file doesn't exist, assume initial version
		log.Warn("Version file not found, defaulting to v1.0.0", "file", versionFile, err)
		return "v1.0.0"
	}
	return string(data)
}

// writeVersionToFile persists the new version to the local file.
func writeVersionToFile(version string) error {
	log.Info("Persisting new version to file", "version", version)
	return os.WriteFile(versionFile, []byte(version), 0644)
}

// sendHeartbeat sends a heartbeat to the gateway.
func (s *AgentState) sendHeartbeat() {
	hb := HeartbeatPayload{
		DeviceID:               s.DeviceID,
		CurrentFirmwareVersion: s.CurrentFirmwareVersion,
	}
	body, _ := json.Marshal(hb)
	url := fmt.Sprintf("%s/heartbeat", s.GatewayURL)
	resp, err := s.Client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Error(err, "Heartbeat failed")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Warn("Heartbeat returned non-2xx status", "status", resp.Status)
	} else {
		log.Info("Heartbeat sent successfully", "version", s.CurrentFirmwareVersion)
	}
}

// checkForTask fetches a task from the gateway.
func (s *AgentState) checkForTask() *edgev1alpha1.FirmwareUpgradeTask {
	url := fmt.Sprintf("%s/tasks/%s", s.GatewayURL, s.DeviceID)
	resp, err := s.Client.Get(url)
	if err != nil {
		log.Error(err, "Failed to check for tasks")
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		log.Warn("No pending tasks.")
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		log.Warn("Check for tasks returned non-200 status", "status", resp.Status)
		return nil
	}

	var task edgev1alpha1.FirmwareUpgradeTask
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		log.Error(err, "Failed to decode task")
		return nil
	}

	log.Info("Received task to upgrade to version", "taskName", task.Name, "version", task.Spec.Firmware.Version)
	return &task
}

// updateTaskStatus reports the task's progress to the gateway.
func (s *AgentState) updateTaskStatus(taskName string, phase edgev1alpha1.TaskPhase, message string) {
	payload := TaskStatusUpdatePayload{Phase: phase, Message: message}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/tasks/%s/status", s.GatewayURL, taskName)

	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		log.Error(err, "Failed to update task status", "taskName", taskName)
		return
	}
	defer resp.Body.Close()
	log.Info("Updated task status successfully", "taskName", taskName, "phase", phase)
}

// executeUpgrade simulates the firmware upgrade process.
func (s *AgentState) executeUpgrade(task *edgev1alpha1.FirmwareUpgradeTask) {
	// 1. Report Downloading
	s.updateTaskStatus(task.Name, edgev1alpha1.TaskPhaseDownloading, "")

	// 2. Simulate download and checksum verification
	log.Debug("SIMULATE: Downloading firmware", "url", task.Spec.Firmware.URL)
	time.Sleep(2 * time.Second) // Simulate download time

	// This is where you would do a real http.Get and sha256.Sum256
	// For the simulation, we assume it's always correct.
	log.Debug(fmt.Sprintf("SIMULATE: Verifying checksum %s... OK", task.Spec.Firmware.Checksum))

	// 3. Report Installing
	s.updateTaskStatus(task.Name, edgev1alpha1.TaskPhaseInstalling, "")
	time.Sleep(2 * time.Second) // Simulate install time

	// 4. Persist new version to simulate reboot
	if err := writeVersionToFile(task.Spec.Firmware.Version); err != nil {
		log.Error(err, "Failed to write new version file")
		s.updateTaskStatus(task.Name, edgev1alpha1.TaskPhaseFailed, err.Error())
		return
	}

	// 5. Update in-memory state and report success
	s.CurrentFirmwareVersion = task.Spec.Firmware.Version
	log.Info(fmt.Sprintf("SUCCESS: Upgrade to %s complete.", s.CurrentFirmwareVersion))
	s.updateTaskStatus(task.Name, edgev1alpha1.TaskPhaseSucceeded, "")
}

func main() {
	log.Init(log.NewOptions())

	deviceID := os.Getenv("DEVICE_ID")
	if deviceID == "" {
		deviceID = "device-local-001" // Use a more descriptive default
	}
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:9090"
	}

	state := &AgentState{
		DeviceID:               deviceID,
		GatewayURL:             gatewayURL,
		CurrentFirmwareVersion: readVersionFromFile(),
		Client:                 &http.Client{Timeout: 10 * time.Second},
	}

	log.Info("anx-edge-agent started", "deviceId", state.DeviceID, "gateway", state.GatewayURL, "initialVersion", state.CurrentFirmwareVersion)

	// Main agent loop
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for ; ; <-ticker.C {
		fmt.Println()
		log.Info("--- Agent loop start ---")
		state.sendHeartbeat()
		if task := state.checkForTask(); task != nil {
			state.executeUpgrade(task)
			// After an upgrade, immediately restart the loop to send a new heartbeat with the updated version
			log.Info("--- Upgrade finished, restarting loop immediately ---")
			continue
		}
		log.Info("--- Agent loop end ---")
	}
}
