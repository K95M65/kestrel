package http

import (
	"errors"
	"log/slog"

	"go.autonomous.ai/os/lib/flow"
	"go.autonomous.ai/os/lib/hal"
)

// deliverTTS sends text to HAL in a background goroutine and logs the outcome:
// hal.ErrSpeakerMuted becomes a tts_muted flow event so the Flow Monitor can
// show that the reply was displayed but never spoken; any other error is a
// real delivery failure.
func (h *AgentHandler) deliverTTS(send func(string) error, text, flowRunID, errCtx string) {
	go func() {
		err := send(text)
		if err == nil {
			return
		}
		if errors.Is(err, hal.ErrSpeakerMuted) {
			slog.Info("TTS muted -- speaker muted on device", "component", "agent", "run_id", flowRunID, "text", text[:min(len(text), 80)])
			flow.Log("tts_muted", map[string]any{"run_id": flowRunID, "text": text}, flowRunID)
			return
		}
		slog.Error(errCtx, "component", "agent", "error", err)
	}()
}
