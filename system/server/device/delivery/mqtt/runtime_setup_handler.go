package mqtthandler

import (
	"log/slog"

	"go.autonomous.ai/os/system/device"
	"go.autonomous.ai/os/system/domain"
)

// handleRuntimeSetup applies a `hermes.setup` / `picoclaw.setup` downlink — swap
// the active agentic backend. The kind itself names the target runtime (passed
// in by the dispatcher), so unlike the former generic agent_runtime.set there is
// no runtime field to read off the wire.
//
// Flow: ack "starting" immediately, then in a goroutine run the switch and WAIT
// for its real outcome. The reserved runner requires the target runtime's
// readiness probe to pass after its systemd unit is active, so the
// success/failure ack here reflects a usable gateway — not an optimistic
// process-active guess. On a confirmed switch we ack "success" and THEN restart
// os-server (the ack must reach the wire first, since the restart kills us); the
// worker sees the swap land via the brief reconnect + new AGENT BACKEND ACTIVE
// banner. On failure switch-runtime has already rolled back, so we ack "failure".
// Every ack echoes the triggering kind so the worker can match hermes.setup vs
// picoclaw.setup.

func (h *DeviceMQTTHandler) publishRuntimeSetupAck(kind, status, errMsg string, data *domain.AgentRuntimeSetData) {
	ack := domain.AgentRuntimeSetAck{
		MQTTInfoResponse: domain.NewMQTTInfoResponse(h.config, "data", device.GetDeviceMac()),
		Kind:             kind,
		Status:           status,
		Error:            errMsg,
		Data:             data,
	}
	if err := h.publish(ack); err != nil {
		slog.Warn("runtime setup: publish ack failed", "component", "mqtt", "kind", kind, "status", status, "error", err)
	}
}

// handleRuntimeSetup is shared by the hermes.setup and picoclaw.setup dispatch
// cases; runtime is the target backend named by the kind.
func (h *DeviceMQTTHandler) handleRuntimeSetup(env domain.MQTTDataCommand, runtime string) error {
	kind := env.Kind
	req := domain.AgentRuntimeSetData{Runtime: runtime}

	slog.Info("runtime setup: received", "component", "mqtt", "kind", kind, "runtime", runtime)

	// Keep MQTT on the same readiness-confirmed switch contract as HTTP. A
	// systemd-active target that has not yet bound its gateway or accepted its
	// protocol must fail and roll back rather than receive a success ack.
	run, err := h.deviceService.ReserveAgentRuntimeSwitchReady(req)
	if err != nil {
		slog.Warn("runtime setup: switch already in progress", "component", "mqtt", "kind", kind, "runtime", runtime)
		h.publishRuntimeSetupAck(kind, "failure", err.Error(), &req)
		h.alertOps("🔴 Runtime setup "+kind+" — FAILED", err.Error())
		return nil
	}

	// Ack immediately so the worker knows the device received the command.
	h.publishRuntimeSetupAck(kind, "starting", "", nil)
	h.alertOps("🚀 Runtime setup "+kind+" — starting", "")

	go func() {
		switched, err := run()
		if err != nil {
			// switch-runtime already rolled back; report the real failure.
			slog.Error("runtime setup: switch failed", "component", "mqtt", "kind", kind, "error", err)
			h.publishRuntimeSetupAck(kind, "failure", err.Error(), &req)
			h.alertOps("🔴 Runtime setup "+kind+" — FAILED", err.Error())
			return
		}
		// Switch confirmed readiness (or was a no-op). Ack success — it must reach
		// the wire BEFORE the os-server restart below, which kills us.
		slog.Info("runtime setup: switch confirmed", "component", "mqtt", "kind", kind, "runtime", runtime, "switched", switched)
		h.publishRuntimeSetupAck(kind, "success", "", &req)
		h.alertOps("🟢 Runtime setup "+kind+" — SUCCESS", "")

		if switched {
			// Restart os-server so factory.go re-resolves the gateway to the new
			// backend. Deferred until after the ack on purpose.
			if rerr := h.deviceService.RestartForAgentRuntime(); rerr != nil {
				slog.Error("runtime setup: os-server restart failed", "component", "mqtt", "kind", kind, "error", rerr)
			}
		}
	}()

	return nil
}
