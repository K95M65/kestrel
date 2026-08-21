// Package appleauth is Sign in with Apple (authorization-code + JWT client secret).
//
// Apple will not complete this on a LAN http:// robot. The return URL must be
// HTTPS (tunnel or domain) and registered on the Services ID.
package appleauth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	AuthorizeURL = "https://appleid.apple.com/auth/authorize"
	TokenURL     = "https://appleid.apple.com/auth/token"
	Audience     = "https://appleid.apple.com"
	Scope        = "name email"
	UserAgent    = "kestrel-appleauth/1"
)

// Client talks to Apple's token endpoint. Tests inject HTTP.
type Client struct {
	HTTP       HTTPDoer
	TokenURL   string
	ServicesID string
	TeamID     string
	KeyID      string
	PrivateKey string
	ReturnURL  string
	Now        func() time.Time
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func New(servicesID, teamID, keyID, privateKey, returnURL string) *Client {
	return &Client{
		HTTP:       &http.Client{Timeout: 30 * time.Second},
		TokenURL:   TokenURL,
		ServicesID: strings.TrimSpace(servicesID),
		TeamID:     strings.TrimSpace(teamID),
		KeyID:      strings.TrimSpace(keyID),
		PrivateKey: strings.TrimSpace(privateKey),
		ReturnURL:  strings.TrimSpace(returnURL),
		Now:        time.Now,
	}
}

func (c *Client) Ready() error {
	if c == nil || c.ServicesID == "" || c.TeamID == "" || c.KeyID == "" || c.PrivateKey == "" {
		return fmt.Errorf("Sign in with Apple needs a Services ID, Team ID, Key ID, and .p8 key")
	}
	if !strings.HasPrefix(c.ReturnURL, "https://") {
		return fmt.Errorf("Sign in with Apple needs an https:// return URL (a LAN http:// page cannot finish Apple's callback)")
	}
	return nil
}

// AuthorizeURL is the browser URL. State is an opaque CSRF token.
func (c *Client) AuthorizeURL(state string) (string, error) {
	if err := c.Ready(); err != nil {
		return "", err
	}
	q := url.Values{
		"client_id":     {c.ServicesID},
		"redirect_uri":  {c.ReturnURL},
		"response_type": {"code"},
		"response_mode": {"form_post"},
		"scope":         {Scope},
		"state":         {state},
	}
	return AuthorizeURL + "?" + q.Encode(), nil
}

type Identity struct {
	Sub   string
	Email string
	Name  string
}

// Exchange trades an authorization code for identity claims.
func (c *Client) Exchange(code string) (Identity, error) {
	if err := c.Ready(); err != nil {
		return Identity{}, err
	}
	secret, err := c.clientSecret()
	if err != nil {
		return Identity{}, err
	}
	form := url.Values{
		"client_id":     {c.ServicesID},
		"client_secret": {secret},
		"code":          {strings.TrimSpace(code)},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {c.ReturnURL},
	}
	req, err := http.NewRequest(http.MethodPost, c.tokenURL(), strings.NewReader(form.Encode()))
	if err != nil {
		return Identity{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UserAgent)
	resp, err := c.http().Do(req)
	if err != nil {
		return Identity{}, fmt.Errorf("Apple token: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Identity{}, fmt.Errorf("Apple token HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var tok struct {
		IDToken string `json:"id_token"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(raw, &tok); err != nil {
		return Identity{}, err
	}
	if tok.Error != "" {
		return Identity{}, fmt.Errorf("Apple token: %s", tok.Error)
	}
	return parseIDToken(tok.IDToken)
}

func (c *Client) http() HTTPDoer {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Client) tokenURL() string {
	if c != nil && c.TokenURL != "" {
		return c.TokenURL
	}
	return TokenURL
}

func (c *Client) now() time.Time {
	if c != nil && c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Client) clientSecret() (string, error) {
	key, err := parseP8(c.PrivateKey)
	if err != nil {
		return "", err
	}
	now := c.now().Unix()
	hdr, _ := json.Marshal(map[string]string{"alg": "ES256", "kid": c.KeyID})
	payload, _ := json.Marshal(map[string]any{
		"iss": c.TeamID,
		"iat": now,
		"exp": now + 15777000,
		"aud": Audience,
		"sub": c.ServicesID,
	})
	signing := b64(hdr) + "." + b64(payload)
	hash := sha256.Sum256([]byte(signing))
	r, s, err := ecdsa.Sign(rand.Reader, key, hash[:])
	if err != nil {
		return "", err
	}
	sig := append(padded(r), padded(s)...)
	return signing + "." + b64(sig), nil
}

func parseP8(pemBytes string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemBytes))
	if block == nil {
		return nil, fmt.Errorf("Apple .p8 key is not PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Apple .p8: %w", err)
	}
	ec, ok := key.(*ecdsa.PrivateKey)
	if !ok || ec.Curve != elliptic.P256() {
		return nil, fmt.Errorf("Apple .p8 must be P-256 ECDSA")
	}
	return ec, nil
}

func padded(n *big.Int) []byte {
	b := n.Bytes()
	if len(b) >= 32 {
		return b[len(b)-32:]
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

func b64(raw []byte) string {
	return base64.RawURLEncoding.EncodeToString(raw)
}

func parseIDToken(tok string) (Identity, error) {
	parts := strings.Split(tok, ".")
	if len(parts) < 2 {
		return Identity{}, fmt.Errorf("Apple id_token is malformed")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.URLEncoding.DecodeString(parts[1])
	}
	if err != nil {
		return Identity{}, fmt.Errorf("Apple id_token payload: %w", err)
	}
	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Identity{}, err
	}
	return Identity{Sub: claims.Sub, Email: claims.Email, Name: claims.Name}, nil
}

// NewState is a CSRF token for the Apple return.
func NewState() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return base64.RawURLEncoding.EncodeToString(b[:])
}
