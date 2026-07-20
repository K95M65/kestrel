package system

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// shellUpgrader allows any origin — same-origin enforcement is already handled
// at the network/proxy layer (this endpoint is only reachable on the LAN).
var shellUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(_ *http.Request) bool { return true },
}

// ShellHandler upgrades the request to a WebSocket and pipes a /bin/bash PTY
// in both directions. Frames from the client are stdin bytes by default; small
// JSON envelopes with `type: "resize"` resize the PTY (rows/cols).
//
// agentEnvFile, when non-nil, is resolved per-connection to the active agent
// runtime's launch env file (e.g. claudecode's /root/.claudecode/.env). Its
// KEY=VALUE pairs are merged into the PTY env (plus IS_SANDBOX=1) so an
// interactive `claude` in the web CLI reuses the campaign API key instead of
// prompting login — the file is otherwise only injected into the gatewayd
// service by systemd, so a bare login shell would not see it. Returning "" (or
// a missing file) skips the injection.
//
// Client → server frames:
//   - TextMessage starting with '{' and ending with '}' AND parseable as
//     {"type":"resize","rows":N,"cols":M}  ⇒ window resize signal
//   - Anything else (text or binary)       ⇒ raw stdin bytes
//
// Server → client frames: raw stdout/stderr bytes as binary messages.
func ShellHandler(agentEnvFile func() string) gin.HandlerFunc {
	return func(c *gin.Context) {
		shellSession(c, agentEnvFile)
	}
}

func shellSession(c *gin.Context, agentEnvFile func() string) {
	ws, err := shellUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[shell] upgrade failed: %v", err)
		return
	}
	defer ws.Close()

	// Spawn an interactive login bash so the user gets aliases, $PATH, prompt.
	cmd := exec.Command("/bin/bash", "-il")
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
	)
	if agentEnvFile != nil {
		if extra := loadAgentEnv(agentEnvFile()); len(extra) > 0 {
			cmd.Env = append(cmd.Env, extra...)
		}
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("[shell] pty start failed: %v", err)
		_ = ws.WriteMessage(websocket.TextMessage, []byte("\r\n[shell] failed to start PTY: "+err.Error()+"\r\n"))
		return
	}
	defer func() {
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_, _ = cmd.Process.Wait()
	}()

	// Initial size — client will send a resize frame on connect to override.
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

	// One writer mutex: WebSocket connections require all writes to be serialized.
	var writeMu sync.Mutex
	writeBytes := func(t int, b []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
		return ws.WriteMessage(t, b)
	}

	done := make(chan struct{})
	var closeOnce sync.Once
	closeDone := func() { closeOnce.Do(func() { close(done) }) }

	// PTY → WebSocket. Read in 4KB chunks; xterm.js handles ANSI just fine.
	go func() {
		defer closeDone()
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				if werr := writeBytes(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("[shell] pty read: %v", err)
				}
				return
			}
		}
	}()

	// WebSocket → PTY. Loop ends when the client closes or we get an error.
	for {
		select {
		case <-done:
			return
		default:
		}
		mt, data, err := ws.ReadMessage()
		if err != nil {
			closeDone()
			return
		}

		// Try to interpret as a control envelope (resize) — only for text frames
		// that look like JSON. Anything else goes straight to PTY stdin.
		if mt == websocket.TextMessage && len(data) > 1 && data[0] == '{' {
			var env struct {
				Type string `json:"type"`
				Rows uint16 `json:"rows"`
				Cols uint16 `json:"cols"`
			}
			if jerr := json.Unmarshal(data, &env); jerr == nil && env.Type == "resize" {
				if env.Rows == 0 {
					env.Rows = 24
				}
				if env.Cols == 0 {
					env.Cols = 80
				}
				_ = pty.Setsize(ptmx, &pty.Winsize{Rows: env.Rows, Cols: env.Cols})
				continue
			}
		}

		if _, werr := ptmx.Write(data); werr != nil {
			log.Printf("[shell] pty write: %v", werr)
			closeDone()
			return
		}
	}
}

// loadAgentEnv parses a KEY=VALUE launch env file (blank/#/no-"=" lines
// skipped, keys/values trimmed, surrounding double quotes stripped — same rules
// as the gatewayd child env loader) and returns "KEY=VALUE" entries for the PTY
// env. IS_SANDBOX=1 is appended when the file loads so a root `claude
// --dangerously-skip-permissions` is allowed. An empty path or unreadable file
// yields nil (no injection).
func loadAgentEnv(path string) []string {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, "=")
		if i < 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.Trim(strings.TrimSpace(line[i+1:]), `"`)
		if key == "" {
			continue
		}
		out = append(out, key+"="+val)
	}
	if len(out) == 0 {
		return nil
	}
	// The device runs as root; claude refuses --dangerously-skip-permissions
	// under uid 0 unless IS_SANDBOX=1 (mirrors the gatewayd child env).
	return append(out, "IS_SANDBOX=1")
}
