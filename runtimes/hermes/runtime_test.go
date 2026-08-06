package hermes

import "testing"

func TestParseHermesVersion(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want string
		ok   bool
	}{
		{
			name: "version on first line",
			out:  "Hermes Agent v0.17.0 (2026.6.19)\n",
			want: "0.17.0",
			ok:   true,
		},
		{
			name: "pre-release version",
			out:  "hermes 0.18.0-rc.1",
			want: "0.18.0-rc.1",
			ok:   true,
		},
		{
			name: "unparseable output",
			out:  "Hermes Agent",
			want: "",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseHermesVersion(tt.out)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parse %q = (%q, %t), want (%q, %t)", tt.out, got, ok, tt.want, tt.ok)
			}
		})
	}
}
