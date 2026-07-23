package opencode

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Coding-session (opencode session) discovery for the Telegram remote-coding
// feature (telegram_coding.go). opencode stores its sessions under
// ~/.local/share/opencode in an internal format — NOT the parseable per-folder
// rollout transcripts codex used — so there is no on-disk cwd we can recover to
// build a cross-folder "resumable sessions" list. `opencode session list` exists
// (table/JSON output, --max-count) but its JSON shape and directory scoping are
// unconfirmed, and it lists sessions for the current runtime context by default
// rather than every folder on the device. Until that is verified on-device,
// discovery is degraded to empty (allCodingSessions returns nil); /new <folder>
// plus per-turn resume via `--session <id>` still work. The retained helpers
// (normalizeFolder, humanizeAgo, oneLine) and the codingSession type keep
// telegram_coding.go's /resume + /use flow compiling and behaving sanely (an
// empty list yields a "no sessions yet" message).

// codingSession is one resumable opencode session. Discovery is currently
// degraded (see allCodingSessions), so these are only ever populated by an
// explicit selection, not by on-disk scanning.
type codingSession struct {
	Folder    string    // working dir — passed to `opencode run --dir`
	SessionID string    // opencode session id — passed to `opencode run --session <id>`
	Modified  time.Time // recency for listing / newest-per-folder
	Recent    []string  // short recent-prompt lines, most-recent first (unused while discovery is degraded)
}

// label is the single-line description for a session (its most recent prompt).
func (c codingSession) label() string {
	if len(c.Recent) > 0 {
		return c.Recent[0]
	}
	return "(no description)"
}

// allCodingSessions returns every resumable opencode session, most-recent first.
//
// TODO(opencode-coding-sessions): opencode stores sessions under
// ~/.local/share/opencode in an internal format and `opencode session list`
// does not (confirmably) expose the working directory needed to resume in the
// right folder; wire this up once verified on-device. Until then this returns
// nil so the Telegram /resume list is honestly empty rather than misleading.
func (s *OpenCodeService) allCodingSessions() []codingSession {
	return nil
}

// codingFolders returns the NEWEST session per folder, most-recent folder first
// — the default /sessions view (one line per project). Empty while discovery is
// degraded.
func (s *OpenCodeService) codingFolders() []codingSession {
	all := s.allCodingSessions()
	seen := map[string]bool{}
	var out []codingSession
	for _, cs := range all { // already newest-first, so the first hit per folder wins
		if seen[cs.Folder] {
			continue
		}
		seen[cs.Folder] = true
		out = append(out, cs)
	}
	return out
}

// folderSessions returns all sessions in one folder, newest first. Empty while
// discovery is degraded.
func (s *OpenCodeService) folderSessions(folder string) []codingSession {
	folder = normalizeFolder(folder)
	var out []codingSession
	for _, cs := range s.allCodingSessions() {
		if cs.Folder == folder {
			out = append(out, cs)
		}
	}
	return out
}

// latestSessionForFolder returns the newest session in folder (ok=false if
// none). Always ok=false while discovery is degraded.
func (s *OpenCodeService) latestSessionForFolder(folder string) (codingSession, bool) {
	sessions := s.folderSessions(folder)
	if len(sessions) == 0 {
		return codingSession{}, false
	}
	return sessions[0], true
}

// normalizeFolder cleans a user-supplied path: trims quotes/space, expands a
// leading ~ to /root, makes it absolute (relative paths resolve under /root)
// and drops any trailing slash.
func normalizeFolder(p string) string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, `"'`)
	if p == "" {
		return ""
	}
	switch {
	case p == "~":
		p = "/root"
	case strings.HasPrefix(p, "~/"):
		p = filepath.Join("/root", p[2:])
	case !filepath.IsAbs(p):
		p = filepath.Join("/root", p)
	}
	return filepath.Clean(p)
}

// oneLine collapses whitespace/newlines into single spaces for a compact label.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// humanizeAgo renders how long ago t was, for session listings.
func humanizeAgo(t time.Time) string {
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
