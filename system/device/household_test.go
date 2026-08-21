package device

import (
	"testing"

	"go.autonomous.ai/os/system/server/config"
)

func TestHouseholdClaimAndRoles(t *testing.T) {
	t.Chdir(t.TempDir())
	s := &Service{config: &config.Config{}}
	pub := s.ensureHousehold()
	if pub.SetupPIN == "" {
		t.Fatal("expected setup pin")
	}
	got, err := s.Claim(ClaimRequest{PIN: pub.SetupPIN, Name: "Chris", Room: "desk"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Claimed || got.Room != "desk" {
		t.Fatalf("%+v", got)
	}
	var owner bool
	for _, m := range got.Members {
		if m.Label == "chris" && m.Role == config.RoleOwner {
			owner = true
		}
	}
	if !owner {
		t.Fatalf("members = %+v", got.Members)
	}
	if _, err := s.Claim(ClaimRequest{PIN: "00000000", Name: "Sam"}); err == nil {
		t.Fatal("second claim without invite must fail")
	}
	inv, err := s.StartInvite(config.RoleKid)
	if err != nil {
		t.Fatal(err)
	}
	if inv.InviteCode == "" || inv.InviteRole != config.RoleKid {
		t.Fatalf("invite = %+v", inv)
	}
	joined, err := s.Claim(ClaimRequest{Code: inv.InviteCode, Name: "Sam", Role: config.RoleKid})
	if err != nil {
		t.Fatal(err)
	}
	if joined.RoleOfMissing() {
		t.Fatal("expected sam")
	}
	if s.Household().RoleOf("sam") != config.RoleKid {
		t.Fatalf("sam role = %q", s.Household().RoleOf("sam"))
	}
}

func (h HouseholdPublic) RoleOfMissing() bool {
	for _, m := range h.Members {
		if m.Label == "sam" {
			return false
		}
	}
	return true
}

func TestNormalizeRole(t *testing.T) {
	if _, err := config.NormalizeRole("stranger"); err == nil {
		t.Fatal("expected error")
	}
	r, err := config.NormalizeRole("Kid")
	if err != nil || r != config.RoleKid {
		t.Fatalf("%q %v", r, err)
	}
}
