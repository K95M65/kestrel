package http

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// Serving a device-local file the agent produced, so the web chat can SHOW it.
//
// The chat can already send a file INTO a turn (base64 on POST /sensing/event),
// but there was no way back out: ask the agent for a photo and it answers with
// an absolute path — `/root/.openclaw/media/hal-snapshots/snap_*.jpg` — which a
// browser cannot read. The web detects such a path in a reply and points an
// <img>/download at this endpoint.
//
// The client sends the RAW PATH, so this handler treats it as hostile input.
// Two independent gates, either of which is enough to refuse:
//
//	1. the path must resolve (symlinks included) inside an allow-listed root
//	2. its extension must be on the served-types list
//
// Deliberately NOT served: `.json` and `.log`. Both are plausible agent output,
// but a runtime's own config JSON carries gateway tokens — the same reason
// GET /api/agent/config-json is loopback-only — and one mis-scoped root would
// turn a viewer into a credential leak. Images and documents are the case that
// motivated this; the rest can be added when something concrete needs it.
//
// A narrower, path-free variant of this already exists for Flow Monitor tool
// results (SensingHandler.GetAgentSnapshot, which takes runtime/source/name
// segments instead). That one stays as is: it serves what the DEVICE resolved,
// while this serves what the AGENT named.

// agentFileMaxBytes caps a single response. Generous for a snapshot or a PDF,
// small enough that a stray path to something huge fails fast.
const agentFileMaxBytes = 32 << 20

// agentFileTypes is the whitelist: extension → Content-Type. An extension that
// isn't here is refused before the filesystem is touched.
var agentFileTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".pdf":  "application/pdf",
	".txt":  "text/plain; charset=utf-8",
	".md":   "text/plain; charset=utf-8",
	".csv":  "text/csv; charset=utf-8",
	".wav":  "audio/wav",
	".mp3":  "audio/mpeg",
	".mp4":  "video/mp4",
	".webm": "video/webm",
}

// agentFileInline is the subset rendered in the page rather than downloaded.
var agentFileInline = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".pdf": true,
}

// agentFileRuntimes mirrors the runtimes GetAgentSnapshot accepts. Every known
// runtime is allow-listed rather than just the active one: a conversation
// scrolled back far enough can reference a file written before a runtime
// switch, and the file is no more sensitive for having been made by the
// previous backend.
var agentFileRuntimes = []string{"openclaw", "hermes", "picoclaw", "codex", "claudecode", "opencode"}

// defaultAgentFileRoots is where an agent's output is allowed to live.
//
// Per runtime this is media/ and workspace/ — NOT the runtime's config dir
// itself, which holds openclaw.json and friends. /tmp is included because the
// CLI-based runtimes genuinely write scratch output there; it is the reason the
// extension whitelist above exists, since /tmp is shared with everything else on
// the device.
func defaultAgentFileRoots() []string {
	roots := make([]string, 0, len(agentFileRuntimes)*2+1)
	for _, rt := range agentFileRuntimes {
		home := "/root/." + rt
		roots = append(roots, filepath.Join(home, "media"), filepath.Join(home, "workspace"))
	}
	return append(roots, "/tmp")
}

var (
	// errFileType is an extension that is not served at all.
	errFileType = errors.New("file type not served")
	// errOutsideRoots is a path that resolved outside every allow-listed root.
	errOutsideRoots = errors.New("path is outside the served roots")
	// errFileNotFound covers absent, unreadable, non-regular and oversized.
	errFileNotFound = errors.New("file not found")
)

// resolveAgentFile validates raw against roots and returns the path to serve
// plus its Content-Type. Split out from the handler so the traversal, symlink
// and prefix cases are testable without a router.
//
// Order matters: the extension is checked FIRST, so a probe for a file type we
// never serve cannot be used to test whether a path exists.
func resolveAgentFile(raw string, roots []string) (path, contentType string, err error) {
	if raw == "" || !filepath.IsAbs(raw) {
		return "", "", errFileNotFound
	}

	ct, ok := agentFileTypes[strings.ToLower(filepath.Ext(raw))]
	if !ok {
		return "", "", errFileType
	}

	// Resolve BEFORE comparing: `..` segments and symlinks both let a path that
	// reads as being under a root actually land outside it.
	resolved, err := filepath.EvalSymlinks(filepath.Clean(raw))
	if err != nil {
		return "", "", errFileNotFound
	}

	if !underAnyRoot(resolved, roots) {
		return "", "", errOutsideRoots
	}

	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return "", "", errFileNotFound
	}
	if info.Size() > agentFileMaxBytes {
		return "", "", errFileNotFound
	}
	return resolved, ct, nil
}

// underAnyRoot reports whether resolved sits inside one of roots. Roots are
// resolved too — /tmp is a symlink to /private/tmp on macOS, and a root that
// only matched textually would reject every real file under it.
//
// The separator suffix is what stops "/root/.openclaw/media-evil" from passing
// as being under "/root/.openclaw/media".
func underAnyRoot(resolved string, roots []string) bool {
	for _, root := range roots {
		r, err := filepath.EvalSymlinks(root)
		if err != nil {
			continue // root doesn't exist on this device — nothing can be under it
		}
		if resolved == r || strings.HasPrefix(resolved, r+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// ServeFile handles GET /api/agent/file?path=<absolute path>.
//
// Bare status codes rather than the usual JSON envelope: the only caller is an
// <img> / download link, which reads the status and nothing else.
func (h *AgentHandler) ServeFile(c *gin.Context) {
	path, contentType, err := resolveAgentFile(c.Query("path"), defaultAgentFileRoots())
	if err != nil {
		switch {
		case errors.Is(err, errFileType), errors.Is(err, errOutsideRoots):
			c.Status(http.StatusForbidden)
		default:
			c.Status(http.StatusNotFound)
		}
		return
	}

	disposition := "attachment"
	if agentFileInline[strings.ToLower(filepath.Ext(path))] {
		disposition = "inline"
	}
	// filename is the basename only — the full path is already known to this
	// caller, but nothing downstream should re-derive a directory from a header.
	c.Header("Content-Disposition", disposition+`; filename="`+filepath.Base(path)+`"`)
	c.Header("Content-Type", contentType)
	// The whitelist decides the type; never let a sniffed one override it.
	c.Header("X-Content-Type-Options", "nosniff")
	c.File(path)
}
