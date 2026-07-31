package mqtthandler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/server/config"
)

// newTestStream builds a stream whose transport captures payloads instead of
// reaching a broker.
func newTestStream() (*ChatStream, func() []domain.MQTTChatEventData) {
	var mu sync.Mutex
	var sent []domain.MQTTChatEventData

	s := &ChatStream{
		cfg:  &config.Config{DeviceID: "dev1", FDChannel: "fd/dev1"},
		runs: map[string]*trackedRun{},
	}
	s.publish = func(body []byte) error {
		var resp struct {
			Kind string                   `json:"kind"`
			Data domain.MQTTChatEventData `json:"data"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		sent = append(sent, resp.Data)
		return nil
	}

	return s, func() []domain.MQTTChatEventData {
		mu.Lock()
		defer mu.Unlock()
		out := make([]domain.MQTTChatEventData, len(sent))
		copy(out, sent)
		return out
	}
}

// The bus carries every turn on the device, including voice ones. Only runs the
// backend started may be mirrored — otherwise a spoken conversation in the room
// would stream to whoever last opened the app.
func TestChatStreamIgnoresUntrackedRuns(t *testing.T) {
	s, sent := newTestStream()

	s.handle(domain.MonitorEvent{Type: "chat_response", RunID: "not-mine", Summary: "hi"})
	s.handle(domain.MonitorEvent{Type: "assistant_delta", RunID: "", Summary: "orphan"})
	s.flushAll()

	if got := sent(); len(got) != 0 {
		t.Fatalf("published %d events for untracked runs: %+v", len(got), got)
	}
}

// Deltas are accumulated, not forwarded 1:1 — at QoS 1 each publish is a
// round-trip, and the bus emits one delta per model chunk.
func TestChatStreamCoalescesDeltas(t *testing.T) {
	s, sent := newTestStream()
	s.Track("run1", "sess1")

	for _, chunk := range []string{"Hello", " there", ", Leo"} {
		s.handle(domain.MonitorEvent{Type: "assistant_delta", RunID: "run1", Summary: chunk})
	}
	if got := sent(); len(got) != 0 {
		t.Fatalf("deltas published before a flush: %+v", got)
	}

	s.flushAll()

	got := sent()
	if len(got) != 1 {
		t.Fatalf("want 1 coalesced event, got %d: %+v", len(got), got)
	}
	if got[0].Event.Summary != "Hello there, Leo" {
		t.Errorf("text = %q, want the three chunks joined", got[0].Event.Summary)
	}
	if got[0].SessionID != "sess1" {
		t.Errorf("session = %q, want it echoed on every event", got[0].SessionID)
	}
	if got[0].RunID != "run1" {
		t.Errorf("run id = %q", got[0].RunID)
	}
	// A coalesced event is not any one of the chunks it was built from, so it
	// must not reuse their id.
	if got[0].Event.ID != "" {
		t.Errorf("coalesced event kept id %q", got[0].Event.ID)
	}
}

// A tool chip must not overtake the sentence that preceded it: pending delta
// text is flushed before any other event goes out.
func TestChatStreamFlushesPendingBeforeOtherEvents(t *testing.T) {
	s, sent := newTestStream()
	s.Track("run1", "sess1")

	s.handle(domain.MonitorEvent{Type: "assistant_delta", RunID: "run1", Summary: "Taking a photo"})
	s.handle(domain.MonitorEvent{Type: "tool_call", RunID: "run1", Summary: "Tool Bash"})

	got := sent()
	if len(got) != 2 {
		t.Fatalf("want 2 events, got %d: %+v", len(got), got)
	}
	if got[0].Event.Type != "assistant_delta" || got[0].Event.Summary != "Taking a photo" {
		t.Errorf("first event = %+v, want the buffered text", got[0].Event)
	}
	if got[1].Event.Type != "tool_call" {
		t.Errorf("second event = %+v, want the tool call", got[1].Event)
	}
}

// chat_response ends the turn: it is published, then the run stops being
// mirrored so late bus noise doesn't leak onto the wire.
func TestChatStreamStopsAtTerminalEvent(t *testing.T) {
	s, sent := newTestStream()
	s.Track("run1", "sess1")

	s.handle(domain.MonitorEvent{Type: "chat_response", RunID: "run1", Summary: "done"})
	s.handle(domain.MonitorEvent{Type: "assistant_delta", RunID: "run1", Summary: "late"})
	s.flushAll()

	got := sent()
	if len(got) != 1 {
		t.Fatalf("want only the terminal event, got %d: %+v", len(got), got)
	}
	if got[0].Event.Type != "chat_response" {
		t.Errorf("event = %+v", got[0].Event)
	}

	s.mu.Lock()
	_, still := s.runs["run1"]
	s.mu.Unlock()
	if still {
		t.Error("run still tracked after its terminal event")
	}
}

// A turn that dies without a terminal event must not be tracked forever.
func TestChatStreamSweepsExpiredRuns(t *testing.T) {
	s, _ := newTestStream()
	s.Track("stale", "sess1")
	s.Track("fresh", "sess2")

	s.mu.Lock()
	s.runs["stale"].started = time.Now().Add(-2 * runTTL)
	s.mu.Unlock()

	s.sweep()

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runs["stale"]; ok {
		t.Error("expired run survived the sweep")
	}
	if _, ok := s.runs["fresh"]; !ok {
		t.Error("sweep removed a live run")
	}
}

// The bus puts delta text on Summary; the flow_event shape nests it under
// detail.text. Both must read.
func TestDeltaText(t *testing.T) {
	cases := []struct {
		name string
		evt  domain.MonitorEvent
		want string
	}{
		{"summary", domain.MonitorEvent{Summary: "hi"}, "hi"},
		{"detail", domain.MonitorEvent{Detail: map[string]any{"text": "hi"}}, "hi"},
		{"summary wins", domain.MonitorEvent{Summary: "a", Detail: map[string]any{"text": "b"}}, "a"},
		{"neither", domain.MonitorEvent{Detail: map[string]any{"other": 1}}, ""},
	}
	for _, tc := range cases {
		if got := deltaText(tc.evt); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// captureFiles is the chat.file half of newTestStream's capture.
func filesFrom(t *testing.T, s *ChatStream) func() []domain.MQTTChatFileData {
	t.Helper()
	var mu sync.Mutex
	var files []domain.MQTTChatFileData
	prev := s.publish
	s.publish = func(body []byte) error {
		var resp struct {
			Kind string          `json:"kind"`
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return err
		}
		if resp.Kind == domain.KindChatFile {
			var f domain.MQTTChatFileData
			if err := json.Unmarshal(resp.Data, &f); err != nil {
				return err
			}
			mu.Lock()
			files = append(files, f)
			mu.Unlock()
			return nil
		}
		return prev(body)
	}
	return func() []domain.MQTTChatFileData {
		mu.Lock()
		defer mu.Unlock()
		out := make([]domain.MQTTChatFileData, len(files))
		copy(out, files)
		return out
	}
}

// seedTmpFile writes a file under a temp dir and points the stream's roots at it
// by returning the path; agentfile.Roots() includes /tmp, so tests use that.
func seedTmpFile(t *testing.T, name string, size int) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "chatfile-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, make([]byte, size), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// The path an agent sends is usually in a tool's ARGS, not in what it says —
// this is the case that made text-only detection useless on the device.
func TestChatStreamRelaysFileFromToolArgs(t *testing.T) {
	s, _ := newTestStream()
	files := filesFrom(t, s)
	s.Track("run1", "sess1")

	img := seedTmpFile(t, "snap.jpg", 64)
	s.handle(domain.MonitorEvent{
		Type:    "tool_call",
		RunID:   "run1",
		Summary: "Tool message",
		Detail:  map[string]any{"args": `{"action":"send","media":"` + img + `"}`},
	})

	got := files()
	if len(got) != 1 {
		t.Fatalf("want 1 file, got %d: %+v", len(got), got)
	}
	if got[0].Name != "snap.jpg" || got[0].MIME != "image/jpeg" || got[0].Size != 64 {
		t.Errorf("metadata = %+v", got[0])
	}
	if got[0].Content == "" || got[0].TooLarge {
		t.Errorf("want inline bytes, got content=%d too_large=%v", len(got[0].Content), got[0].TooLarge)
	}
	if got[0].RunID != "run1" || got[0].SessionID != "sess1" {
		t.Errorf("correlation = %+v", got[0])
	}
}

// The same snapshot shows up in a tool's args, its result AND the reply. Sending
// its bytes three times is the most expensive mistake available here.
func TestChatStreamSendsEachFileOnce(t *testing.T) {
	s, _ := newTestStream()
	files := filesFrom(t, s)
	s.Track("run1", "sess1")

	img := seedTmpFile(t, "snap.jpg", 32)
	s.handle(domain.MonitorEvent{Type: "tool_call", RunID: "run1",
		Detail: map[string]any{"args": `{"media":"` + img + `"}`}})
	s.handle(domain.MonitorEvent{Type: "tool_call", RunID: "run1",
		Summary: "Tool Bash done: {\"path\": \"" + img + "\"}"})
	s.handle(domain.MonitorEvent{Type: "chat_response", RunID: "run1", Summary: "đây nè: " + img})

	if got := files(); len(got) != 1 {
		t.Fatalf("want the file once, got %d", len(got))
	}
}

// Past the inline budget the metadata still goes out — a client should be able
// to say "a big file was produced" rather than show nothing.
func TestChatStreamMarksOversizedFile(t *testing.T) {
	s, _ := newTestStream()
	files := filesFrom(t, s)
	s.Track("run1", "sess1")

	big := seedTmpFile(t, "clip.mp4", chatFileMaxInlineBytes+1)
	s.handle(domain.MonitorEvent{Type: "chat_response", RunID: "run1", Summary: "made " + big})

	got := files()
	if len(got) != 1 {
		t.Fatalf("want 1 file record, got %d", len(got))
	}
	if !got[0].TooLarge {
		t.Error("want too_large set")
	}
	if got[0].Content != "" {
		t.Error("oversized file must not carry bytes")
	}
	if got[0].Size <= chatFileMaxInlineBytes {
		t.Errorf("size = %d, want the real size reported anyway", got[0].Size)
	}
}

// A path the allow-list refuses, or one that isn't there, is silently skipped —
// an agent mentioning a config file is not an error worth publishing.
func TestChatStreamSkipsRefusedPaths(t *testing.T) {
	s, _ := newTestStream()
	files := filesFrom(t, s)
	s.Track("run1", "sess1")

	secret := seedTmpFile(t, "creds.json", 16) // served root, unserved type
	s.handle(domain.MonitorEvent{Type: "chat_response", RunID: "run1",
		Summary: "see " + secret + " and /root/.openclaw/openclaw.json and /tmp/gone-9f3a.png"})

	if got := files(); len(got) != 0 {
		t.Fatalf("published %d files that should have been refused: %+v", len(got), got)
	}
}

// A file named by a turn nobody asked to mirror must not leave the device.
func TestChatStreamNoFilesForUntrackedRun(t *testing.T) {
	s, _ := newTestStream()
	files := filesFrom(t, s)

	img := seedTmpFile(t, "snap.jpg", 16)
	s.handle(domain.MonitorEvent{Type: "chat_response", RunID: "someone-elses", Summary: img})

	if got := files(); len(got) != 0 {
		t.Fatalf("leaked %d files from an untracked run", len(got))
	}
}
