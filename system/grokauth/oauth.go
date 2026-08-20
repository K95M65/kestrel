// Package grokauth is the xAI Grok account login used by Kestrel.
//
// It is a Go port of OpenCode's official xAI plugin
// (packages/opencode/src/plugin/xai.ts): the public Grok-CLI OAuth client,
// RFC 8628 device-code grant for headless robots, and refresh_token rotation.
// Access tokens are accepted by https://api.x.ai/v1 as Bearer credentials, so
// OpenClaw can run as a BYO OpenAI-compatible brain without an xai- console key.
package grokauth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	errDeviceDenied  = errors.New("xAI device authorization was denied")
	errDeviceExpired = errors.New("xAI device code expired — please re-run login")
)

// TerminalDeviceError reports a device-code outcome that should not be retried.
func TerminalDeviceError(err error) bool {
	return errors.Is(err, errDeviceDenied) || errors.Is(err, errDeviceExpired)
}

const (
	// ClientID is the public desktop OAuth client used by the Grok CLI and by
	// OpenCode's SuperGrok login. It is not a secret.
	ClientID = "b1a00492-073a-47ea-816f-4c329264a828"

	TokenURL               = "https://auth.x.ai/oauth2/token"
	DeviceAuthorizationURL = "https://auth.x.ai/oauth2/device/code"
	DeviceCodeGrantType    = "urn:ietf:params:oauth:grant-type:device_code"
	Scope                  = "openid profile email offline_access grok-cli:access api:access"
	APIBaseURL             = "https://api.x.ai/v1"
	DefaultModel           = "grok-4.6"
	Referrer               = "autonomous-os"
	UserAgent              = "autonomous-os-grokauth/1"

	AccessTokenRefreshSkew    = 2 * time.Minute
	DeviceCodeDefaultInterval = 5 * time.Second
	DeviceCodeMinInterval     = 1 * time.Second
	DeviceCodeSlowDownStep    = 5 * time.Second
	DeviceCodeDefaultExpires  = 5 * time.Minute
	DeviceCodePollSafety      = 3 * time.Second
)

// Tokens is one xAI OAuth token set. Refresh tokens rotate; persist the
// returned RefreshToken after every successful Refresh.
type Tokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
	TokenType    string
	Scope        string
}

// DeviceCode is the RFC 8628 authorization payload shown to the operator.
type DeviceCode struct {
	DeviceCode              string
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresIn               time.Duration
	Interval                time.Duration
}

// Client talks to xAI's OAuth endpoints. Tests inject a custom HTTPDoer.
type Client struct {
	HTTP      HTTPDoer
	TokenURL  string
	DeviceURL string
	Referrer  string
	Now       func() time.Time
	Sleep     func(time.Duration)
}

// HTTPDoer is the subset of http.Client used here.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// New returns a Client pointed at the live xAI endpoints.
func New() *Client {
	return &Client{
		HTTP:      &http.Client{Timeout: 30 * time.Second},
		TokenURL:  TokenURL,
		DeviceURL: DeviceAuthorizationURL,
		Referrer:  Referrer,
		Now:       time.Now,
		Sleep:     time.Sleep,
	}
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

func (c *Client) deviceURL() string {
	if c != nil && c.DeviceURL != "" {
		return c.DeviceURL
	}
	return DeviceAuthorizationURL
}

func (c *Client) referrer() string {
	if c != nil && c.Referrer != "" {
		return c.Referrer
	}
	return Referrer
}

func (c *Client) now() time.Time {
	if c != nil && c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Client) sleep(d time.Duration) {
	if c != nil && c.Sleep != nil {
		c.Sleep(d)
		return
	}
	time.Sleep(d)
}

// Refresh exchanges a refresh token for a new access token. xAI rotates the
// refresh token; callers must persist Tokens.RefreshToken.
func (c *Client) Refresh(refreshToken string) (Tokens, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return Tokens{}, fmt.Errorf("xAI refresh token is empty")
	}
	return c.postToken(url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {ClientID},
	}, refreshToken)
}

