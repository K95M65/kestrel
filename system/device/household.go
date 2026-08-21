package device

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"go.autonomous.ai/os/system/server/config"
	"go.autonomous.ai/os/system/skillcontext"
)

const (
	householdInviteTTL = 10 * time.Minute
	setupPINLen        = 8
)

type pendingInvite struct {
	code      string
	role      string
	expiresAt time.Time
}

func (s *Service) initHousehold() {
	skillcontext.SetHouseholdSource(func() config.Household {
		if s == nil || s.config == nil {
			return config.Household{}
		}
		return s.Household()
	})
}

// Household is the persisted household, with a setup PIN minted if unclaimed.
func (s *Service) Household() config.Household {
	if s == nil || s.config == nil || s.config.Household == nil {
		return config.Household{}
	}
	h := *s.config.Household
	if h.Members != nil {
		h.Members = append([]config.HouseholdMember(nil), h.Members...)
	}
	return h
}

// GetHouseholdPublic is the secret-free row for Home / claim.
type HouseholdPublic struct {
	Claimed    bool                     `json:"claimed"`
	Room       string                   `json:"room,omitempty"`
	OwnerEmail string                   `json:"owner_email,omitempty"`
	SetupPIN   string                   `json:"setup_pin,omitempty"`
	Members    []config.HouseholdMember `json:"members,omitempty"`
	InviteCode string                   `json:"invite_code,omitempty"`
	InviteRole string                   `json:"invite_role,omitempty"`
	InviteTTL  int                      `json:"invite_ttl,omitempty"`
}

func (s *Service) GetHouseholdPublic(includePIN bool) HouseholdPublic {
	h := s.ensureHousehold()
	out := HouseholdPublic{
		Claimed:    h.Claimed,
		Room:       h.Room,
		OwnerEmail: h.OwnerEmail,
		Members:    append([]config.HouseholdMember(nil), h.Members...),
	}
	if includePIN && !h.Claimed {
		out.SetupPIN = h.SetupPIN
	}
	s.inviteMu.Lock()
	inv := s.invite
	s.inviteMu.Unlock()
	if inv != nil && time.Now().Before(inv.expiresAt) {
		out.InviteCode = inv.code
		out.InviteRole = inv.role
		out.InviteTTL = int(time.Until(inv.expiresAt).Seconds())
	}
	return out
}

func (s *Service) ensureHousehold() config.Household {
	h := s.Household()
	if h.Claimed || strings.TrimSpace(h.SetupPIN) != "" {
		s.initHousehold()
		return h
	}
	pin := newSetupPIN()
	_ = s.config.WithLockSave(func(c *config.Config) {
		if c.Household == nil {
			c.Household = &config.Household{}
		}
		if c.Household.Claimed || c.Household.SetupPIN != "" {
			return
		}
		c.Household.SetupPIN = pin
	})
	s.initHousehold()
	return s.Household()
}

// ClaimRequest is the LAN claim form.
type ClaimRequest struct {
	PIN   string `json:"pin"`
	Code  string `json:"code"`
	Name  string `json:"name"`
	Room  string `json:"room"`
	Role  string `json:"role"`
	Email string `json:"email"`
}

