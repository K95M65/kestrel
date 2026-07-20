package main

// `os-server claude-sessions` — the unified claude coding-session picker for
// the device terminal.
//
// Claude's interactive `/resume` picker excludes headless (`claude -p`)
// sessions BY DESIGN, so sessions created over Telegram remote-coding
// (runtimes/claudecode/telegram_coding.go) never show up in it — but
// `claude --resume <id>` opens ANY session by id, including headless ones
// (device-proven). This subcommand closes that gap: it lists every session
// for the current folder (or all folders with --all) using the SAME discovery
// the Telegram feature uses (claudecode.ListCodingSessions — one source of
// truth), lets the user pick one by number, and execs `claude --resume <id>`
// in the session's folder.
//
// Codex needs no picker: its own `codex resume` lists every thread globally,
// including Telegram-created ones — claude-sessions just points there when
// the active runtime is codex.
//
// The claudecode presync installs a thin `/usr/local/bin/claude-sessions`
// wrapper that sudo-reexecs into this subcommand, so on the device it is just
// `claude-sessions`. (Internal cc* naming predates the rename from `cc`.)

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.autonomous.ai/os/runtimes/claudecode"
)

// ccConfigJSON is the device config carrying agent_runtime.
const ccConfigJSON = "/root/config/config.json"

func ccMain(args []string) int {
	fs := flag.NewFlagSet("claude-sessions", flag.ContinueOnError)
	var all, asJSON bool
	fs.BoolVar(&all, "a", false, "list sessions from every folder, not just the current directory")
	fs.BoolVar(&all, "all", false, "alias of -a")
	fs.BoolVar(&asJSON, "json", false, "print the listing as JSON and exit (no picker)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: claude-sessions [-a] [--json] [folder]\nClaude coding-session picker: lists every session (terminal- AND Telegram-created) and resumes the picked one.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if rt := ccAgentRuntime(ccConfigJSON); rt != "" && rt != "claudecode" {
		if rt == "codex" {
			fmt.Println("Active runtime is codex — its own picker already lists every thread (including Telegram ones): run `codex resume`.")
		} else {
			fmt.Printf("Active runtime is %s — claude-sessions only picks claude sessions (runtime claudecode).\n", rt)
		}
		return 0
	}

	sessions := claudecode.ListCodingSessions()

	scope := "all folders"
	if !all {
		dir, err := ccScopeDir(fs.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, "claude-sessions:", err)
			return 1
		}
		scope = dir
		sessions = ccFilterFolder(sessions, dir)
	}

	if asJSON {
		if sessions == nil {
			sessions = []claudecode.CodingSessionInfo{} // emit [] not null
		}
		if err := json.NewEncoder(os.Stdout).Encode(sessions); err != nil {
			return 1
		}
		return 0
	}
	if len(sessions) == 0 {
		if all {
			fmt.Println("No claude coding sessions found on this device.")
		} else {
			fmt.Printf("No claude sessions in %s.\nTry `claude-sessions -a` to list every folder, or start one with `claude`.\n", scope)
		}
		return 0
	}

	ccPrintMenu(scope, sessions, all)
	pick, ok := ccReadPick(len(sessions))
	if !ok {
		return 0
	}
	if err := ccResume(sessions[pick-1]); err != nil {
		fmt.Fprintln(os.Stderr, "claude-sessions:", err)
		return 1
	}
	return 0 // unreachable: ccResume execs on success
}

// ccAgentRuntime resolves the active runtime from config.json ("" when the
// file is unreadable or the field is absent — treated as claudecode).
func ccAgentRuntime(configPath string) string {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	var c struct {
		AgentRuntime string `json:"agent_runtime"`
	}
	if json.Unmarshal(data, &c) != nil {
		return ""
	}
	return c.AgentRuntime
}

