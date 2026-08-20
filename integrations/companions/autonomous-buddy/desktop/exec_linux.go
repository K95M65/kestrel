//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func currentOS() string { return "Linux " + runtime.GOARCH }

func openURL(raw, browser string) (result, error) {
	u, err := parseHTTPURL(raw)
	if err != nil {
		return nil, err
	}
	bin := firstOnPath("xdg-open", "gio")
	if bin == "" {
		return nil, fmt.Errorf("no xdg-open on PATH")
	}
	args := []string{u.String()}
	if bin == "gio" {
		args = []string{"open", u.String()}
	}
	if err := exec.Command(bin, args...).Start(); err != nil {
		return nil, err
	}
	return result{"opened": true, "browser": orDefault(browser)}, nil
}

func openApp(app string) (result, error) {
	if app == "" {
		return nil, fmt.Errorf("missing app")
	}
	name := strings.ToLower(strings.ReplaceAll(app, " ", "-"))
	if err := exec.Command(name).Start(); err != nil {
		if gtk := exec.Command("gtk-launch", name).Start(); gtk == nil {
			return result{"opened": true, "app": app}, nil
		}
		return nil, err
	}
	return result{"opened": true, "app": app}, nil
}

func closeApp(app string) (result, error) {
	if app == "" {
		return nil, fmt.Errorf("missing app")
	}
	name := strings.ToLower(strings.ReplaceAll(app, " ", "-"))
	out, err := exec.Command("pkill", "-f", name).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return result{"closed": true}, nil
}

func notify(title, body string) (result, error) {
	if title == "" {
		return nil, fmt.Errorf("missing title")
	}
	bin := firstOnPath("notify-send")
	if bin == "" {
		return nil, fmt.Errorf("notify-send not installed")
	}
	_ = exec.Command(bin, title, body).Run()
	return result{"delivered": true}, nil
}

func writeClipboard(text string) (result, error) {
	if text == "" {
		return nil, fmt.Errorf("missing text")
	}
	var cmd *exec.Cmd
	if os.Getenv("WAYLAND_DISPLAY") != "" && onPath("wl-copy") {
		cmd = exec.Command("wl-copy")
	} else if onPath("xclip") {
		cmd = exec.Command("xclip", "-selection", "clipboard")
	} else if onPath("xsel") {
		cmd = exec.Command("xsel", "--clipboard", "--input")
	} else {
		return nil, fmt.Errorf("install wl-copy, xclip, or xsel")
	}
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return result{"copied": true}, nil
}

func readClipboard() (result, error) {
	var cmd *exec.Cmd
	if os.Getenv("WAYLAND_DISPLAY") != "" && onPath("wl-paste") {
		cmd = exec.Command("wl-paste", "-n")
	} else if onPath("xclip") {
		cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
	} else if onPath("xsel") {
		cmd = exec.Command("xsel", "--clipboard", "--output")
	} else {
		return nil, fmt.Errorf("install wl-paste, xclip, or xsel")
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return result{"text": string(out)}, nil
}

func typeText(text string) (result, error) {
	if text == "" {
		return nil, fmt.Errorf("missing text")
	}
	if onPath("xdotool") {
		if err := exec.Command("xdotool", "type", "--", text).Run(); err != nil {
			return nil, err
		}
		return result{"typed": true}, nil
	}
	if onPath("ydotool") {
		if err := exec.Command("ydotool", "type", text).Run(); err != nil {
			return nil, err
		}
		return result{"typed": true}, nil
	}
	return nil, fmt.Errorf("install xdotool (X11) or ydotool (Wayland) to type")
}

func keyCombo(raw any) (result, error) {
	return nil, fmt.Errorf("key_combo needs xdotool/ydotool — not wired on this build yet")
}

func screenshot() (result, error) {
	return nil, fmt.Errorf("screenshot is not on this Linux build yet")
}

func firstOnPath(names ...string) string {
	for _, n := range names {
		if onPath(n) {
			return n
		}
	}
	return ""
}

func onPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func orDefault(s string) string {
	if s == "" {
		return "default"
	}
	return s
}
