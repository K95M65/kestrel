package main

import (
	"fmt"
	"net/url"
	"strings"
)

type result map[string]any

func dispatch(action string, params map[string]any) (result, error) {
	switch action {
	case "ping":
		return result{"pong": true, "os": currentOS()}, nil
	case "open_url":
		return openURL(str(params, "url"), str(params, "browser"))
	case "open_app":
		return openApp(str(params, "app"))
	case "close_app":
		return closeApp(str(params, "app"))
	case "notification":
		return notify(str(params, "title"), str(params, "body"))
	case "write_clipboard":
		return writeClipboard(str(params, "text"))
	case "read_clipboard":
		return readClipboard()
	case "type_text":
		return typeText(str(params, "text"))
	case "key_combo":
		return keyCombo(params["keys"])
	case "screenshot":
		return screenshot()
	default:
		return nil, fmt.Errorf("not supported on this computer yet: %s", action)
	}
}

func str(params map[string]any, key string) string {
	v, _ := params[key].(string)
	return strings.TrimSpace(v)
}

func parseHTTPURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, fmt.Errorf("missing url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid url")
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		if u.Host == "" {
			return nil, fmt.Errorf("invalid url")
		}
		return u, nil
	default:
		return nil, fmt.Errorf("url must be http or https")
	}
}
