package hub

import (
	"encoding/json"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	firmwarev1alpha1 "cloupeer.io/cloupeer/pkg/apis/firmware/v1alpha1"
	iotv1alpha1 "cloupeer.io/cloupeer/pkg/apis/iot/v1alpha1"
	"cloupeer.io/cloupeer/pkg/log"
)

// HeartbeatRequest defines the payload from the agent's heartbeat.
type HeartbeatRequest struct {
	DeviceID               string `json:"deviceID"`
	CurrentFirmwareVersion string `json:"currentFirmwareVersion"`
}

// TaskStatusUpdateRequest defines the payload for updating a task's status.
type TaskStatusUpdateRequest struct {
	Phase   firmwarev1alpha1.UpgradePhase `json:"phase"`
	Message string                        `json:"message,omitempty"`
}

func (s *HubServer) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	log.Info("Received heartbeat", "deviceID", req.DeviceID, "firmwareVersion", req.CurrentFirmwareVersion)

	// Use DeviceID as the resource name
	deviceName := req.DeviceID
	var device iotv1alpha1.Device
	err := s.k8sclient.Get(ctx, types.NamespacedName{Name: deviceName, Namespace: s.namespace}, &device)
	if err != nil {
		if errors.IsNotFound(err) {
			// Device not found, create it (auto-registration)
			log.Info("Device not found, creating new Device resource.", "deviceID", deviceName)
			newDevice := &iotv1alpha1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      deviceName,
					Namespace: s.namespace,
					Labels:    map[string]string{"iot.cloupeer.io/device-name": deviceName},
				},
				Spec: iotv1alpha1.DeviceSpec{
					// --- 修改点 1: Removed DeviceID from Spec ---
					// DeviceModelRef and Protocol likely need defaulting or later configuration
				},
				// Initial Status can be set here if desired
				Status: iotv1alpha1.DeviceStatus{
					Phase: iotv1alpha1.DevicePhaseUnknown, // Start as Unknown until first proper status update
				},
			}
			if err := s.k8sclient.Create(ctx, newDevice); err != nil {
				log.Error(err, "Failed to create Device")
				http.Error(w, "Failed to register device", http.StatusInternalServerError)
				return
			}
			// Fallthrough to update status on the newly created device
			device = *newDevice
		} else {
			log.Error(err, "Failed to get Device")
			http.Error(w, "Failed to get device info", http.StatusInternalServerError)
			return
		}
	}

	// Patch the status to avoid conflicts
	patch := client.MergeFrom(device.DeepCopy())
	// Update LastSeenTime and Phase
	now := metav1.Now()
	device.Status.LastOnlineTime = &now
	device.Status.Phase = iotv1alpha1.DevicePhaseOnline // Mark as online on heartbeat

	// Update or Add the firmwareVersion Twin
	foundFirmwareTwin := false
	for i := range device.Status.Twins {
		if device.Status.Twins[i].PropertyName == "firmwareVersion" {
			// Update existing twin's reported value and metadata
			device.Status.Twins[i].Reported.Value = req.CurrentFirmwareVersion
			if device.Status.Twins[i].Reported.Metadata == nil {
				device.Status.Twins[i].Reported.Metadata = make(map[string]string)
			}
			device.Status.Twins[i].Reported.Metadata["timestamp"] = now.UTC().Format(time.RFC3339Nano)
			foundFirmwareTwin = true
			break
		}
	}

	// If firmwareVersion twin doesn't exist, add it
	if !foundFirmwareTwin {
		device.Status.Twins = append(device.Status.Twins, iotv1alpha1.Twin{
			PropertyName: "firmwareVersion",
			Reported: iotv1alpha1.TwinProperty{
				Value: req.CurrentFirmwareVersion,
				Metadata: map[string]string{
					"timestamp": now.UTC().Format(time.RFC3339Nano),
				},
			},
			// ObservedDesired can be initialized or updated here if applicable
		})
	}

	// Add/Update Ready Condition
	readyCondition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: device.Generation, // Reflects the generation observed for this status
		LastTransitionTime: metav1.Now(),
		Reason:             "DeviceOnline",
		Message:            "Device reported heartbeat successfully.",
	}
	meta.SetStatusCondition(&device.Status.Conditions, readyCondition)

	if err := s.k8sclient.Status().Patch(ctx, &device, patch); err != nil {
		log.Error(err, "Failed to patch Device status")
		http.Error(w, "Failed to update device status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *HubServer) handleGetTask(w http.ResponseWriter, r *http.Request, deviceID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	var upgradeList firmwarev1alpha1.FirmwareUpgradeList
	// Find tasks that target this device via labels
	err := s.k8sclient.List(ctx, &upgradeList,
		client.InNamespace(s.namespace),
		client.MatchingLabels{"iot.cloupeer.io/device-name": deviceID},
	)
	if err != nil {
		log.Error(err, "Failed to list FirmwareUpgrades for device", "deviceID", deviceID)
		http.Error(w, "Failed to query tasks", http.StatusInternalServerError)
		return
	}

	// Find the first pending task
	for _, task := range upgradeList.Items {
		// We only distribute tasks that are explicitly in the Pending state.
		if task.Status.Phase == firmwarev1alpha1.UpgradePhasePending {
			log.Info("Found pending task for device, attempting to claim it.", "deviceID", deviceID, "taskName", task.Name)

			// Claim the task by patching its status to Upgrading ---
			patch := client.MergeFrom(task.DeepCopy())
			task.Status.Phase = firmwarev1alpha1.UpgradePhaseUpgrading
			if err := s.k8sclient.Status().Patch(ctx, &task, patch); err != nil {
				// If we fail to claim it (e.g., due to a conflict), we can't give it to the agent.
				log.Error(err, "Failed to claim task by patching status to Upgrading", "taskName", task.Name)
				// We can either try the next task or just fail this request. Let's fail for now.
				http.Error(w, "Failed to claim a task, please try again", http.StatusInternalServerError)
				return
			}

			log.Info("Task successfully claimed. Distributing to agent.", "taskName", task.Name)
			w.Header().Set("Content-Type", "application/json")
			// Encode the *updated* task object so the agent knows it's now Upgrading.
			json.NewEncoder(w).Encode(task)
			return
		}
	}

	// No pending task found
	w.WriteHeader(http.StatusNoContent)
}

func (s *HubServer) handleUpdateTaskStatus(w http.ResponseWriter, r *http.Request, taskName string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	var req TaskStatusUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var task firmwarev1alpha1.FirmwareUpgrade
	if err := s.k8sclient.Get(ctx, types.NamespacedName{Name: taskName, Namespace: s.namespace}, &task); err != nil {
		log.Error(err, "Failed to get FirmwareUpgrade task to update status", "taskName", taskName)
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	patch := client.MergeFrom(task.DeepCopy())
	task.Status.Phase = req.Phase
	// You might want to add a message field to your FirmwareUpgradeStatus struct
	// task.Status.Message = req.Message

	if err := s.k8sclient.Status().Patch(ctx, &task, patch); err != nil {
		log.Error(err, "Failed to patch FirmwareUpgrade status", "taskName", taskName)
		http.Error(w, "Failed to update task status", http.StatusInternalServerError)
		return
	}

	log.Info("Task status updated by agent", "taskName", taskName, "newPhase", req.Phase)
	w.WriteHeader(http.StatusOK)
}
