package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type session struct {
	mu      sync.Mutex
	store   *fileStore
	rec     *pairingRecord
	conn    *websocket.Conn
	paused  bool
	status  string
	lastErr string
	lastCmd string
	stop    chan struct{}
	looping bool
}

func newSession(store *fileStore) *session {
	s := &session{store: store, status: "idle", stop: make(chan struct{})}
	if rec, err := store.load(); err == nil && rec != nil {
		s.rec = rec
		s.status = "disconnected"
	}
	return s
}

func (s *session) snapshot() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]any{
		"status":  s.status,
		"paused":  s.paused,
		"error":   s.lastErr,
		"last":    s.lastCmd,
		"os":      currentOS(),
		"paired":  s.rec != nil,
	}
	if s.rec != nil {
		out["host"] = s.rec.DeviceHost
		out["buddy_id"] = s.rec.BuddyID
	}
	return out
}

func (s *session) setPaused(v bool) {
	s.mu.Lock()
	s.paused = v
	s.mu.Unlock()
}

func (s *session) pair(host, code string) error {
	rec, err := pairWithDevice(host, code, deviceName(), fingerprint(), currentOS())
	if err != nil {
		return err
	}
	if err := s.store.save(rec); err != nil {
		return err
	}
	s.mu.Lock()
	s.rec = &rec
	s.status = "disconnected"
	s.lastErr = ""
	s.mu.Unlock()
	s.start()
	return nil
}

func (s *session) unpair() {
	s.mu.Lock()
	rec := s.rec
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
	s.rec = nil
	s.status = "idle"
	s.mu.Unlock()
	if rec != nil {
		notifyRevoke(rec.DeviceHost, rec.Token)
		_ = s.store.clear()
	}
}

func (s *session) start() {
	s.mu.Lock()
	if s.rec == nil || s.looping {
		s.mu.Unlock()
		return
	}
	s.looping = true
	s.mu.Unlock()
	go s.loop()
}

func (s *session) loop() {
	defer func() {
		s.mu.Lock()
		s.looping = false
		s.mu.Unlock()
	}()
	delay := time.Second
	for {
		s.mu.Lock()
		rec := s.rec
		s.mu.Unlock()
		if rec == nil {
			return
		}
		s.mu.Lock()
		s.status = "connecting"
		s.mu.Unlock()
		err := s.dial(*rec)
		s.mu.Lock()
		if s.rec == nil {
			s.mu.Unlock()
			return
		}
		s.status = "disconnected"
		if err != nil {
			s.lastErr = err.Error()
		}
		s.mu.Unlock()
		time.Sleep(delay)
		if delay < 15*time.Second {
			delay *= 2
		}
	}
}

func (s *session) dial(rec pairingRecord) error {
	hdr := http.Header{}
	hdr.Set("Authorization", "Bearer "+rec.Token)
	c, _, err := websocket.DefaultDialer.Dial("ws://"+rec.DeviceHost+"/api/buddy/ws", hdr)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.conn = c
	s.status = "connected"
	s.lastErr = ""
	s.mu.Unlock()
	log.Printf("connected to %s", rec.DeviceHost)
	defer func() {
		s.mu.Lock()
		if s.conn == c {
			s.conn = nil
		}
		s.mu.Unlock()
		_ = c.Close()
	}()

	c.SetReadDeadline(time.Now().Add(90 * time.Second))
	c.SetPongHandler(func(string) error {
		c.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for range t.C {
			if err := c.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}()

	for {
		_, data, err := c.ReadMessage()
		if err != nil {
			return err
		}
		c.SetReadDeadline(time.Now().Add(90 * time.Second))
		resp := s.handle(data)
		if err := c.WriteMessage(websocket.TextMessage, resp); err != nil {
			return err
		}
	}
}

func (s *session) handle(data []byte) []byte {
	start := time.Now()
	var cmd struct {
		ID     string         `json:"id"`
		Action string         `json:"action"`
		Params map[string]any `json:"params"`
	}
	if err := json.Unmarshal(data, &cmd); err != nil || cmd.ID == "" {
		return mustJSON(map[string]any{"id": "unknown", "ok": false, "error": "malformed command", "duration_ms": 0})
	}
	s.mu.Lock()
	paused := s.paused
	s.lastCmd = cmd.Action
	s.mu.Unlock()
	if paused {
		return mustJSON(map[string]any{"id": cmd.ID, "ok": false, "error": "buddy paused by user", "duration_ms": 0})
	}
	out, err := dispatch(cmd.Action, cmd.Params)
	dur := int(time.Since(start).Milliseconds())
	if err != nil {
		return mustJSON(map[string]any{"id": cmd.ID, "ok": false, "error": err.Error(), "duration_ms": dur})
	}
	return mustJSON(map[string]any{"id": cmd.ID, "ok": true, "result": out, "duration_ms": dur})
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
