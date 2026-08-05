package codex

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"go.autonomous.ai/os/system/lib/core/system"
)

// codexVersionProbeTimeout caps one `/usr/local/bin/codex --version` probe.
// Codex can be slow to start immediately after boot, so keep this aligned with
// the generous cold-start allowance used by the other CLI runtimes.
const codexVersionProbeTimeout = 20 * time.Second

// codexVersionProbeRetries lets a transient first-boot failure self-heal
// instead of leaving the Monitor's Agent version blank until os-server is
// restarted. A warm probe returns immediately, so retries affect only failures.
const codexVersionProbeRetries = 6

// codexVersionProbeBackoff is the wait between failed Codex version probes.
const codexVersionProbeBackoff = 10 * time.Second

// codexBinary is the absolute path installed by install.sh. Avoiding PATH keeps
// the startup probe independent of the os-server service environment.
const codexBinary = "/usr/local/bin/codex"

// codexSemverRe extracts the release from `codex --version` output
// (e.g. "codex-cli 0.142.5" → "0.142.5").
var codexSemverRe = regexp.MustCompile(`(\d+\.\d+\.\d+(?:[-+._][0-9A-Za-z.-]+)?)`)
var codexVersion atomic.Pointer[string]

func GetCodexVersion() string {
	if v := codexVersion.Load(); v != nil {
		return *v
	}
	return ""
}

// PopulateCodexVersion shells out to `codex --version`, normalizes the semver,
// and caches it. It retries failed or unparseable probes so boot-time CLI startup
// races do not leave the Monitor's Agent version blank for the process lifetime.
func PopulateCodexVersion() {
	for attempt := 0; ; attempt++ {
		if v, ok := probeCodexVersion(); ok {
			codexVersion.Store(&v)
			return
		}
		if attempt >= codexVersionProbeRetries {
			slog.Warn("read codex version gave up after retries (expected if not on codex backend)", "component", "codex-probe", "attempts", attempt+1)
			return
		}
		time.Sleep(codexVersionProbeBackoff)
	}
}

// probeCodexVersion runs one version probe. A failed command, timeout, or
// unparseable output returns ok=false so PopulateCodexVersion can retry.
func probeCodexVersion() (version string, ok bool) {
	ctx, cancel := context.WithTimeout(context.Background(), codexVersionProbeTimeout)
	defer cancel()
	out, err := system.Run(ctx, codexBinary, "--version")
	if err != nil {
		slog.Warn("read codex version failed (expected if not on codex backend)", "component", "codex-probe", "error", err)
		return "", false
	}
	return parseCodexVersion(string(out))
}

func parseCodexVersion(output string) (version string, ok bool) {
	line := strings.TrimSpace(output)
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = strings.TrimSpace(line[:i])
	}
	if loc := codexSemverRe.FindStringSubmatch(line); len(loc) > 1 {
		return loc[1], true
	}
	return "", false
}

// Version satisfies domain.AgentGateway.Version(): the cached Codex CLI
// version, or empty when undetected.
func (s *CodexService) Version() string {
	return GetCodexVersion()
}
