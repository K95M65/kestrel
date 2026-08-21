package buzz

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client dials a remote hive (another Kestrel or a Buzz-shaped websocket).
type Client struct {
	URL    string
	Device string
	Name   string
	OnNote func(Note)

	mu   sync.Mutex
	conn *websocket.Conn
}

func (c *Client) Start(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		default:
		}
		if err := c.loop(stop); err != nil {
			slog.Warn("buzz client", "component", "buzz", "error", err)
		}
		select {
		case <-stop:
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (c *Client) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

func (c *Client) loop(stop <-chan struct{}) error {
	u := strings.TrimSpace(c.URL)
	if u == "" {
		return fmt.Errorf("buzz relay url is empty")
	}
	if strings.HasPrefix(u, "http://") {
		u = "ws://" + strings.TrimPrefix(u, "http://")
	}
	if strings.HasPrefix(u, "https://") {
		u = "wss://" + strings.TrimPrefix(u, "https://")
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-stop:
			cancel()
		case <-ctx.Done():
		}
	}()
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.DialContext(ctx, parsed.String(), http.Header{})
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	slog.Info("buzz connected", "component", "buzz", "url", parsed.String())
	defer func() {
		c.mu.Lock()
		c.conn = nil
		c.mu.Unlock()
		conn.Close()
	}()
	hello, _ := json.Marshal(Note{Type: "hello", Device: c.Device, Name: c.Name})
	if err := conn.WriteMessage(websocket.TextMessage, hello); err != nil {
		return err
	}
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				_ = conn.Close()
				return
			case <-done:
				return
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
					_ = conn.Close()
					return
				}
			}
		}
	}()
	defer close(done)
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var n Note
		if json.Unmarshal(raw, &n) != nil || n.Type != "note" || n.Text == "" {
			continue
		}
		if n.From == c.Device {
			continue
		}
		if c.OnNote != nil {
			c.OnNote(n)
		}
	}
}

func (c *Client) Say(text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("empty")
	}
	n := Note{Type: "note", From: c.Device, Name: c.Name, Device: c.Device, Text: text, TS: time.Now().Unix()}
	body, _ := json.Marshal(n)
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("buzz not connected")
	}
	return conn.WriteMessage(websocket.TextMessage, body)
}
