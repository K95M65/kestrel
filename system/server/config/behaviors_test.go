package config

import "testing"

func TestValidateBehaviorsOK(t *testing.T) {
	if err := ValidateBehaviors(DefaultBehaviors()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateBehaviorsKitchenSame(t *testing.T) {
	b := DefaultBehaviors()
	b.Kitchen.LunchStart = "12:00"
	b.Kitchen.LunchEnd = "12:00"
	if err := ValidateBehaviors(b); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateBehaviorsHAURL(t *testing.T) {
	b := DefaultBehaviors()
	b.HomeAssistant.Enabled = true
	b.HomeAssistant.URL = "not-a-url"
	if err := ValidateBehaviors(b); err == nil {
		t.Fatal("expected error")
	}
	b.HomeAssistant.URL = "http://homeassistant.local:8123"
	if err := ValidateBehaviors(b); err != nil {
		t.Fatal(err)
	}
}

func TestRedactedForAPI(t *testing.T) {
	b := DefaultBehaviors()
	b.HomeAssistant.Token = "secret"
	if b.RedactedForAPI().HomeAssistant.Token != "" {
		t.Fatal("token leaked")
	}
}
