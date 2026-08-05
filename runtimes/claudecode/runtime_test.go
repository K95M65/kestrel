package claudecode

import "testing"

func TestParseClaudeCodeVersion(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want string
		ok   bool
	}{
		{name: "standard output", out: "2.1.83 (Claude Code)\n", want: "2.1.83", ok: true},
		{name: "prerelease", out: "Claude Code v2.2.0-beta.1", want: "2.2.0-beta.1", ok: true},
		{name: "first line only", out: "Claude Code\n2.1.83", want: "", ok: false},
		{name: "no version", out: "Claude Code", want: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseClaudeCodeVersion(tt.out)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parse version from %q = (%q, %t), want (%q, %t)", tt.out, got, ok, tt.want, tt.ok)
			}
		})
	}
}
