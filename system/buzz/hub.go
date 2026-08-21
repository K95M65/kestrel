// Package buzz is a LAN hive so Kestrel bodies and agents can talk to each
// other. The wire is a small JSON websocket (hello + note). Point another
// unit at this host, or later at a Block Buzz / Nostr relay URL — same
// Device → Channels card.
package buzz

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Note is one hive message. Secrets never go here.
type Note struct {
	Type   string `json:"type"`
	ID     string `json:"id,omitempty"`
	From   string `json:"from,omitempty"`
	Name   string `json:"name,omitempty"`
	Text   string `json:"text,omitempty"`
	TS     int64  `json:"ts,omitempty"`
	Device string `json:"device,omitempty"`
}

const (
	pingPeriod = 30 * time.Second
	pongWait   = 60 * time.Second
	writeWait  = 8 * time.Second
)

type client struct {
	conn   *websocket.Conn
	send   chan []byte
	device string
	name   string
}

// Hub fans notes to every connected peer, including the local agent.
type Hub struct {
	mu      sync.Mutex
	clients map[*client]struct{}
	onNote  func(Note)
}

func NewHub() *Hub {
	return &Hub{clients: map[*client]struct{}{}}
}

// SetHandler is called for every note except our own echoes.
func (h *Hub) SetHandler(fn func(Note)) {
	h.mu.Lock()
	h.onNote = fn
	h.mu.Unlock()
}

func (h *Hub) Broadcast(n Note) {
	if n.TS == 0 {
		n.TS = time.Now().Unix()
	}
	if n.Type == "" {
		n.Type = "note"
	}
	body, err := json.Marshal(n)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c.send <- body:
		default:
		}
	}
}

func (h *Hub) handler() func(Note) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.onNote
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWS is GET /api/buzz/ws.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("buzz upgrade", "component", "buzz", "error", err)
		return
	}
	c := &client{conn: conn, send: make(chan []byte, 8)}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	n := len(h.clients)
	h.mu.Unlock()
	slog.Info("buzz peer joined", "component", "buzz", "peers", n)
	go c.writePump()
	c.readPump(h)
	h.mu.Lock()
	delete(h.clients, c)
	left := len(h.clients)
	h.mu.Unlock()
	conn.Close()
	slog.Info("buzz peer left", "component", "buzz", "peers", left)
}

func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case body, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, body); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
				return
			}
		}
	}
}

func (c *client) readPump(h *Hub) {
	defer close(c.send)
	c.conn.SetReadLimit(1 << 16)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var n Note
		if json.Unmarshal(raw, &n) != nil {
			continue
		}
		switch n.Type {
		case "hello":
			c.device = n.Device
			c.name = n.Name
		case "note", "":
			if n.Text == "" {
				continue
			}
			n.Type = "note"
			n.From = c.device
			if n.Name == "" {
				n.Name = c.name
			}
			n.Device = c.device
			if n.TS == 0 {
				n.TS = time.Now().Unix()
			}
			h.Broadcast(n)
			if fn := h.handler(); fn != nil {
				fn(n)
			}
		}
	}
}

// Count is connected peers (not counting this process).
func (h *Hub) Count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}
