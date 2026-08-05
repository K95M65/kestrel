package codex

import "testing"

func TestParseCodexVersion(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want string
		ok   bool
	}{
		{name: "standard", out: "codex-cli 0.142.5", want: "0.142.5", ok: true},
		{name: "prerelease", out: "codex-cli 0.142.5-beta.1", want: "0.142.5-beta.1", ok: true},
		{name: "first line only", out: "codex-cli 0.142.5\nother 9.9.9", want: "0.142.5", ok: true},
		{name: "unparseable", out: "not installed", want: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseCodexVersion(tt.out)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parseCodexVersion(%q) = (%q, %v), want (%q, %v)", tt.out, got, ok, tt.want, tt.ok)
			}
		})
	}
}
