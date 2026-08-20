//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
)

func currentOS() string { return "Windows " + runtime.GOARCH }

func openURL(raw, browser string) (result, error) {
	u, err := parseHTTPURL(raw)
	if err != nil {
		return nil, err
	}
	// cmd /c start needs an empty title argument when the URL is quoted.
	cmd := exec.Command("cmd", "/c", "start", "", u.String())
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return result{"opened": true, "browser": orDefault(browser)}, nil
}

func openApp(app string) (result, error) {
	if app == "" {
		return nil, fmt.Errorf("missing app")
	}
	cmd := exec.Command("cmd", "/c", "start", "", app)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return result{"opened": true, "app": app}, nil
}

func closeApp(app string) (result, error) {
	if app == "" {
		return nil, fmt.Errorf("missing app")
	}
	name := app
	if !strings.HasSuffix(strings.ToLower(name), ".exe") {
		name += ".exe"
	}
	cmd := exec.Command("taskkill", "/IM", name)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return result{"closed": true}, nil
}

func notify(title, body string) (result, error) {
	if title == "" {
		return nil, fmt.Errorf("missing title")
	}
	ps := fmt.Sprintf(
		`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null; $xml = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02); $t = $xml.GetElementsByTagName('text'); $t.Item(0).AppendChild($xml.CreateTextNode(%s)) | Out-Null; $t.Item(1).AppendChild($xml.CreateTextNode(%s)) | Out-Null; [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('Kestrel Buddy').Show([Windows.UI.Notifications.ToastNotification]::new($xml))`,
		psQuote(title), psQuote(body),
	)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Run()
	return result{"delivered": true}, nil
}

func writeClipboard(text string) (result, error) {
	if text == "" {
		return nil, fmt.Errorf("missing text")
	}
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "Set-Clipboard -Value $input")
	cmd.Stdin = strings.NewReader(text)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return result{"copied": true}, nil
}

func readClipboard() (result, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "Get-Clipboard")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return result{"text": strings.TrimRight(string(out), "\r\n")}, nil
}

func typeText(text string) (result, error) {
	if text == "" {
		return nil, fmt.Errorf("missing text")
	}
	escaped := strings.ReplaceAll(text, "}", "}}")
	escaped = strings.ReplaceAll(escaped, "{", "{{")
	ps := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.SendKeys]::SendWait(%s)`, psQuote(escaped))
	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return result{"typed": true}, nil
}

func keyCombo(raw any) (result, error) {
	return nil, fmt.Errorf("key_combo is not on this Windows build yet")
}

func screenshot() (result, error) {
	return nil, fmt.Errorf("screenshot is not on this Windows build yet")
}

func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func orDefault(s string) string {
	if s == "" {
		return "default"
	}
	return s
}
