package device

import (
	"fmt"
	"strings"

	"go.autonomous.ai/os/system/appleauth"
	"go.autonomous.ai/os/system/lib/usercanon"
	"go.autonomous.ai/os/system/server/config"
)

// AppleStatus is the secret-free Sign in with Apple row.
type AppleStatus struct {
	Ready   bool   `json:"ready"`
	Hint    string `json:"hint"`
	HasID   bool   `json:"has_services_id"`
	HasKey  bool   `json:"has_key"`
	Return  string `json:"return_url,omitempty"`
	HTTPS   bool   `json:"https"`
	Email   string `json:"user_email,omitempty"`
}

func (s *Service) AppleOAuthClient() *appleauth.Client {
	if s == nil || s.config == nil {
		return nil
	}
	id := strings.TrimSpace(s.config.AppleServicesID)
	if id == "" {
		return nil
	}
	return appleauth.New(id, s.config.AppleTeamID, s.config.AppleKeyID, s.config.ApplePrivateKey, s.config.AppleReturnURL)
}

func (s *Service) AppleStatus() AppleStatus {
	st := AppleStatus{
		Hint: "Sign in with Apple needs an Apple Developer Services ID and a HTTPS return URL. A LAN http:// robot cannot complete Apple's callback. Use Google sign-in on the desk, or put a tunnel in front and paste the https:// callback.",
	}
	if s == nil || s.config == nil {
		return st
	}
	st.HasID = strings.TrimSpace(s.config.AppleServicesID) != ""
	st.HasKey = strings.TrimSpace(s.config.ApplePrivateKey) != ""
	st.Return = strings.TrimSpace(s.config.AppleReturnURL)
	st.HTTPS = strings.HasPrefix(st.Return, "https://")
	st.Email = s.Household().OwnerEmail
	cli := s.AppleOAuthClient()
	if cli != nil && cli.Ready() == nil {
		st.Ready = true
		st.Hint = "Tap Sign in with Apple. Apple opens in a new tab; this page waits for the callback."
	} else if st.HasID && !st.HTTPS {
		st.Hint = "Return URL must start with https://. cloudflared tunnel or a domain in front of this robot, then paste that URL here as …/api/auth/apple/callback."
	}
	return st
}

func (s *Service) SetAppleClient(servicesID, teamID, keyID, privateKey, returnURL string) error {
	return s.config.WithLockSave(func(c *config.Config) {
		if v := strings.TrimSpace(servicesID); v != "" {
			c.AppleServicesID = v
		}
		if v := strings.TrimSpace(teamID); v != "" {
			c.AppleTeamID = v
		}
		if v := strings.TrimSpace(keyID); v != "" {
			c.AppleKeyID = v
		}
		if v := strings.TrimSpace(privateKey); v != "" {
			c.ApplePrivateKey = v
		}
		if v := strings.TrimSpace(returnURL); v != "" {
			c.AppleReturnURL = v
		}
	})
}

// ApplyAppleIdentity claims the household owner from Sign in with Apple.
func (s *Service) ApplyAppleIdentity(id appleauth.Identity) error {
	email := strings.TrimSpace(id.Email)
	if email == "" && strings.TrimSpace(id.Sub) == "" {
		return fmt.Errorf("Apple did not return an email or subject")
	}
	if email != "" {
		_ = s.SetOwnerEmail(email)
	}
	h := s.ensureHousehold()
	name := strings.TrimSpace(id.Name)
	if name == "" && email != "" {
		name = strings.Split(email, "@")[0]
	}
	if !h.Claimed && name != "" {
		_, _ = s.Claim(ClaimRequest{
			PIN:   h.SetupPIN,
			Name:  usercanon.Slugify(name),
			Email: email,
			Role:  config.RoleOwner,
		})
	}
	return nil
}
