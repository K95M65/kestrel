package config

import (
	"fmt"
	"strings"

	"go.autonomous.ai/os/system/lib/usercanon"
)

// Household roles. Owner is the account that claimed the body. Family may
// talk and use connectors. Kid is bound like the kids pack (no mail,
// calendar, or computer-use). Guest is greet-and-chat only.
const (
	RoleOwner  = "owner"
	RoleFamily = "family"
	RoleKid    = "kid"
	RoleGuest  = "guest"
)

// Household is who this body belongs to. LAN-only: no Apple/Google Home.
type Household struct {
	Claimed    bool              `json:"claimed,omitempty"`
	Room       string            `json:"room,omitempty"`
	OwnerEmail string            `json:"owner_email,omitempty"`
	SetupPIN   string            `json:"setup_pin,omitempty"`
	Members    []HouseholdMember `json:"members,omitempty"`
}

// HouseholdMember is one enrolled person plus a role.
type HouseholdMember struct {
	Label string `json:"label"`
	Role  string `json:"role"`
}

// NormalizeRole returns a known role or an error.
func NormalizeRole(raw string) (string, error) {
	r := strings.ToLower(strings.TrimSpace(raw))
	switch r {
	case RoleOwner, RoleFamily, RoleKid, RoleGuest:
		return r, nil
	case "":
		return RoleFamily, nil
	default:
		return "", fmt.Errorf("role must be owner, family, kid, or guest")
	}
}

// NormalizeMemberLabel is the People slug.
func NormalizeMemberLabel(raw string) (string, error) {
	slug := usercanon.Slugify(raw)
	if slug == "" || slug == usercanon.DefaultUser {
		return "", fmt.Errorf("pick a person")
	}
	return slug, nil
}

func (h Household) Member(label string) (HouseholdMember, bool) {
	slug := usercanon.Slugify(label)
	for _, m := range h.Members {
		if m.Label == slug {
			return m, true
		}
	}
	return HouseholdMember{}, false
}

func (h Household) RoleOf(label string) string {
	if m, ok := h.Member(label); ok {
		return m.Role
	}
	return ""
}

func (h *Household) UpsertMember(label, role string) error {
	slug, err := NormalizeMemberLabel(label)
	if err != nil {
		return err
	}
	r, err := NormalizeRole(role)
	if err != nil {
		return err
	}
	if h.Members == nil {
		h.Members = []HouseholdMember{}
	}
	for i, m := range h.Members {
		if m.Label == slug {
			h.Members[i].Role = r
			return nil
		}
	}
	h.Members = append(h.Members, HouseholdMember{Label: slug, Role: r})
	return nil
}

func (h *Household) RemoveMember(label string) {
	slug := usercanon.Slugify(label)
	out := h.Members[:0]
	for _, m := range h.Members {
		if m.Label != slug {
			out = append(out, m)
		}
	}
	h.Members = out
}

// DemoteOwners sets every owner except keep to family. keep may be empty.
func (h *Household) DemoteOwners(keep string) {
	keep = usercanon.Slugify(keep)
	for i, m := range h.Members {
		if m.Role == RoleOwner && m.Label != keep {
			h.Members[i].Role = RoleFamily
		}
	}
}
