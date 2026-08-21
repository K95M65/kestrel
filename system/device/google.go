package device

import (
	"fmt"
	"os"
	"strings"
	"time"

	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/googleauth"
	"go.autonomous.ai/os/system/lib/usercanon"
	"go.autonomous.ai/os/system/server/config"
)

// GoogleStatus is the secret-free Sign in with Google row.
type GoogleStatus struct {
	Ready     bool   `json:"ready"`
	Connected bool   `json:"connected"`
	UserEmail string `json:"user_email,omitempty"`
	HasClient bool   `json:"has_client"`
	HasSecret bool   `json:"has_secret"`
	AuthType  string `json:"auth_type,omitempty"`
}

func (s *Service) googleClient() *googleauth.Client {
	return s.GoogleOAuthClient()
}

// GoogleOAuthClient is the live Google device-login client, or nil.
func (s *Service) GoogleOAuthClient() *googleauth.Client {
	id, secret := s.googleClientCreds()
	if id == "" {
		return nil
	}
	return googleauth.New(id, secret)
}

func (s *Service) googleClientCreds() (id, secret string) {
	if s == nil || s.config == nil {
		return "", ""
	}
	id = strings.TrimSpace(s.config.GoogleOAuthClientID)
	secret = strings.TrimSpace(s.config.GoogleOAuthClientSecret)
	if env := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_ID")); env != "" {
		id = env
	}
	if env := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET")); env != "" {
		secret = env
	}
	return id, secret
}

func (s *Service) GoogleStatus() GoogleStatus {
	id, secret := s.googleClientCreds()
	st := GoogleStatus{HasClient: id != "", HasSecret: secret != "", Ready: id != ""}
	gmail, _, _ := s.readConnector("gmail")
	cal, _, _ := s.readConnector("google_calendar")
	if gmail.AuthType == "oauth" && gmail.AccessToken != "" {
		st.Connected = true
		st.AuthType = "oauth"
		st.UserEmail = gmail.UserEmail
	}
	if !st.Connected && cal.AuthType == "oauth" && cal.AccessToken != "" {
		st.Connected = true
		st.AuthType = "oauth"
		st.UserEmail = cal.UserEmail
	}
	if st.UserEmail == "" {
		st.UserEmail = s.Household().OwnerEmail
	}
	return st
}

// SetGoogleOAuthClient stores the operator's Google Cloud TV/limited-input client.
func (s *Service) SetGoogleOAuthClient(clientID, clientSecret string) error {
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	return s.config.WithLockSave(func(c *config.Config) {
		if clientID != "" {
			c.GoogleOAuthClientID = clientID
		}
		if clientSecret != "" {
			c.GoogleOAuthClientSecret = clientSecret
		}
	})
}

// ApplyGoogleTokens writes Gmail + Calendar as one Google account and claims
// the household owner when still unclaimed.
func (s *Service) ApplyGoogleTokens(tok googleauth.Tokens, info googleauth.UserInfo) error {
	if strings.TrimSpace(tok.AccessToken) == "" {
		return fmt.Errorf("Google access token is empty")
	}
	expires := time.Now().Add(tok.ExpiresIn).Unix()
	if tok.ExpiresIn <= 0 {
		expires = time.Now().Add(time.Hour).Unix()
	}
	entry := domain.ConnectorEntry{
		AuthType:     "oauth",
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
		ExpiresAt:    expires,
		Scopes:       strings.Fields(tok.Scope),
		UserEmail:    info.Email,
		Refresh:      tok.RefreshToken != "",
		ObtainedAt:   time.Now().Unix(),
	}
	if len(entry.Scopes) == 0 {
		entry.Scopes = strings.Fields(googleauth.Scope)
	}
	for _, code := range []string{"gmail", "google_calendar"} {
		path, err := s.connectorPath(code)
		if err != nil {
			return err
		}
		file, err := loadConnectorsFile(path)
		if err != nil {
			return err
		}
		file.Connectors[code] = entry
		if err := writeConnectorsFile(path, file); err != nil {
			return err
		}
	}
	if info.Email != "" {
		_ = s.SetOwnerEmail(info.Email)
	}
	h := s.ensureHousehold()
	if !h.Claimed && strings.TrimSpace(info.Name) != "" {
		_, _ = s.Claim(ClaimRequest{
			PIN:   h.SetupPIN,
			Name:  usercanon.Slugify(info.Name),
			Email: info.Email,
			Role:  config.RoleOwner,
		})
	}
	return nil
}
