package skillcontext

import (
	"testing"
	"time"

	"go.autonomous.ai/os/system/server/config"
)

func TestKitchenWindowOverride(t *testing.T) {
	SetBehaviorsSource(func() config.Behaviors {
		b := config.DefaultBehaviors()
		b.Kitchen.Enabled = true
		b.Kitchen.LunchStart = "12:00"
		b.Kitchen.LunchEnd = "13:00"
		return b
	})
	t.Cleanup(func() { SetBehaviorsSource(nil) })

	noon := time.Date(2026, 8, 18, 12, 15, 0, 0, time.UTC)
	w, ok := KitchenWindowFor(noon)
	if !ok || w != "lunch" {
		t.Fatalf("got %q ok=%v", w, ok)
	}
	eleven := time.Date(2026, 8, 18, 11, 40, 0, 0, time.UTC)
	w, ok = KitchenWindowFor(eleven)
	if !ok || w != "" {
		t.Fatalf("11:40 should be outside custom lunch, got %q", w)
	}
}

func TestBuildBehaviorsContext(t *testing.T) {
	tag := BuildBehaviorsContext()
	if tag == "" || tag[:13] != "\n[behaviors: " {
		t.Fatalf("tag = %q", tag)
	}
}
