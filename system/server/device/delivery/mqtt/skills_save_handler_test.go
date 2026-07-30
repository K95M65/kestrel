package mqtthandler

import (
	"encoding/json"
	"fmt"
	"testing"

	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/skills"
)

func TestParseSkillsSaveData(t *testing.T) {
	// Whitespace must be trimmed before it reaches the gateway — a name with a
	// trailing space would otherwise fail ValidateSkillName for no good reason.
	draft, errMsg := parseSkillsSaveData(json.RawMessage(
		`{"name":"  weekly-report  ","description":" Sums up the week. ","instructions":"\n1. Collect\n"}`))
	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	want := domain.SkillDraft{
		Name:         "weekly-report",
		Description:  "Sums up the week.",
		Instructions: "1. Collect",
	}
	if draft != want {
		t.Fatalf("draft = %+v, want %+v", draft, want)
	}
}

func TestParseSkillsSaveDataRejectsIncomplete(t *testing.T) {
	cases := []struct{ label, payload string }{
		{"malformed json", `{"name":`},
		{"empty object", `{}`},
		{"missing description", `{"name":"x","instructions":"i"}`},
		{"missing instructions", `{"name":"x","description":"d"}`},
		{"missing name", `{"description":"d","instructions":"i"}`},
		{"whitespace-only name", `{"name":"   ","description":"d","instructions":"i"}`},
		{"whitespace-only instructions", `{"name":"x","description":"d","instructions":" \n\t "}`},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			draft, errMsg := parseSkillsSaveData(json.RawMessage(tc.payload))
			if errMsg == "" {
				t.Fatalf("expected a failure message, got draft %+v", draft)
			}
			if draft != (domain.SkillDraft{}) {
				t.Errorf("a rejected payload must not yield a draft: %+v", draft)
			}
		})
	}
}

// The name SHAPE is deliberately NOT checked here — SaveSkill owns that via
// skills.ValidateSkillName, so the HTTP and MQTT paths can't drift on what a
// legal name is. Parsing accepts it; the gateway is what rejects it.
func TestParseSkillsSaveDataDefersNameShapeToGateway(t *testing.T) {
	draft, errMsg := parseSkillsSaveData(json.RawMessage(
		`{"name":"NOT a slug","description":"d","instructions":"i"}`))
	if errMsg != "" {
		t.Fatalf("parsing must not police the name shape, got: %s", errMsg)
	}
	if draft.Name != "NOT a slug" {
		t.Errorf("name = %q, want it passed through verbatim", draft.Name)
	}
	// And the shared validator is what says no.
	if err := skills.ValidateSkillName(draft.Name); err == nil {
		t.Error("skills.ValidateSkillName should reject this name")
	}
}

func TestClassifySkillsSaveError(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{domain.ErrNotSupportedByRuntime, "unsupported_runtime"},
		{skills.ErrInvalidSkillName, "validate_name"},
		{skills.ErrSkillExists, "already_exists"},
		{fmt.Errorf("disk full"), "write"},
		// Wrapped sentinels must still classify — SaveSkill wraps with %w.
		{fmt.Errorf("save: %w", skills.ErrSkillExists), "already_exists"},
		{fmt.Errorf("gateway: %w", domain.ErrNotSupportedByRuntime), "unsupported_runtime"},
	}
	for _, tc := range cases {
		if got := classifySkillsSaveError(tc.err); got != tc.want {
			t.Errorf("classify(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

// skills.save must be its own kind — reusing skills.install would conflate
// "write this one authored skill" with "fetch a whole role bundle from the CDN".
func TestSkillsSaveKindIsDistinct(t *testing.T) {
	if domain.KindSkillsSave == domain.KindSkillsInstall {
		t.Fatal("skills.save and skills.install must be distinct kinds")
	}
	if domain.KindSkillsSave != "skills.save" {
		t.Errorf("kind = %q, want skills.save", domain.KindSkillsSave)
	}
}