// RequestDeviceCode starts an RFC 8628 device-code login. Show UserCode and
// VerificationURI (or VerificationURIComplete) to the operator, then Poll.
func (c *Client) RequestDeviceCode() (DeviceCode, error) {
	form := url.Values{
		"client_id": {ClientID},
		"scope":     {Scope},
		"referrer":  {c.referrer()},
	}
	body, err := c.postForm(c.deviceURL(), form)
	if err != nil {
		return DeviceCode{}, fmt.Errorf("xAI device code request: %w", err)
	}
	var raw struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               any    `json:"expires_in"`
		Interval                any    `json:"interval"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return DeviceCode{}, fmt.Errorf("xAI device code response: %w", err)
	}
	if raw.DeviceCode == "" || raw.UserCode == "" || raw.VerificationURI == "" {
		return DeviceCode{}, fmt.Errorf("xAI device code response is missing device_code / user_code / verification_uri")
	}
	return DeviceCode{
		DeviceCode:              raw.DeviceCode,
		UserCode:                raw.UserCode,
		VerificationURI:         raw.VerificationURI,
		VerificationURIComplete: raw.VerificationURIComplete,
		ExpiresIn:               positiveSeconds(raw.ExpiresIn, DeviceCodeDefaultExpires),
		Interval:                positiveSeconds(raw.Interval, DeviceCodeDefaultInterval),
	}, nil
}

// Exchange tries one token request. pending is the RFC 8628 error name
// ("authorization_pending", "slow_down") while the operator has not finished.
func (c *Client) Exchange(dc DeviceCode) (tok Tokens, pending string, err error) {
	if dc.DeviceCode == "" {
		return Tokens{}, "", fmt.Errorf("xAI device_code is empty")
	}
	body, status, err := c.postFormStatus(c.tokenURL(), url.Values{
		"grant_type":  {DeviceCodeGrantType},
		"client_id":   {ClientID},
		"device_code": {dc.DeviceCode},
	})
	if err != nil {
		return Tokens{}, "", err
	}
	if status >= 200 && status < 300 {
		tok, err = parseTokens(body, "")
		return tok, "", err
	}
	var terr struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	_ = json.Unmarshal(body, &terr)
	switch terr.Error {
	case "authorization_pending", "slow_down":
		return Tokens{}, terr.Error, nil
	case "access_denied", "authorization_denied":
		return Tokens{}, "", errDeviceDenied
	case "expired_token":
		return Tokens{}, "", errDeviceExpired
	default:
		detail := terr.ErrorDescription
		if detail == "" {
			detail = terr.Error
		}
		if detail == "" {
			detail = string(body)
		}
		return Tokens{}, "", fmt.Errorf("xAI device token exchange failed (%d): %s", status, detail)
	}
}

// Poll waits until the operator finishes the device-code login or the code expires.
func (c *Client) Poll(dc DeviceCode) (Tokens, error) {
	deadline := c.now().Add(dc.ExpiresIn)
	if dc.ExpiresIn <= 0 {
		deadline = c.now().Add(DeviceCodeDefaultExpires)
	}
	interval := dc.Interval
	if interval < DeviceCodeMinInterval {
		interval = DeviceCodeMinInterval
	}
	for c.now().Before(deadline) {
		tok, pending, err := c.Exchange(dc)
		if err != nil {
			return Tokens{}, err
		}
		if pending == "" {
			return tok, nil
		}
		if pending == "slow_down" {
			interval += DeviceCodeSlowDownStep
		}
		remaining := deadline.Sub(c.now())
		if remaining < 0 {
			break
		}
		c.sleep(minDuration(interval+DeviceCodePollSafety, remaining))
	}
	return Tokens{}, fmt.Errorf("xAI device authorization timed out")
}

// AccessTokenIsExpiring reports whether a JWT access token is inside the
// refresh skew window. Opaque tokens return false so the 401 path drives refresh.
func AccessTokenIsExpiring(token string, skew time.Duration, now time.Time) bool {
	token = strings.TrimSpace(token)
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return false
	}
	payload := parts[1]
	payload = strings.ReplaceAll(payload, "-", "+")
	payload = strings.ReplaceAll(payload, "_", "/")
	for len(payload)%4 != 0 {
		payload += "="
	}
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil || claims.Exp == 0 {
		return false
	}
	if skew < 0 {
		skew = 0
	}
	return time.Unix(claims.Exp, 0).Before(now.Add(skew))
}

func (c *Client) postToken(form url.Values, fallbackRefresh string) (Tokens, error) {
	body, err := c.postForm(c.tokenURL(), form)
	if err != nil {
		return Tokens{}, err
	}
	return parseTokens(body, fallbackRefresh)
}

func (c *Client) postForm(endpoint string, form url.Values) ([]byte, error) {
	body, status, err := c.postFormStatus(endpoint, form)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("xAI OAuth HTTP %d: %s", status, truncate(string(body), 300))
	}
	return body, nil
}

func (c *Client) postFormStatus(endpoint string, form url.Values) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func parseTokens(body []byte, fallbackRefresh string) (Tokens, error) {
	var raw struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
		ExpiresIn    any    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return Tokens{}, fmt.Errorf("xAI token response: %w", err)
	}
	access := strings.TrimSpace(raw.AccessToken)
	refresh := strings.TrimSpace(raw.RefreshToken)
	if refresh == "" {
		refresh = strings.TrimSpace(fallbackRefresh)
	}
	if access == "" {
		return Tokens{}, fmt.Errorf("xAI token response did not include access_token")
	}
	if refresh == "" {
		return Tokens{}, fmt.Errorf("xAI token response did not include refresh_token")
	}
	tok := Tokens{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    positiveSeconds(raw.ExpiresIn, time.Hour),
		TokenType:    raw.TokenType,
		Scope:        raw.Scope,
	}
	if tok.TokenType == "" {
		tok.TokenType = "Bearer"
	}
	return tok, nil
}

func positiveSeconds(v any, fallback time.Duration) time.Duration {
	switch n := v.(type) {
	case nil:
		return fallback
	case float64:
		if n > 0 {
			return time.Duration(n * float64(time.Second))
		}
	case json.Number:
		f, err := n.Float64()
		if err == nil && f > 0 {
			return time.Duration(f * float64(time.Second))
		}
	case string:
		f, err := strconv.ParseFloat(n, 64)
		if err == nil && f > 0 {
			return time.Duration(f * float64(time.Second))
		}
	}
	return fallback
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
