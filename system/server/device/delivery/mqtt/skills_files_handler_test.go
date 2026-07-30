package mqtthandler

import (
	"strings"
	"testing"
	"unicode/utf8"

	"go.autonomous.ai/os/system/domain"
)

func sampleSkillFiles() []domain.SkillBundleFile {
	return []domain.SkillBundleFile{
		{Path: "music/SKILL.md", Size: 12, Text: "# Music"},
		{Path: "music/reference/tempo.md", Size: 5, Text: "tempo"},
		{Path: "music/assets/icon.png", Size: 900, Binary: true},
	}
}

// List mode must be bounded: a caller asking for the listing hasn't asked for
// contents, and a skill's combined text can exceed what a broker will carry.
func TestStripSkillFileText(t *testing.T) {
	got := stripSkillFileText(sampleSkillFiles())
	if len(got) != 3 {
		t.Fatalf("want 3 entries, got %d", len(got))
	}
	for _, f := range got {
		if f.Text != "" || f.Truncated {
			t.Errorf("%s still carries a body: text=%q truncated=%v", f.Path, f.Text, f.Truncated)
		}
	}
	// Metadata survives so a client can render the list and pick a file.
	if got[0].Path != "music/SKILL.md" || got[0].Size != 12 {
		t.Errorf("metadata lost: %+v", got[0])
	}
	if !got[2].Binary {
		t.Error("binary flag lost")
	}
}

func TestCapSkillFileText(t *testing.T) {
	// Under the cap: untouched.
	small := domain.SkillBundleFile{Path: "a.md", Size: 7, Text: "small"}
	if got := capSkillFileText(small); got != small {
		t.Errorf("small file was modified: %+v", got)
	}

	big := strings.Repeat("a", mqttMaxFileText+500)
	got := capSkillFileText(domain.SkillBundleFile{Path: "b.md", Size: int64(len(big)), Text: big})
	if !got.Truncated {
		t.Error("oversized file must be flagged truncated")
	}
	if len(got.Text) > mqttMaxFileText {
		t.Errorf("inlined %d bytes, cap is %d", len(got.Text), mqttMaxFileText)
	}
	// Size keeps reporting the real length, not the truncated preview's.
	if got.Size != int64(len(big)) {
		t.Errorf("size = %d, want %d", got.Size, len(big))
	}
}

// A cut must not split a multi-byte rune, or json.Marshal replaces the tail with
// U+FFFD and the client sees corruption instead of a clean truncation.
func TestCapSkillFileTextKeepsValidUTF8(t *testing.T) {
	// Pad so the cap lands in the middle of a 3-byte rune.
	text := strings.Repeat("a", mqttMaxFileText-1) + "日本語"
	got := capSkillFileText(domain.SkillBundleFile{Path: "c.md", Text: text})

	if !utf8.ValidString(got.Text) {
		t.Fatalf("truncated text is not valid UTF-8 (%d bytes)", len(got.Text))
	}
	if !got.Truncated {
		t.Error("must be flagged truncated")
	}
}