// ccScopeDir resolves the folder to list: the optional positional arg, else
// the current directory.
func ccScopeDir(arg string) (string, error) {
	if strings.TrimSpace(arg) != "" {
		abs, err := filepath.Abs(strings.TrimSpace(arg))
		if err != nil {
			return "", err
		}
		return filepath.Clean(abs), nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(cwd), nil
}

// ccFilterFolder keeps only the sessions whose cwd is folder.
func ccFilterFolder(sessions []claudecode.CodingSessionInfo, folder string) []claudecode.CodingSessionInfo {
	var out []claudecode.CodingSessionInfo
	for _, cs := range sessions {
		if cs.Folder == folder {
			out = append(out, cs)
		}
	}
	return out
}

// ccPrintMenu renders the numbered listing: index + (folder with --all) +
// recent prompts + short id + age.
func ccPrintMenu(scope string, sessions []claudecode.CodingSessionInfo, all bool) {
	fmt.Printf("claude sessions in %s (newest first):\n\n", scope)
	for i, cs := range sessions {
		fmt.Printf("%3d. [%s]  %s\n", i+1, ccShortID(cs.SessionID), ccAgo(cs.Modified))
		if all {
			fmt.Printf("     📂 %s\n", cs.Folder)
		}
		if len(cs.Recent) == 0 {
			fmt.Println("     📝 (no description)")
		}
		for j, p := range cs.Recent {
			marker := "📝"
			if j > 0 {
				marker = "  ↳"
			}
			fmt.Printf("     %s %s\n", marker, p)
		}
	}
	fmt.Println()
}

// ccReadPick reads a 1-based menu choice from stdin (empty / q cancels).
func ccReadPick(n int) (int, bool) {
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("Pick a session [1-%d], q to quit: ", n)
		line, err := r.ReadString('\n')
		if err != nil && line == "" {
			return 0, false
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.EqualFold(line, "q") {
			return 0, false
		}
		if k, err := strconv.Atoi(line); err == nil && k >= 1 && k <= n {
			return k, true
		}
		fmt.Println("Invalid choice.")
	}
}

// ccResume replaces this process with interactive `claude --resume <id>` run
// in the session's own folder (claude resume is cwd-scoped).
func ccResume(cs claudecode.CodingSessionInfo) error {
	path, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH: %w", err)
	}
	if err := os.Chdir(cs.Folder); err != nil {
		return fmt.Errorf("enter session folder: %w", err)
	}
	fmt.Printf("→ claude --resume %s (in %s)\n", cs.SessionID, cs.Folder)
	return syscall.Exec(path, []string{"claude", "--resume", cs.SessionID}, ccChildEnv())
}

// ccChildEnv builds the exec env: the process env overlaid with the presync
// .env (auth vars — cc may be reached from a shell that never sourced
// /etc/profile.d/agent-cli-env.sh, e.g. via the sudo re-exec in the wrapper)
// plus the same vars that profile.d snippet exports. Later duplicates win via
// dedupe.
func ccChildEnv() []string {
	env := os.Environ()
	env = append(env, ccEnvFilePairs("/root/.claudecode/.env")...)
	env = append(env, "IS_SANDBOX=1", "HOME=/root")
	return ccDedupeEnv(env)
}

// ccEnvFilePairs parses a KEY=VALUE env file into "KEY=VALUE" entries — same
// rules as the gatewayd child loader (claudecode.loadEnvFilePairs): blank/#/
// no-"=" lines skipped, trimmed, surrounding double quotes stripped.
func ccEnvFilePairs(path string) []string {
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
		if key == "" {
			continue
		}
		val := strings.Trim(strings.TrimSpace(line[i+1:]), `"`)
		out = append(out, key+"="+val)
	}
	return out
}

// ccDedupeEnv collapses duplicate KEY= entries, last occurrence winning, so
// the execve child never sees two values for one variable.
func ccDedupeEnv(pairs []string) []string {
	idx := map[string]int{}
	out := make([]string, 0, len(pairs))
	for _, kv := range pairs {
		key := kv
		if i := strings.Index(kv, "="); i >= 0 {
			key = kv[:i]
		}
		if j, ok := idx[key]; ok {
			out[j] = kv
			continue
		}
		idx[key] = len(out)
		out = append(out, kv)
	}
	return out
}

// ccShortID compacts a session uuid for the listing (resume uses the full id).
func ccShortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// ccAgo renders how long ago t was (mirrors claudecode.humanizeAgo).
func ccAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
