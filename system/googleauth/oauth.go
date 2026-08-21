// Package googleauth is RFC 8628 device-code login for a Google account.
//
// Same shape as grokauth: the robot shows a code, the operator visits
// google.com/device, we poll for tokens. Client ID + secret come from
// config (a TV / limited-input OAuth client the operator created). There
// is no shipped Google Cloud project — without those fields Sign in with
// Google stays off and Gmail/Calendar keep the app-password / iCal path.
package googleauth

import (
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
	errDeviceDenied  = errors.New("Google device authorization was denied")
	errDeviceExpired = errors.New("Google device code expired — start again")
)

// TerminalDeviceError reports an outcome that should not be retried.
func TerminalDeviceError(err error) bool {
	return errors.Is(err, errDeviceDenied) || errors.Is(err, errDeviceExpired)
}

const (
	TokenURL               = "https://oauth2.googleapis.com/token"
	DeviceAuthorizationURL = "https://oauth2.googleapis.com/device/code"
	UserInfoURL            = "https://www.googleapis.com/oauth2/v3/userinfo"
	DeviceCodeGrantType    = "urn:ietf:params:oauth:grant-type:device_code"
	// Gmail modify + Calendar + Drive read + identity. Writes still honor
	// Behaviors ask-level / draft-not-send on the robot.
	Scope = "openid email profile https://www.googleapis.com/auth/gmail.modify https://www.googleapis.com/auth/calendar https://www.googleapis.com/auth/drive.readonly"

	UserAgent = "kestrel-googleauth/1"

	AccessTokenRefreshSkew    = 2 * time.Minute
	DeviceCodeDefaultInterval = 5 * time.Second
	DeviceCodeMinInterval     = 1 * time.Second
	DeviceCodeSlowDownStep    = 5 * time.Second
	DeviceCodeDefaultExpires  = 30 * time.Minute
	DeviceCodePollSafety      = 3 * time.Second
)

// Tokens is one Google OAuth token set.
type Tokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
	TokenType    string
	Scope        string
	IDToken      string
}

// DeviceCode is the RFC 8628 payload shown to the operator.
type DeviceCode struct {
	DeviceCode              string
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresIn               time.Duration
	Interval                time.Duration
}

// UserInfo is the secret-free identity of the signed-in Google account.
type UserInfo struct {
	Email string
	Name  string
	Sub   string
}

// HTTPDoer is the subset of http.Client used here.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Client talks to Google's OAuth endpoints. Tests inject HTTP.
type Client struct {
	HTTP        HTTPDoer
	TokenURL    string
	DeviceURL   string
	UserInfoURL string
	ClientID    string
	Secret      string
	Now         func() time.Time
	Sleep       func(time.Duration)
}

// New returns a Client for the given OAuth client. ID is required; secret is
// required for Google's TV/limited-input clients.
func New(clientID, secret string) *Client {
	return &Client{
		HTTP:        &http.Client{Timeout: 30 * time.Second},
		TokenURL:    TokenURL,
		DeviceURL:   DeviceAuthorizationURL,
		UserInfoURL: UserInfoURL,
		ClientID:    strings.TrimSpace(clientID),
		Secret:      strings.TrimSpace(secret),
		Now:         time.Now,
		Sleep:       time.Sleep,
	}
}

