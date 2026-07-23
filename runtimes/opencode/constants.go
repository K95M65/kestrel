package opencode

// Wire constants for the OpenCode backend. The opencode CLI is driven per-turn
// (`opencode run --format json`), so the device runs a thin local bridge —
// compiled into the os-server binary (runtimes/opencode/gatewayd, systemd unit
// opencode.service runs `os-server opencode-gatewayd`). The bridge spawns one
// `opencode run --format json --auto` subprocess per
// turn (resuming the persisted session id via --session) and exposes this
// WebSocket — the main os-server process only acts as a client.
//
// Frame protocol over the socket:
//
//	os-server → bridge:  {"type":"message.send","id":..,"payload":{"content":..,
//	                      "attachments":[{"type":"image","url":"data:..;base64,.."}]}}
//	                     {"type":"session.new"}   → bridge drops the session id
//	                     {"type":"ping","id":..}  → {"type":"pong"}
//	bridge → os-server:  opencode run JSONL events forwarded VERBATIM
//	                     (type: step_start / text / reasoning / tool_use /
//	                     step_finish / message.updated / session.idle /
//	                     session.error), plus {"type":"pong"} and
//	                     {"type":"bridge.status",...} / {"type":"bridge.error",...}.
//
// The run events are translated in translator.go into the same
// domain.WSEvent shape the OpenClaw handler consumes.
const (
	// WSURL is the local bridge WebSocket endpoint (served by the gatewayd —
	// runtimes/opencode/gatewayd, default port 18793).
	WSURL = "ws://127.0.0.1:18793/opencode/ws/"

	// Token is the bearer token sent in the Authorization header on connect.
	// The bridge reads the same value from /root/.opencode/.env (OPENCODE_WS_TOKEN,
	// presync-owned) — a fixed device-local token, mirroring the picoclaw and
	// claudecode contracts.
	Token = "autonomous_opencode_token"

	// Conversation is a label only — opencode owns its session ids; the real
	// session id is captured from the sessionID field on every run JSONL line.
	Conversation = "device-main"

	// opencodeHome is the backend's device-local state dir: the presync-owned
	// .env, session.json, attachments/ and the workspace/ opencode runs in.
	// (The opencode CLI's own config/data live under XDG: ~/.config/opencode
	// and ~/.local/share/opencode.) Wiped whole by ResetAgent.
	opencodeHome = "/root/.opencode"
)
