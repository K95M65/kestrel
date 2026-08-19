package device

import (
	"testing"
	"time"

	"go.autonomous.ai/os/system/server/config"
)

func loc() *time.Location { return time.FixedZone("test", -5*3600) }

func at(h, m int) time.Time {
	return time.Date(2026, 8, 18, h, m, 0, 0, loc()) // Tuesday
}

func TestInQuietHoursOvernight(t *testing.T) {
	sched := config.SleepSchedule{Enabled: true, SleepAt: "23:00", WakeAt: "07:00"}
	if !InQuietHours(at(23, 30), sched) {
		t.Fatal("23:30 should be asleep")
	}
	if !InQuietHours(at(2, 0), sched) {
		t.Fatal("02:00 should be asleep")
	}
	if InQuietHours(at(7, 0), sched) {
		t.Fatal("07:00 should be awake")
	}
	if InQuietHours(at(12, 0), sched) {
		t.Fatal("noon should be awake")
	}
	if InQuietHours(at(22, 59), sched) {
		t.Fatal("22:59 should be awake")
	}
}

func TestInQuietHoursSameDayNap(t *testing.T) {
	sched := config.SleepSchedule{Enabled: true, SleepAt: "13:00", WakeAt: "14:30"}
	if !InQuietHours(at(13, 0), sched) {
		t.Fatal("13:00 should be asleep")
	}
	if !InQuietHours(at(14, 0), sched) {
		t.Fatal("14:00 should be asleep")
	}
	if InQuietHours(at(14, 30), sched) {
		t.Fatal("14:30 should be awake")
	}
	if InQuietHours(at(12, 0), sched) {
		t.Fatal("12:00 should be awake")
	}
}

func TestInQuietHoursDaysOvernight(t *testing.T) {
	// Only Monday (1). Tuesday 02:00 is still Monday night's window.
	sched := config.SleepSchedule{Enabled: true, SleepAt: "23:00", WakeAt: "07:00", Days: []int{1}}
	monNight := time.Date(2026, 8, 17, 23, 30, 0, 0, loc()) // Monday
	tueMorning := time.Date(2026, 8, 18, 2, 0, 0, 0, loc()) // Tuesday
	tueNight := time.Date(2026, 8, 18, 23, 30, 0, 0, loc())
	if !InQuietHours(monNight, sched) {
		t.Fatal("Monday 23:30 should be asleep")
	}
	if !InQuietHours(tueMorning, sched) {
		t.Fatal("Tuesday 02:00 is still Monday's quiet hours")
	}
	if InQuietHours(tueNight, sched) {
		t.Fatal("Tuesday 23:30 is not a scheduled sleep day")
	}
}

func TestInQuietHoursDisabled(t *testing.T) {
	sched := config.SleepSchedule{Enabled: false, SleepAt: "23:00", WakeAt: "07:00"}
	if InQuietHours(at(23, 30), sched) {
		t.Fatal("disabled schedule must not sleep")
	}
}

func TestValidateSleepSchedule(t *testing.T) {
	if err := ValidateSleepSchedule(config.SleepSchedule{Enabled: true, SleepAt: "23:00", WakeAt: "07:00"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSleepSchedule(config.SleepSchedule{Enabled: true, SleepAt: "25:00", WakeAt: "07:00"}); err == nil {
		t.Fatal("expected invalid sleep_at")
	}
	if err := ValidateSleepSchedule(config.SleepSchedule{Enabled: true, SleepAt: "23:00", WakeAt: "23:00"}); err == nil {
		t.Fatal("expected identical times rejected")
	}
}

func TestNextSleepAndWake(t *testing.T) {
	sched := config.SleepSchedule{Enabled: true, SleepAt: "23:00", WakeAt: "07:00"}
	n := at(12, 0)
	got := NextSleepTime(n, sched)
	want := time.Date(2026, 8, 18, 23, 0, 0, 0, loc())
	if !got.Equal(want) {
		t.Fatalf("next sleep = %s want %s", got, want)
	}
	got = NextWakeTime(at(23, 30), sched)
	want = time.Date(2026, 8, 19, 7, 0, 0, 0, loc())
	if !got.Equal(want) {
		t.Fatalf("next wake = %s want %s", got, want)
	}
}
