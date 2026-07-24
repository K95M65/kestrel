package picoclaw

import (
	"context"
	"log/slog"
	"regexp"
	"sync/atomic"
	"time"

	"go.autonomous.ai/os/system/lib/core/system"
)

// picoclawVersionProbeTimeout caps a single `picoclaw version` probe. Kept
// generous so a cold-start slowdown on a busy box right after boot can't get the
// probe killed and leave the version blank for the whole process lifetime.
const picoclawVersionProbeTimeout = 20 * time.Second

// picoclawVersionProbeRetries bounds retries so a killed/empty boot-time probe
// self-heals instead of leaving the Overview card blank until the next restart.
const picoclawVersionProbeRetries = 6

// picoclawVersionProbeBackoff is the wait between failed probe attempts.
const picoclawVersionProbeBackoff = 10 * time.Second

var picoclawGoVersionRe = regexp.MustCompile(`(?i)\bgo\d+\.\d+\.\d+\b`)
var picoclawSemverRe = regexp.MustCompile(`(\d+\.\d+\.\d+(?:[-+._][0-9A-Za-z.-]+)?)`)
var picoclawVersion atomic.Pointer[string]

func GetPicoclawVersion() string {
	if v := picoclawVersion.Load(); v != nil {
		return *v
	}
	return ""
}

// PopulatePicoclawVersion probes `picoclaw version` and caches the semver,
// retrying on a killed/empty probe (picoclawVersionProbeRetries) so a boot-time
// cold-start slowdown self-heals. Runs in a startup goroutine; a warm probe
// returns on the first try. Stops once a non-empty version is stored.
func PopulatePicoclawVersion() {
	for attempt := 0; ; attempt++ {
		if v, ok := probePicoclawVersion(); ok {
			picoclawVersion.Store(&v)
			return
		}
		if attempt >= picoclawVersionProbeRetries {
			slog.Warn("read picoclaw version gave up after retries (expected if not on picoclaw backend)", "component", "picoclaw-probe", "attempts", attempt+1)
			return
		}
		time.Sleep(picoclawVersionProbeBackoff)
	}
}

// probePicoclawVersion runs a single probe; ok is false on failure/timeout or
// when no semver token is present, signalling the caller to retry.
func probePicoclawVersion() (version string, ok bool) {
	ctx, cancel := context.WithTimeout(context.Background(), picoclawVersionProbeTimeout)
	defer cancel()
	out, err := system.Run(ctx, "picoclaw", "version")
	if err != nil {
		slog.Warn("read picoclaw version failed (expected if not on picoclaw backend)", "component", "picoclaw-probe", "error", err)
		return "", false
	}
	// Drop the Go toolchain version so "go1.25.11" is never matched as the release.
	cleaned := picoclawGoVersionRe.ReplaceAllString(string(out), "")
	loc := picoclawSemverRe.FindStringSubmatch(cleaned)
	if len(loc) <= 1 {
		return "", false
	}
	return loc[1], true
}

// Version satisfies domain.AgentGateway.Version(): the cached PicoClaw CLI
// version, or empty when undetected.
func (s *PicoclawService) Version() string {
	return GetPicoclawVersion()
}
