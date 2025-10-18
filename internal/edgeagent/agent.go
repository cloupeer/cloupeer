package edgeagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	firmwarev1alpha1 "cloupeer.io/cloupeer/pkg/apis/firmware/v1alpha1"
	"cloupeer.io/cloupeer/pkg/log"
)

// Agent is the core struct for the edge agent business logic.
type Agent struct {
	deviceID    string
	heartbeat   *heartbeat
	state       *state
	versionFile string
	client      *http.Client

	upgradeLocker sync.Mutex // A mutex to protect the isUpgrading flag
	isUpgrading   bool       // A flag to indicate if an upgrade is in progress
}

type heartbeat struct {
	url      string
	interval time.Duration
}

type state struct {
	firmwareVersion string
}

// Run starts the main loop of the agent and handles graceful shutdown.
func (a *Agent) Run(ctx context.Context) error {
	log.Info("Starting cpeer-edge-agent", "deviceID", a.deviceID, "gatewayURL", a.heartbeat.url, "initialVersion", a.state.firmwareVersion)

	ticker := time.NewTicker(a.heartbeat.interval)
	defer ticker.Stop()

	// Perform an initial run immediately on startup.
	a.runOnce(ctx)

	// This is the main application loop.
	for {
		select {
		case <-ticker.C:
			// Triggered by the ticker for periodic execution.
			a.runOnce(ctx)
		case <-ctx.Done():
			// Triggered by SIGINT/SIGTERM signals.
			// Perform any cleanup here.
			log.Info("Shutting down cpeer-edge-agent.")
			return nil // Exit the Run function, terminating the application.
		}
	}
}

// runOnce performs a single cycle of the agent's logic.
func (a *Agent) runOnce(ctx context.Context) {
	log.Info("--- Agent loop start ---")
	defer log.Info("--- Agent loop end ---\n\n")

	a.sendHeartbeat(ctx)

	// Check if an upgrade is already in progress before trying to fetch a new task.
	a.upgradeLocker.Lock()
	if a.isUpgrading {
		a.upgradeLocker.Unlock()
		log.Info("An upgrade is already in progress. Skipping task check for this cycle.")
		return
	}
	a.upgradeLocker.Unlock()

	task := a.checkForTask(ctx)
	if task == nil {
		// No task found, this is the end of a normal loop.
		return
	}

	if a.state.firmwareVersion == task.Spec.Version {
		log.Info("Agent is already at the target version.", "version", a.state.firmwareVersion)
		if task.Status.Phase != firmwarev1alpha1.UpgradePhaseSucceeded {
			a.updateTaskStatus(ctx, task.Name, firmwarev1alpha1.UpgradePhaseSucceeded, "Version already up to date.")
		}
		return
	}

	// A new upgrade is needed. We'll run it in a separate goroutine
	// so it doesn't block the main agent loop if it takes a long time.
	go a.executeUpgrade(ctx, task)
}

// --- The rest of the methods remain largely the same, with minor context handling improvements ---

type HeartbeatPayload struct {
	DeviceID               string `json:"deviceID"`
	CurrentFirmwareVersion string `json:"currentFirmwareVersion"`
}

type TaskStatusUpdatePayload struct {
	Phase   firmwarev1alpha1.UpgradePhase `json:"phase"`
	Message string                        `json:"message,omitempty"`
}

func (a *Agent) sendHeartbeat(ctx context.Context) {
	hb := HeartbeatPayload{
		DeviceID:               a.deviceID,
		CurrentFirmwareVersion: a.state.firmwareVersion,
	}

	body, _ := json.Marshal(hb)
	url := fmt.Sprintf("%s/heartbeat", a.heartbeat.url)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Error(err, "Failed to create heartbeat request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		// Check if the context was canceled (e.g., during shutdown)
		if ctx.Err() != nil {
			return
		}
		log.Error(err, "Heartbeat failed")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Warn("Heartbeat returned non-2xx status", "status", resp.Status)
	} else {
		log.Info("Heartbeat sent successfully", "version", a.state.firmwareVersion)
	}
}

func (a *Agent) checkForTask(ctx context.Context) *firmwarev1alpha1.FirmwareUpgrade {
	// ... This function remains the same as your version ...
	url := fmt.Sprintf("%s/tasks/%s", a.heartbeat.url, a.deviceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Error(err, "Failed to create task check request")
		return nil
	}

	resp, err := a.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		log.Error(err, "Failed to check for tasks")
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		log.Info("No pending tasks found.")
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		log.Warn("Check for tasks returned non-200 status", "status", resp.Status)
		return nil
	}

	var task firmwarev1alpha1.FirmwareUpgrade
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		log.Error(err, "Failed to decode task")
		return nil
	}

	log.Info("Received task to upgrade", "taskName", task.Name, "targetVersion", task.Spec.Version)
	return &task
}

func (a *Agent) executeUpgrade(ctx context.Context, task *firmwarev1alpha1.FirmwareUpgrade) {
	// Set the agent's state to "upgrading" to prevent new tasks from starting.
	a.upgradeLocker.Lock()
	if a.isUpgrading {
		a.upgradeLocker.Unlock()
		log.Warn("executeUpgrade called while another upgrade was in progress. Aborting redundant call.")
		return
	}
	a.isUpgrading = true
	a.upgradeLocker.Unlock()

	// Use defer to ensure the state is unlocked no matter how the function exits (success, panic, or error).
	defer func() {
		a.upgradeLocker.Lock()
		a.isUpgrading = false
		a.upgradeLocker.Unlock()
		log.Info("Upgrade process finished. Agent is ready for new tasks.")
	}()

	log.Info("Starting firmware upgrade process", "taskName", task.Name)

	// Simulate a long-running download, longer than the heartbeat interval
	a.updateTaskStatus(ctx, task.Name, firmwarev1alpha1.UpgradePhaseUpgrading, "Download started")
	const upgradeDuration = 20 * time.Second
	log.Info("SIMULATE: Downloading firmware, this will take some time...", "duration", upgradeDuration)
	// Use a timer that respects the context cancellation
	select {
	case <-time.After(upgradeDuration):
		// Download finished
	case <-ctx.Done():
		log.Warn("Upgrade cancelled because agent is shutting down.")
		a.updateTaskStatus(ctx, task.Name, firmwarev1alpha1.UpgradePhaseFailed, "Upgrade cancelled by agent shutdown.")
		return
	}

	if err := writeVersionToFile(a.versionFile, task.Spec.Version); err != nil {
		log.Error(err, "Failed to write new version file")
		a.updateTaskStatus(ctx, task.Name, firmwarev1alpha1.UpgradePhaseFailed, err.Error())
		return
	}

	a.state.firmwareVersion = task.Spec.Version
	a.updateTaskStatus(ctx, task.Name, firmwarev1alpha1.UpgradePhaseSucceeded, "Upgrade complete")
	log.Info("Upgrade successful", "newVersion", a.state.firmwareVersion)
}

func (a *Agent) updateTaskStatus(ctx context.Context, taskName string, phase firmwarev1alpha1.UpgradePhase, message string) {
	// ... This function remains the same as your version ...
	payload := TaskStatusUpdatePayload{Phase: phase, Message: message}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/tasks/%s/status", a.heartbeat.url, taskName)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Error(err, "Failed to create task status update request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		log.Error(err, "Failed to update task status", "taskName", taskName)
		return
	}
	defer resp.Body.Close()
	log.Info("Updated task status successfully", "taskName", taskName, "phase", phase)
}