func (c *Client) ready() error {
	if c == nil || strings.TrimSpace(c.ClientID) == "" {
		return fmt.Errorf("Google OAuth client id is not set")
	}
	return nil
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

func (c *Client) userInfoURL() string {
	if c != nil && c.UserInfoURL != "" {
		return c.UserInfoURL
	}
	return UserInfoURL
}

// RequestDeviceCode starts an RFC 8628 device-code login.
func (c *Client) RequestDeviceCode() (DeviceCode, error) {
	if err := c.ready(); err != nil {
		return DeviceCode{}, err
	}
	form := url.Values{
		"client_id": {c.ClientID},
		"scope":     {Scope},
	}
	body, err := c.postForm(c.deviceURL(), form)
	if err != nil {
		return DeviceCode{}, fmt.Errorf("Google device code request: %w", err)
	}
	var raw struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURL         string `json:"verification_url"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               any    `json:"expires_in"`
		Interval                any    `json:"interval"`
		Error                   string `json:"error"`
		ErrorDescription        string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return DeviceCode{}, fmt.Errorf("Google device code response: %w", err)
	}
	if raw.Error != "" {
		detail := raw.ErrorDescription
		if detail == "" {
			detail = raw.Error
		}
		return DeviceCode{}, fmt.Errorf("Google device code: %s", detail)
	}
	uri := raw.VerificationURI
	if uri == "" {
		uri = raw.VerificationURL
	}
	if raw.DeviceCode == "" || raw.UserCode == "" || uri == "" {
		return DeviceCode{}, fmt.Errorf("Google device code response is missing device_code / user_code / verification_uri")
	}
	return DeviceCode{
		DeviceCode:              raw.DeviceCode,
		UserCode:                raw.UserCode,
		VerificationURI:         uri,
		VerificationURIComplete: raw.VerificationURIComplete,
		ExpiresIn:               positiveSeconds(raw.ExpiresIn, DeviceCodeDefaultExpires),
		Interval:                positiveSeconds(raw.Interval, DeviceCodeDefaultInterval),
	}, nil
}

// Exchange tries one token request. pending is the RFC 8628 error name while
// the operator has not finished.
func (c *Client) Exchange(dc DeviceCode) (tok Tokens, pending string, err error) {
	if err := c.ready(); err != nil {
		return Tokens{}, "", err
	}
	if dc.DeviceCode == "" {
		return Tokens{}, "", fmt.Errorf("Google device_code is empty")
	}
	form := url.Values{
		"grant_type":  {DeviceCodeGrantType},
		"client_id":   {c.ClientID},
		"device_code": {dc.DeviceCode},
	}
	if c.Secret != "" {
		form.Set("client_secret", c.Secret)
	}
	body, status, err := c.postFormStatus(c.tokenURL(), form)
	if err != nil {
		return Tokens{}, "", err
	}
	if status >= 200 && status < 300 {
		tok, err = parseTokens(body)
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
			detail = fmt.Sprintf("HTTP %d", status)
		}
		return Tokens{}, "", fmt.Errorf("Google token: %s", detail)
	}
}

// Refresh exchanges a refresh token for a new access token.
func (c *Client) Refresh(refreshToken string) (Tokens, error) {
	if err := c.ready(); err != nil {
		return Tokens{}, err
	}
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return Tokens{}, fmt.Errorf("Google refresh token is empty")
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {c.ClientID},
	}
	if c.Secret != "" {
		form.Set("client_secret", c.Secret)
	}
	body, status, err := c.postFormStatus(c.tokenURL(), form)
	if err != nil {
		return Tokens{}, err
	}
	if status < 200 || status >= 300 {
		return Tokens{}, fmt.Errorf("Google refresh failed (HTTP %d)", status)
	}
	return parseTokens(body)
}

// LookupUser fetches email/name for the access token.
func (c *Client) LookupUser(accessToken string) (UserInfo, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return UserInfo{}, fmt.Errorf("Google access token is empty")
	}
	req, err := http.NewRequest(http.MethodGet, c.userInfoURL(), nil)
	if err != nil {
		return UserInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", UserAgent)
	resp, err := c.http().Do(req)
	if err != nil {
		return UserInfo{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return UserInfo{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return UserInfo{}, fmt.Errorf("Google userinfo HTTP %d", resp.StatusCode)
	}
	var raw struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Sub   string `json:"sub"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return UserInfo{}, err
	}
	return UserInfo{Email: raw.Email, Name: raw.Name, Sub: raw.Sub}, nil
}

func (c *Client) postForm(endpoint string, form url.Values) ([]byte, error) {
	body, status, err := c.postFormStatus(endpoint, form)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return body, fmt.Errorf("HTTP %d: %s", status, truncate(string(body), 240))
	}
	return body, nil
}

func (c *Client) postFormStatus(endpoint string, form url.Values) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

func parseTokens(body []byte) (Tokens, error) {
	var raw struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    any    `json:"expires_in"`
		Scope        string `json:"scope"`
		IDToken      string `json:"id_token"`
		Error        string `json:"error"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return Tokens{}, fmt.Errorf("Google token response: %w", err)
	}
	if raw.Error != "" {
		return Tokens{}, fmt.Errorf("Google token: %s", raw.Error)
	}
	if raw.AccessToken == "" {
		return Tokens{}, fmt.Errorf("Google token response is missing access_token")
	}
	return Tokens{
		AccessToken:  raw.AccessToken,
		RefreshToken: raw.RefreshToken,
		ExpiresIn:    positiveSeconds(raw.ExpiresIn, time.Hour),
		TokenType:    raw.TokenType,
		Scope:        raw.Scope,
		IDToken:      raw.IDToken,
	}, nil
}

func positiveSeconds(v any, fallback time.Duration) time.Duration {
	var n int64
	switch t := v.(type) {
	case float64:
		n = int64(t)
	case int:
		n = int64(t)
	case int64:
		n = t
	case json.Number:
		n, _ = t.Int64()
	case string:
		n, _ = strconv.ParseInt(t, 10, 64)
	}
	if n <= 0 {
		return fallback
	}
	d := time.Duration(n) * time.Second
	if d < DeviceCodeMinInterval && fallback == DeviceCodeDefaultInterval {
		return DeviceCodeDefaultInterval
	}
	return d
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
