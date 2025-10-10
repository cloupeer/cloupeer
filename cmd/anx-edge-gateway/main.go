// Copyright 2025 Anankix.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	edgev1alpha1 "github.com/anankix/anankix/pkg/apis/edge/v1alpha1"
)

// HeartbeatRequest defines the payload for the heartbeat endpoint.
type HeartbeatRequest struct {
	DeviceID               string `json:"deviceId"`
	CurrentFirmwareVersion string `json:"currentFirmwareVersion"`
}

// TaskStatusUpdateRequest defines the payload for updating a task's status.
type TaskStatusUpdateRequest struct {
	Phase   edgev1alpha1.TaskPhase `json:"phase"`
	Message string                 `json:"message,omitempty"`
}

// Gateway holds the K8s client and other gateway-wide dependencies.
type Gateway struct {
	K8sClient ctrlclient.Client
	Namespace string
}

func getRemoteIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// handleHeartbeat processes device heartbeats.
func (g *Gateway) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var hb HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
		http.Error(w, fmt.Sprintf("invalid body: %v", err), http.StatusBadRequest)
		return
	}
	if hb.DeviceID == "" {
		http.Error(w, "deviceId is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	name := hb.DeviceID
	now := metav1.NewTime(time.Now().UTC())
	ip := getRemoteIP(r)

	var pd edgev1alpha1.PhysicalDevice
	if err := g.K8sClient.Get(ctx, types.NamespacedName{Namespace: g.Namespace, Name: name}, &pd); err != nil {
		if apierrors.IsNotFound(err) {
			pd = edgev1alpha1.PhysicalDevice{
				ObjectMeta: metav1.ObjectMeta{Namespace: g.Namespace, Name: name},
				Spec:       edgev1alpha1.PhysicalDeviceSpec{DeviceID: hb.DeviceID},
			}
			if err := g.K8sClient.Create(ctx, &pd); err != nil {
				log.Printf("ERROR: failed to create PhysicalDevice: %v", err)
				http.Error(w, "failed to create PhysicalDevice", http.StatusInternalServerError)
				return
			}
			// refetch to get the full object
			_ = g.K8sClient.Get(ctx, types.NamespacedName{Namespace: g.Namespace, Name: name}, &pd)
		} else {
			log.Printf("ERROR: failed to get PhysicalDevice: %v", err)
			http.Error(w, "failed to get PhysicalDevice", http.StatusInternalServerError)
			return
		}
	}

	// Update status
	patch := ctrlclient.MergeFrom(pd.DeepCopy())
	pd.Status.LastHeartbeatTime = now
	pd.Status.IPAddress = ip
	pd.Status.CurrentFirmwareVersion = hb.CurrentFirmwareVersion
	if err := g.K8sClient.Status().Patch(ctx, &pd, patch); err != nil {
		log.Printf("ERROR: failed to update status: %v", err)
		http.Error(w, "failed to update status", http.StatusInternalServerError)
		return
	}

	log.Printf("Heartbeat received: deviceId=%s, version=%s, ip=%s", hb.DeviceID, hb.CurrentFirmwareVersion, ip)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleGetTask retrieves the first pending task for a device.
func (g *Gateway) handleGetTask(w http.ResponseWriter, r *http.Request) {
	deviceID := strings.TrimPrefix(r.URL.Path, "/tasks/")
	if deviceID == "" {
		http.Error(w, "Device ID is required in the URL path", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	var taskList edgev1alpha1.FirmwareUpgradeTaskList
	if err := g.K8sClient.List(ctx, &taskList,
		ctrlclient.InNamespace(g.Namespace),
		ctrlclient.MatchingLabels{"edge.anankix/deviceId": deviceID},
	); err != nil {
		log.Printf("ERROR: failed to list tasks for device %s: %v", deviceID, err)
		http.Error(w, "Failed to list tasks", http.StatusInternalServerError)
		return
	}

	// Find the first pending task
	for _, task := range taskList.Items {
		if task.Status.Phase == "" || task.Status.Phase == edgev1alpha1.TaskPhasePending {
			log.Printf("Task %s found for device %s", task.Name, deviceID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(task)
			return
		}
	}

	log.Printf("No pending tasks found for device %s", deviceID)
	w.WriteHeader(http.StatusNoContent)
}

// handleUpdateTaskStatus updates the status of a specific task.
func (g *Gateway) handleUpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	// Path should be /tasks/{taskName}/status
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "tasks" || parts[2] != "status" {
		http.Error(w, "Invalid path format. Use /tasks/{taskName}/status", http.StatusBadRequest)
		return
	}
	taskName := parts[1]

	var req TaskStatusUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid body: %v", err), http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	var task edgev1alpha1.FirmwareUpgradeTask
	if err := g.K8sClient.Get(ctx, types.NamespacedName{Namespace: g.Namespace, Name: taskName}, &task); err != nil {
		log.Printf("ERROR: failed to get task %s: %v", taskName, err)
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	patch := ctrlclient.MergeFrom(task.DeepCopy())
	task.Status.Phase = req.Phase
	task.Status.Message = req.Message
	now := metav1.NewTime(time.Now().UTC())
	if task.Status.StartTime == nil && req.Phase != edgev1alpha1.TaskPhasePending {
		task.Status.StartTime = &now
	}
	if req.Phase == edgev1alpha1.TaskPhaseSucceeded || req.Phase == edgev1alpha1.TaskPhaseFailed {
		task.Status.CompletionTime = &now
	}

	if err := g.K8sClient.Status().Patch(ctx, &task, patch); err != nil {
		log.Printf("ERROR: failed to update task status for %s: %v", taskName, err)
		http.Error(w, "Failed to update task status", http.StatusInternalServerError)
		return
	}

	log.Printf("Task %s status updated to %s", taskName, req.Phase)
	w.WriteHeader(http.StatusOK)
}

func main() {
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatalf("FATAL: failed to get k8s config: %v", err)
	}

	scheme := runtime.NewScheme()
	// IMPORTANT: Register both types with the scheme
	if err := edgev1alpha1.AddToScheme(scheme); err != nil {
		log.Fatalf("FATAL: failed to add scheme: %v", err)
	}

	k8sClient, err := ctrlclient.New(cfg, ctrlclient.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("FATAL: failed to create client: %v", err)
	}

	gateway := &Gateway{
		K8sClient: k8sClient,
		Namespace: namespace,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/heartbeat", gateway.handleHeartbeat)
	mux.HandleFunc("/tasks/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/status") {
			gateway.handleUpdateTaskStatus(w, r)
		} else {
			gateway.handleGetTask(w, r)
		}
	})

	addr := ":9090"
	if v := os.Getenv("ADDR"); v != "" {
		addr = v
	}
	log.Printf("anx-edge-gateway listening on %s (namespace=%s)", addr, namespace)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("FATAL: server error: %v", err)
	}
}
