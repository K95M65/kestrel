package device

import (
	"os"
	"path/filepath"
	"testing"

	"go.autonomous.ai/os/system/server/config"
)

func TestGmailPATRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := &Service{config: &config.Config{OpenclawConfigDir: dir}}
	if err := s.SetGmailPAT(SetConnectorPAT{Code: "gmail", Email: "me@example.com", Key: "abcdefghijklmnop"}); err != nil {
		t.Fatal(err)
	}
	st := s.ListServices()
	var gmail ServiceStatus
	for _, row := range st {
		if row.ID == "gmail" {
			gmail = row
		}
	}
	if !gmail.Connected || gmail.UserEmail != "me@example.com" || gmail.AuthType != "pat" {
		t.Fatalf("gmail status = %+v", gmail)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "workspace", "configs", "gmail_access_tokens.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("empty token file")
	}
	if err := s.RemoveConnector("gmail"); err != nil {
		t.Fatal(err)
	}
	st = s.ListServices()
	for _, row := range st {
		if row.ID == "gmail" && row.Connected {
			t.Fatal("gmail still connected after remove")
		}
	}
}

func TestSetGmailPATRejectsCalendar(t *testing.T) {
	s := &Service{config: &config.Config{OpenclawConfigDir: t.TempDir()}}
	err := s.SetGmailPAT(SetConnectorPAT{Code: "google_calendar", Email: "a@b.c", Key: "abcdefghijklmnop"})
	if err == nil {
		t.Fatal("calendar PAT must fail")
	}
}

func TestSetCalendarICS(t *testing.T) {
	s := &Service{config: &config.Config{OpenclawConfigDir: t.TempDir()}}
	good := "https://calendar.google.com/calendar/ical/me%40example.com/private-abcdef/basic.ics"
	if err := s.SetCalendarICS(good); err != nil {
		t.Fatal(err)
	}
	st := s.ListServices()
	var cal ServiceStatus
	for _, row := range st {
		if row.ID == "google_calendar" {
			cal = row
		}
	}
	if !cal.Connected || cal.AuthType != "ical" || cal.ConnectHow != "ical" {
		t.Fatalf("calendar = %+v", cal)
	}
	if err := s.SetCalendarICS("http://evil.example/x.ics"); err == nil {
		t.Fatal("non-google URL must fail")
	}
	if err := s.SetCalendarICS("https://calendar.google.com/calendar/render"); err == nil {
		t.Fatal("non-ical path must fail")
	}
}

func TestListServicesTelegramFromConfig(t *testing.T) {
	s := &Service{config: &config.Config{
		OpenclawConfigDir: t.TempDir(),
		TelegramBotToken:  "123:abc",
		TelegramUserID:    "42",
	}}
	st := s.ListServices()
	var tg ServiceStatus
	for _, row := range st {
		if row.ID == "telegram" {
			tg = row
		}
	}
	if !tg.Connected || tg.Label != "42" {
		t.Fatalf("telegram = %+v", tg)
	}
}
