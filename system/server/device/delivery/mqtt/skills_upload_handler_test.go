package mqtthandler

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/skills"
)

func TestParseSkillsUploadData(t *testing.T) {
	content, errMsg := parseSkillsUploadData(json.RawMessage(`{"content":"---\\nname: daily-note\\ndescription: Capture a daily note.\\n---\\n\\n# Daily note"}`))
	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	if !strings.Contains(content, "name: daily-note") {
		t.Errorf("content = %q", content)
	}
}

func TestParseSkillsUploadDataRejectsMissingOrOversizeContent(t *testing.T) {
	for _, payload := range []json.RawMessage{
		json.RawMessage(`{"content":"  "}`),
		json.RawMessage(`{"content":`),
		json.RawMessage(fmt.Sprintf(`{"content":%q}`, strings.Repeat("a", int(skills.StoreMaxBytes)+1))),
	} {
		if content, errMsg := parseSkillsUploadData(payload); errMsg == "" || content != "" {
			t.Errorf("parseSkillsUploadData(%s) = %q, %q; want failure", payload, content, errMsg)
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
