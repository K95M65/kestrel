package picoclaw

import "testing"

func TestParsePicoclawVersion(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want string
		ok   bool
	}{
		{
			name: "release before Go toolchain version",
			out:  "PicoClaw v0.2.1 go1.25.11\n",
			want: "0.2.1",
			ok:   true,
		},
		{
			name: "Go toolchain version alone is not a release",
			out:  "PicoClaw go1.25.11\n",
			want: "",
			ok:   false,
		},
		{
			name: "unparseable output",
			out:  "PicoClaw",
			want: "",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parsePicoclawVersion(tt.out)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parse %q = (%q, %t), want (%q, %t)", tt.out, got, ok, tt.want, tt.ok)
			}
		})
	}
}
