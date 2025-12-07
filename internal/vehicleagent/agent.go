package vehicleagent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pb "cloupeer.io/cloupeer/api/proto/v1"
	"cloupeer.io/cloupeer/internal/vehicleagent/core"
	"cloupeer.io/cloupeer/internal/vehicleagent/hub"
	"cloupeer.io/cloupeer/pkg/log"
)

type Agent struct {
	hal core.HAL
	hub *hub.Hub

	modules []core.Module
}

func NewAgent(hal core.HAL, hub *hub.Hub, modules ...core.Module) *Agent {
	return &Agent{
		hal:     hal,
		hub:     hub,
		modules: modules,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	vid := a.hal.GetVehicleID()
	log.Info("Starting cpeer-edge-agent", "vehicleID", vid, "version", a.hal.GetFirmwareVersion())

	for _, m := range a.modules {
		if err := m.Setup(ctx, a.hal, a.hub); err != nil {
			return err
		}

		for event, handler := range m.Routes() {
			if err := a.hub.Register(event, handler); err != nil {
				return fmt.Errorf("module %s register event %s failed: %w", m.Name(), event, err)
			}
		}
	}

	if err := a.hub.Start(ctx); err != nil {
		return err
	}
	defer a.hub.Stop()

	if err := a.reportStatus(ctx, true); err != nil {
		log.Error(err, "Failed to publish online status")
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = a.reportStatus(shutdownCtx, false, "GracefulShutdown")
	}()

	go a.confirmSystemHealth(ctx)
	go a.registerIdentity(ctx)

	<-ctx.Done()
	log.Info("Agent shutting down...")

	return nil
}

func (a *Agent) reportStatus(ctx context.Context, online bool, reason ...string) error {
	payload := &pb.OnlineStatus{
		VehicleId: a.hal.GetVehicleID(),
		Online:    online,
	}

	if len(reason) > 0 {
		payload.Reason = reason[0]
	}

	data, _ := json.Marshal(payload)
	return a.hub.Send(ctx, core.EventOnline, data)
}

func (a *Agent) confirmSystemHealth(ctx context.Context) {
	// 策略：让系统先跑 10 秒。
	// 如果这 10 秒内 Agent 没有 Crash，且 MQTT 连接保持正常，我们才认为“启动成功”。
	select {
	case <-ctx.Done():
		return // 如果 10秒内系统就要关闭了，那就不标记了
	case <-time.After(10 * time.Second):
		if !a.hub.IsConnected() {
			log.Warn("System running but MQTT not connected. Skipping Boot Success Mark.")
			return
		}

		// --- 调用 HAL 标记成功 ---
		log.Info("System stabilized. Marking boot as successful (Committing Slot B).")
		if err := a.hal.MarkBootSuccessful(); err != nil {
			// 仅仅记录错误，不退出程序，保业务
			log.Error(err, "CRITICAL: Failed to write Boot Success flag to Bootloader. System might rollback on next reboot.")
		} else {
			log.Info("Boot successful marked. Rollback disabled.")
		}
	}
}

// registerIdentity sends the initial registration packet to the Hub.
func (a *Agent) registerIdentity(ctx context.Context) {
	req := &pb.RegisterVehicleRequest{
		VehicleId:       a.hal.GetVehicleID(),
		FirmwareVersion: a.hal.GetFirmwareVersion(),
		Description:     "Vehicle Agent Auto-Registration",
		Timestamp:       time.Now().Unix(),
	}

	// Retry logic could be added here, but for now we send once (QoS 1 handles delivery)
	if err := a.hub.SendProto(ctx, core.EventRegister, req); err != nil {
		log.Error(err, "Failed to send registration request")
		return
	}

	log.Info("Agent registered successfully", "version", req.FirmwareVersion)
}
