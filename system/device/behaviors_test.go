package device

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.autonomous.ai/os/system/server/config"
)

func TestNormalizeMeLabel(t *testing.T) {
	got, err := NormalizeMeLabel("  Alex  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "alex" {
		t.Fatalf("got %q want alex", got)
	}
	cleared, err := NormalizeMeLabel("  ")
	if err != nil || cleared != "" {
		t.Fatalf("empty should clear, got %q err %v", cleared, err)
	}
	if _, err := NormalizeMeLabel("unknown"); err == nil {
		t.Fatal("unknown must be rejected")
	}
}

func TestValidateBehaviorsDefaults(t *testing.T) {
	b := config.DefaultBehaviors()
	if err := config.ValidateBehaviors(b); err != nil {
		t.Fatal(err)
	}
}

func TestValidateBehaviorsBadTime(t *testing.T) {
	b := config.DefaultBehaviors()
	b.MorningBrief.At = "25:00"
	if err := config.ValidateBehaviors(b); err == nil {
		t.Fatal("expected error")
	}
}

func TestInBriefWindow(t *testing.T) {
	mb := config.MorningBrief{Enabled: true, At: "07:30"}
	now := time.Date(2026, 8, 18, 7, 30, 0, 0, loc())
	if !inBriefWindow(now, mb) {
		t.Fatal("07:30 should fire")
	}
	if inBriefWindow(now.Add(3*time.Minute), mb) {
		t.Fatal("07:33 is past the 2-minute window")
	}
	mb.Days = []int{1} // Monday; 2026-08-18 is Tuesday
	if inBriefWindow(now, mb) {
		t.Fatal("Tuesday should be skipped")
	}
}

func TestNextBriefTime(t *testing.T) {
	mb := config.MorningBrief{At: "07:30"}
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, loc())
	got := nextBriefTime(now, mb)
	want := time.Date(2026, 8, 19, 7, 30, 0, 0, loc())
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestMemoriesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	memoryDir = dir
	t.Cleanup(func() { memoryDir = "/root/local/companion" })

	item, err := AddMemory("Sarah recommended Bar Moruno", 10)
	if err != nil {
		t.Fatal(err)
	}
	if item.Text != "Sarah recommended Bar Moruno" {
		t.Fatalf("text = %q", item.Text)
	}
	list := ListMemories()
	if len(list) != 1 || list[0].ID != item.ID {
		t.Fatalf("list = %+v", list)
	}
	if err := DeleteMemory(item.ID); err != nil {
		t.Fatal(err)
	}
	if n := CountMemories(); n != 0 {
		t.Fatalf("count = %d", n)
	}
}

func TestMemoriesTrim(t *testing.T) {
	memoryDir = t.TempDir()
	t.Cleanup(func() { memoryDir = "/root/local/companion" })
	for i := 0; i < 5; i++ {
		if _, err := AddMemory("note", 3); err != nil {
			t.Fatal(err)
		}
	}
	if n := CountMemories(); n != 3 {
		t.Fatalf("count = %d want 3", n)
	}
}

func TestBriefDayPersist(t *testing.T) {
	memoryDir = t.TempDir()
	t.Cleanup(func() { memoryDir = "/root/local/companion" })
	saveBriefDay("2026-08-18")
	if got := loadBriefDay(); got != "2026-08-18" {
		t.Fatalf("got %q", got)
	}
	if _, err := os.Stat(filepath.Join(memoryDir, "last_brief_day")); err != nil {
		t.Fatal(err)
	}
}
