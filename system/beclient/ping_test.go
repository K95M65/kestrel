package beclient

import (
	"encoding/json"
	"testing"
)

func TestPingPayloadIncludesWakeWordState(t *testing.T) {
	for _, want := range []bool{true, false} {
		payload, err := json.Marshal(PingPayload{WakeWordEnabled: want})
		if err != nil {
			t.Fatalf("marshal ping payload: %v", err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(payload, &fields); err != nil {
			t.Fatalf("unmarshal ping payload: %v", err)
		}

		raw, ok := fields["wakeword_enabled"]
		if !ok {
			t.Fatal("wakeword_enabled must be present even when disabled")
		}
		var got bool
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("unmarshal wakeword_enabled: %v", err)
		}
		if got != want {
			t.Fatalf("wakeword_enabled = %t, want %t", got, want)
		}
	}
}
