package opencode

import (
	"context"
	"log/slog"
	"regexp"
	"sync/atomic"
	"time"

	"go.autonomous.ai/os/system/lib/core/system"
)

const opencodeVersionProbeTimeout = 5 * time.Second

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

func PopulateOpenCodeVersion() {
	ctx, cancel := context.WithTimeout(context.Background(), opencodeVersionProbeTimeout)
	defer cancel()
	out, err := system.Run(ctx, "opencode", "--version")
	if err != nil {
		slog.Warn("read opencode version failed (expected if not on opencode backend)", "component", "opencode-probe", "error", err)
		return
	}
	v := ""
	if loc := opencodeSemverRe.FindStringSubmatch(string(out)); len(loc) > 1 {
		v = loc[1]
	}
	opencodeVersion.Store(&v)
}

// Version satisfies domain.AgentGateway.Version(): the cached OpenCode CLI
// version, or empty when undetected.
func (s *OpenCodeService) Version() string {
	return GetOpenCodeVersion()
}
