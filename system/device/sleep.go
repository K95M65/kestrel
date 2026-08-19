package device

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/lib/hal"
	"go.autonomous.ai/os/system/server/config"
)

const sleepTickInterval = 30 * time.Second

var hhmmRe = regexp.MustCompile(`^([01]?\d|2[0-3]):([0-5]\d)$`)

type sleepRuntime struct {
	asleep           bool
	manualAsleep     bool
	manualAwakeUntil time.Time
}

// InQuietHours reports whether now is inside the schedule (device-local clock).
func InQuietHours(now time.Time, sched config.SleepSchedule) bool {
	if !sched.Enabled {
		return false
	}
	sleepM, ok1 := parseHHMM(sched.SleepAt)
	wakeM, ok2 := parseHHMM(sched.WakeAt)
	if !ok1 || !ok2 || sleepM == wakeM {
		return false
	}
	nowM := now.Hour()*60 + now.Minute()
	if sleepM < wakeM {
		if nowM < sleepM || nowM >= wakeM {
			return false
		}
		return dayAllowed(now.Weekday(), sched.Days)
	}
	if nowM >= sleepM {
		return dayAllowed(now.Weekday(), sched.Days)
	}
	if nowM < wakeM {
		yest := now.AddDate(0, 0, -1).Weekday()
		return dayAllowed(yest, sched.Days)
	}
	return false
}

func NextSleepTime(now time.Time, sched config.SleepSchedule) time.Time {
	h, m, ok := splitHHMM(sched.SleepAt)
	if !ok {
		return time.Time{}
	}
	for i := 0; i < 8; i++ {
		day := now.AddDate(0, 0, i)
		t := time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, now.Location())
		if !t.After(now) {
			continue
		}
		if dayAllowed(t.Weekday(), sched.Days) {
			return t
		}
	}
	return time.Time{}
}

func NextWakeTime(now time.Time, sched config.SleepSchedule) time.Time {
	h, m, ok := splitHHMM(sched.WakeAt)
	if !ok {
		return time.Time{}
	}
	for i := 0; i < 8; i++ {
		day := now.AddDate(0, 0, i)
		t := time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, now.Location())
		if !t.After(now) {
			continue
		}
		// Overnight: wake belongs to the morning after a scheduled sleep day.
		sleepM, wakeM, ok := parseHHMMPair(sched.SleepAt, sched.WakeAt)
		if !ok {
			return time.Time{}
		}
		if sleepM < wakeM {
			if dayAllowed(t.Weekday(), sched.Days) {
				return t
			}
			continue
		}
		yest := t.AddDate(0, 0, -1).Weekday()
		if dayAllowed(yest, sched.Days) {
			return t
		}
	}
	return time.Time{}
}

func ValidateSleepSchedule(sched config.SleepSchedule) error {
	if strings.TrimSpace(sched.SleepAt) == "" && strings.TrimSpace(sched.WakeAt) == "" && !sched.Enabled {
		return nil
	}
	if _, ok := parseHHMM(sched.SleepAt); !ok {
		return fmt.Errorf("sleep_at must be HH:MM")
	}
	if _, ok := parseHHMM(sched.WakeAt); !ok {
		return fmt.Errorf("wake_at must be HH:MM")
	}
	if sched.SleepAt == sched.WakeAt {
		return fmt.Errorf("sleep_at and wake_at must differ")
	}
	for _, d := range sched.Days {
		if d < 0 || d > 6 {
			return fmt.Errorf("days must be 0–6 (Sunday–Saturday)")
		}
	}
	return nil
}

func parseHHMM(s string) (int, bool) {
	h, m, ok := splitHHMM(s)
	if !ok {
		return 0, false
	}
	return h*60 + m, true
}

func parseHHMMPair(sleepAt, wakeAt string) (sleepM, wakeM int, ok bool) {
	sleepM, ok1 := parseHHMM(sleepAt)
	wakeM, ok2 := parseHHMM(wakeAt)
	return sleepM, wakeM, ok1 && ok2
}

func splitHHMM(s string) (hour, minute int, ok bool) {
	m := hhmmRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, 0, false
	}
	hour = atoi2(m[1])
	minute = atoi2(m[2])
	return hour, minute, true
}

