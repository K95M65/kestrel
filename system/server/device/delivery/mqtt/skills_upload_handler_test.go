package mqtthandler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/skills"
)

func TestParseSkillsUploadData(t *testing.T) {
	want := "---\nname: daily-note\ndescription: Capture a daily note.\n---\n\n# Daily note"
	filename, content, errMsg := parseSkillsUploadData(json.RawMessage(fmt.Sprintf(`{"filename":"daily-note.md","content_base64":%q}`, base64.StdEncoding.EncodeToString([]byte(want)))))
	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	if filename != "daily-note.md" {
		t.Errorf("filename = %q", filename)
	}
	if string(content) != want {
		t.Errorf("content = %q", content)
	}
}

func TestParseSkillsUploadDataRejectsInvalidInput(t *testing.T) {
	for _, payload := range []json.RawMessage{
		json.RawMessage(`{"filename":"  ","content_base64":"YQ=="}`),
		json.RawMessage(`{"filename":"x.md","content_base64":"%%%"}`),
		json.RawMessage(`{"filename":"x.md","content_base64":`),
		json.RawMessage(fmt.Sprintf(`{"filename":"x.md","content_base64":%q}`, strings.Repeat("a", base64.StdEncoding.EncodedLen(int(skills.StoreMaxBytes))+1))),
	} {
		if filename, content, errMsg := parseSkillsUploadData(payload); errMsg == "" || filename != "" || content != nil {
			t.Errorf("parseSkillsUploadData(%s) = %q, %q, %q; want failure", payload, filename, content, errMsg)
		}
	}
}

func TestClassifySkillsUploadError(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{domain.ErrNotSupportedByRuntime, "unsupported_runtime"},
		{skills.ErrInvalidFrontMatter, "validate_front_matter"},
		{skills.ErrEmptyArchive, "archive"},
		{skills.ErrMissingSkillMD, "archive"},
		{skills.ErrInvalidSkillName, "validate_name"},
		{fmt.Errorf("disk full"), "install"},
	}
	for _, tc := range cases {
		if got := classifySkillsUploadError(tc.err); got != tc.want {
			t.Errorf("classify(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestSkillsUploadKindIsDistinct(t *testing.T) {
	if domain.KindSkillsUpload != "skills.upload" {
		t.Errorf("kind = %q, want skills.upload", domain.KindSkillsUpload)
	}
	for _, other := range []string{domain.KindSkillsInstall, domain.KindSkillsSave, domain.KindSkillsInstallStore, domain.KindSkillsFiles} {
		if domain.KindSkillsUpload == other {
			t.Errorf("skills.upload collides with %q", other)
		}
	}
}