func (s *Service) Claim(in ClaimRequest) (HouseholdPublic, error) {
	name, err := config.NormalizeMemberLabel(in.Name)
	if err != nil {
		return HouseholdPublic{}, err
	}
	room := strings.TrimSpace(in.Room)
	if len(room) > 40 {
		room = room[:40]
	}
	h := s.ensureHousehold()
	if !h.Claimed {
		pin := strings.TrimSpace(in.PIN)
		if pin == "" {
			pin = strings.TrimSpace(in.Code)
		}
		if pin == "" || pin != h.SetupPIN {
			return HouseholdPublic{}, fmt.Errorf("wrong setup code")
		}
		email := strings.TrimSpace(in.Email)
		if err := s.config.WithLockSave(func(c *config.Config) {
			if c.Household == nil {
				c.Household = &config.Household{}
			}
			c.Household.Claimed = true
			c.Household.Room = room
			c.Household.OwnerEmail = email
			c.Household.SetupPIN = ""
			c.Household.Members = []config.HouseholdMember{{Label: name, Role: config.RoleOwner}}
			if c.Behaviors == nil {
				d := config.DefaultBehaviors()
				c.Behaviors = &d
			}
			c.Behaviors.Me = name
		}); err != nil {
			return HouseholdPublic{}, err
		}
		s.initHousehold()
		return s.GetHouseholdPublic(true), nil
	}
	// Already claimed: consume a family invite.
	s.inviteMu.Lock()
	inv := s.invite
	ok := inv != nil && time.Now().Before(inv.expiresAt) && (in.Code == inv.code || in.PIN == inv.code)
	role := config.RoleFamily
	if ok {
		role = inv.role
		s.invite = nil
	}
	s.inviteMu.Unlock()
	if !ok {
		return HouseholdPublic{}, fmt.Errorf("this robot is already claimed — ask the owner for an invite code")
	}
	if strings.TrimSpace(in.Role) != "" {
		if r, err := config.NormalizeRole(in.Role); err == nil {
			role = r
		}
	}
	if role == config.RoleOwner {
		role = config.RoleFamily
	}
	if err := s.SetMemberRole(name, role); err != nil {
		return HouseholdPublic{}, err
	}
	if room != "" {
		_ = s.config.WithLockSave(func(c *config.Config) {
			if c.Household != nil && c.Household.Room == "" {
				c.Household.Room = room
			}
		})
	}
	s.initHousehold()
	return s.GetHouseholdPublic(false), nil
}

// StartInvite issues a short-lived code so a family/kid/guest can join.
func (s *Service) StartInvite(role string) (HouseholdPublic, error) {
	h := s.ensureHousehold()
	if !h.Claimed {
		return HouseholdPublic{}, fmt.Errorf("claim this robot first")
	}
	r, err := config.NormalizeRole(role)
	if err != nil {
		return HouseholdPublic{}, err
	}
	if r == config.RoleOwner {
		r = config.RoleFamily
	}
	code := newSixDigit()
	s.inviteMu.Lock()
	s.invite = &pendingInvite{code: code, role: r, expiresAt: time.Now().Add(householdInviteTTL)}
	s.inviteMu.Unlock()
	return s.GetHouseholdPublic(false), nil
}

func (s *Service) SetMemberRole(label, role string) error {
	slug, err := config.NormalizeMemberLabel(label)
	if err != nil {
		return err
	}
	r, err := config.NormalizeRole(role)
	if err != nil {
		return err
	}
	if err := s.config.WithLockSave(func(c *config.Config) {
		if c.Household == nil {
			c.Household = &config.Household{Claimed: true}
		}
		if r == config.RoleOwner {
			c.Household.DemoteOwners(slug)
			if c.Behaviors == nil {
				d := config.DefaultBehaviors()
				c.Behaviors = &d
			}
			c.Behaviors.Me = slug
		}
		_ = c.Household.UpsertMember(slug, r)
	}); err != nil {
		return err
	}
	s.initHousehold()
	return nil
}

func (s *Service) SetHouseholdRoom(room string) error {
	room = strings.TrimSpace(room)
	if len(room) > 40 {
		room = room[:40]
	}
	if err := s.config.WithLockSave(func(c *config.Config) {
		if c.Household == nil {
			c.Household = &config.Household{}
		}
		c.Household.Room = room
	}); err != nil {
		return err
	}
	s.initHousehold()
	return nil
}

func (s *Service) SetOwnerEmail(email string) error {
	email = strings.TrimSpace(email)
	if err := s.config.WithLockSave(func(c *config.Config) {
		if c.Household == nil {
			c.Household = &config.Household{}
		}
		c.Household.OwnerEmail = email
		if !c.Household.Claimed && email != "" {
			c.Household.Claimed = true
			c.Household.SetupPIN = ""
		}
	}); err != nil {
		return err
	}
	s.initHousehold()
	return nil
}

func newSetupPIN() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	n := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	return fmt.Sprintf("%08d", n%90000000+10000000)
}

func newSixDigit() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	n := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	return fmt.Sprintf("%06d", n%900000+100000)
}