func atoi2(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

func dayAllowed(wd time.Weekday, days []int) bool {
	if len(days) == 0 {
		return true
	}
	for _, d := range days {
		if time.Weekday(d) == wd {
			return true
		}
	}
	return false
}

func (s *Service) clockNow() time.Time {
	if s.config != nil && s.config.Timezone != "" {
		if loc, err := time.LoadLocation(s.config.Timezone); err == nil {
			return time.Now().In(loc)
		}
	}
	return time.Now()
}

func (s *Service) wantsSleep(now time.Time) bool {
	if s.sleep.manualAsleep {
		return true
	}
	if !s.sleep.manualAwakeUntil.IsZero() && now.Before(s.sleep.manualAwakeUntil) {
		return false
	}
	return InQuietHours(now, s.config.SleepScheduleOrZero())
}

// IsQuiet is true while scheduled or manual quiet hours are applied.
func (s *Service) IsQuiet() bool {
	s.sleepMu.Lock()
	defer s.sleepMu.Unlock()
	return s.sleep.asleep
}

func (s *Service) GetSleepStatus() domain.SleepStatus {
	s.sleepMu.Lock()
	asleep := s.sleep.asleep
	s.sleepMu.Unlock()

	sched := s.config.SleepScheduleOrZero()
	now := s.clockNow()
	st := domain.SleepStatus{
		Sleeping:  asleep,
		Scheduled: InQuietHours(now, sched) && sched.Enabled,
		Schedule:  domain.SleepSchedule(sched),
	}
	if emotion, err := hal.GetEmotion(); err == nil {
		st.Emotion = emotion
	}
	if sched.Enabled {
		if asleep {
			if t := NextWakeTime(now, sched); !t.IsZero() {
				st.NextTransition = t.Format(time.RFC3339)
				st.NextTransitionKind = "wake"
			}
		} else if t := NextSleepTime(now, sched); !t.IsZero() {
			st.NextTransition = t.Format(time.RFC3339)
			st.NextTransitionKind = "sleep"
		}
	}
	return st
}

func (s *Service) SetSleepSchedule(sched config.SleepSchedule) error {
	if err := ValidateSleepSchedule(sched); err != nil {
		return err
	}
	if err := s.config.WithLockSave(func(c *config.Config) {
		cp := sched
		c.SleepSchedule = &cp
	}); err != nil {
		return err
	}
	s.reconcileSleep("schedule-save")
	return nil
}

func (s *Service) SleepNow() error {
	s.sleepMu.Lock()
	s.sleep.manualAsleep = true
	s.sleep.manualAwakeUntil = time.Time{}
	s.sleepMu.Unlock()
	return s.applySleep("sleep-now")
}

func (s *Service) WakeNow() error {
	now := s.clockNow()
	s.sleepMu.Lock()
	s.sleep.manualAsleep = false
	if InQuietHours(now, s.config.SleepScheduleOrZero()) {
		s.sleep.manualAwakeUntil = NextSleepTime(now, s.config.SleepScheduleOrZero())
	} else {
		s.sleep.manualAwakeUntil = time.Time{}
	}
	s.sleepMu.Unlock()
	return s.applyWake("wake-now")
}

func (s *Service) StartSleepLoop(ctx context.Context) {
	s.reconcileSleep("boot")
	t := time.NewTicker(sleepTickInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.reconcileSleep("tick")
		}
	}
}

func (s *Service) reconcileSleep(reason string) {
	now := s.clockNow()
	s.sleepMu.Lock()
	if !s.sleep.manualAwakeUntil.IsZero() && !now.Before(s.sleep.manualAwakeUntil) {
		s.sleep.manualAwakeUntil = time.Time{}
	}
	want := s.wantsSleep(now)
	already := s.sleep.asleep
	s.sleepMu.Unlock()
	if want && !already {
		_ = s.applySleep(reason)
	} else if !want && already {
		_ = s.applyWake(reason)
	}
}

func (s *Service) applySleep(reason string) error {
	slog.Info("quiet hours: sleep", "component", "device", "reason", reason)
	_ = hal.StopAudio()
	_ = hal.StopServoTracking()
	if s.plugins != nil {
		s.plugins.StopRunning()
	}
	if err := hal.SetEmotion("sleepy", 0.8); err != nil {
		slog.Warn("quiet hours: sleepy emotion failed", "component", "device", "error", err)
		// Still mark quiet so sensing stays dark even if HAL is briefly down.
	}
	s.sleepMu.Lock()
	s.sleep.asleep = true
	s.sleepMu.Unlock()
	return nil
}

func (s *Service) applyWake(reason string) error {
	slog.Info("quiet hours: wake", "component", "device", "reason", reason)
	if err := hal.SetEmotion("greeting", 0.7); err != nil {
		slog.Warn("quiet hours: greeting emotion failed", "component", "device", "error", err)
	}
	s.sleepMu.Lock()
	s.sleep.asleep = false
	s.sleepMu.Unlock()
	return nil
}
