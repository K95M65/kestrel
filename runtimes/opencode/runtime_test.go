package opencode

import "testing"

func TestParseOpenCodeVersion(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want string
		ok   bool
	}{
		{name: "version on first line", out: "opencode-cli 0.142.5\n", want: "0.142.5", ok: true},
		{name: "pre-release version", out: "opencode 1.2.3-rc.1", want: "1.2.3-rc.1", ok: true},
		{name: "unparseable output", out: "opencode", want: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseOpenCodeVersion(tt.out)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parse %q = (%q, %t), want (%q, %t)", tt.out, got, ok, tt.want, tt.ok)
			}
		})
	}
}
