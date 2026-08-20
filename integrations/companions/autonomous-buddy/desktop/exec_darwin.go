//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func currentOS() string { return "macOS " + runtime.GOARCH }

func openURL(raw, browser string) (result, error) {
	u, err := parseHTTPURL(raw)
	if err != nil {
		return nil, err
	}
	args := []string{u.String()}
	if b := strings.ToLower(browser); b != "" && b != "default" {
		args = []string{"-a", browserApp(b), u.String()}
	}
	if err := exec.Command("open", args...).Start(); err != nil {
		return nil, err
	}
	return result{"opened": true, "browser": orDefault(browser)}, nil
}

func browserApp(name string) string {
	switch name {
	case "chrome", "google chrome":
		return "Google Chrome"
	case "safari":
		return "Safari"
	case "firefox":
		return "Firefox"
	case "edge", "microsoft edge":
		return "Microsoft Edge"
	case "brave":
		return "Brave Browser"
	default:
		return name
	}
}

func openApp(app string) (result, error) {
	if app == "" {
		return nil, fmt.Errorf("missing app")
	}
	if err := exec.Command("open", "-a", app).Start(); err != nil {
		return nil, err
	}
	return result{"opened": true, "app": app}, nil
}

func closeApp(app string) (result, error) {
	if app == "" {
		return nil, fmt.Errorf("missing app")
	}
	script := fmt.Sprintf(`tell application %q to quit`, app)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return result{"closed": true}, nil
}

func notify(title, body string) (result, error) {
	if title == "" {
		return nil, fmt.Errorf("missing title")
	}
	script := fmt.Sprintf(`display notification %q with title %q`, body, title)
	_ = exec.Command("osascript", "-e", script).Run()
	return result{"delivered": true}, nil
}

func writeClipboard(text string) (result, error) {
	if text == "" {
		return nil, fmt.Errorf("missing text")
	}
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return result{"copied": true}, nil
}

func readClipboard() (result, error) {
	out, err := exec.Command("pbpaste").Output()
	if err != nil {
		return nil, err
	}
	return result{"text": string(out)}, nil
}

func typeText(text string) (result, error) {
	if text == "" {
		return nil, fmt.Errorf("missing text")
	}
	script := fmt.Sprintf(`tell application "System Events" to keystroke %q`, text)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return result{"typed": true}, nil
}

func keyCombo(raw any) (result, error) {
	keys, ok := raw.([]any)
	if !ok || len(keys) == 0 {
		return nil, fmt.Errorf("missing keys")
	}
	return nil, fmt.Errorf("key_combo on this desktop build: use the Mac app for Accessibility shortcuts")
}

func screenshot() (result, error) {
	return nil, fmt.Errorf("screenshot: use the Mac Kestrel Buddy app for screen capture")
}

func orDefault(s string) string {
	if s == "" {
		return "default"
	}
	return s
}
