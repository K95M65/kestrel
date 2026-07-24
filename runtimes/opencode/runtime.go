package opencode

import (
	"context"
	"log/slog"
	"regexp"
	"sync/atomic"
	"time"

	"go.autonomous.ai/os/system/lib/core/system"
)

// opencodeVersionProbeTimeout caps a single `opencode --version` probe. OpenCode
// is a Node CLI whose cold-start can exceed a few seconds on a busy box right
// after boot; a 5s cap killed the probe and left the version blank for the whole
// process lifetime.
const opencodeVersionProbeTimeout = 20 * time.Second

// opencodeVersionProbeRetries bounds retries so a killed/empty boot-time probe
// self-heals instead of leaving the Overview card blank until the next restart.
const opencodeVersionProbeRetries = 6

// opencodeVersionProbeBackoff is the wait between failed probe attempts.
const opencodeVersionProbeBackoff = 10 * time.Second

// opencodeSemverRe extracts the release from `opencode --version` output
// (e.g. "opencode-cli 0.142.5" → "0.142.5").
var opencodeSemverRe = regexp.MustCompile(`(\d+\.\d+\.\d+(?:[-+._][0-9A-Za-z.-]+)?)`)
var opencodeVersion atomic.Pointer[string]

func GetOpenCodeVersion() string {
	if v := opencodeVersion.Load(); v != nil {
		return *v
	}
	return ""
}

// PopulateOpenCodeVersion probes `opencode --version` and caches the semver,
// retrying on a killed/empty probe (opencodeVersionProbeRetries) so a boot-time
// cold-start slowdown self-heals. Runs in a startup goroutine; a warm probe
// returns on the first try. Stops once a non-empty version is stored.
func PopulateOpenCodeVersion() {
	for attempt := 0; ; attempt++ {
		if v, ok := probeOpenCodeVersion(); ok {
			opencodeVersion.Store(&v)
			return
		}
		if attempt >= opencodeVersionProbeRetries {
			slog.Warn("read opencode version gave up after retries (expected if not on opencode backend)", "component", "opencode-probe", "attempts", attempt+1)
			return
		}
		time.Sleep(opencodeVersionProbeBackoff)
	}
}

// probeOpenCodeVersion runs a single probe; ok is false on failure/timeout or
// when no semver token is present, signalling the caller to retry.
func probeOpenCodeVersion() (version string, ok bool) {
	ctx, cancel := context.WithTimeout(context.Background(), opencodeVersionProbeTimeout)
	defer cancel()
	out, err := system.Run(ctx, "opencode", "--version")
	if err != nil {
		slog.Warn("read opencode version failed (expected if not on opencode backend)", "component", "opencode-probe", "error", err)
		return "", false
	}
	loc := opencodeSemverRe.FindStringSubmatch(string(out))
	if len(loc) <= 1 {
		return "", false
	}
	return loc[1], true
}

// Version satisfies domain.AgentGateway.Version(): the cached OpenCode CLI
// version, or empty when undetected.
func (s *OpenCodeService) Version() string {
	return GetOpenCodeVersion()
}
