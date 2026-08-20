package device

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/lib/hal"
	"go.autonomous.ai/os/system/lib/usercanon"
	"go.autonomous.ai/os/system/server/config"
	"go.autonomous.ai/os/system/skillcontext"
)

const behaviorsTickInterval = 30 * time.Second

type pomodoroRuntime struct {
	running bool
	phase   string // work | break
	endsAt  time.Time
}

type behaviorsRuntime struct {
	meeting      bool
	lastBriefDay string // YYYY-MM-DD in device zone
	pomodoro     pomodoroRuntime
}

func (s *Service) initBehaviors() {
	skillcontext.SetBehaviorsSource(func() config.Behaviors {
		if s == nil || s.config == nil {
			return config.DefaultBehaviors()
		}
		return s.config.BehaviorsOrDefault()
	})
}

func (s *Service) GetBehaviorsStatus() domain.BehaviorsStatus {
	b := s.config.BehaviorsOrDefault()
	now := s.clockNow()
	st := domain.BehaviorsStatus{
		Config:      b.RedactedForAPI(),
		HATokenSet:  s.config.Behaviors != nil && strings.TrimSpace(s.config.Behaviors.HomeAssistant.Token) != "",
		MemoryCount: CountMemories(),
	}
	s.behaviorsMu.Lock()
	st.Meeting = s.behaviors.meeting
	st.LastBrief = s.behaviors.lastBriefDay
	st.Pomodoro = pomodoroStatus(s.behaviors.pomodoro, now)
	s.behaviorsMu.Unlock()
	if b.MorningBrief.Enabled {
		if t := nextBriefTime(now, b.MorningBrief); !t.IsZero() {
			st.NextBrief = t.Format(time.RFC3339)
		}
	}
	return st
}

func pomodoroStatus(p pomodoroRuntime, now time.Time) domain.PomodoroStatus {
	if !p.running {
		return domain.PomodoroStatus{}
	}
	remain := int(p.endsAt.Sub(now).Seconds())
	if remain < 0 {
		remain = 0
	}
	return domain.PomodoroStatus{
		Running:   true,
		Phase:     p.phase,
		EndsAt:    p.endsAt.Format(time.RFC3339),
		RemainSec: remain,
	}
}

func (s *Service) SetBehaviors(in config.Behaviors) error {
	in.FillDefaults()
	me, err := NormalizeMeLabel(in.Me)
	if err != nil {
		return err
	}
	in.Me = me
	if err := config.ValidateBehaviors(in); err != nil {
		return err
	}
	if err := s.config.WithLockSave(func(c *config.Config) {
		prev := config.DefaultBehaviors()
		if c.Behaviors != nil {
			prev = *c.Behaviors
		}
		if strings.TrimSpace(in.HomeAssistant.Token) == "" {
			in.HomeAssistant.Token = prev.HomeAssistant.Token
		}
		// Settings PUT omits me; keep the People-card operator unless SetMe cleared it.
		if strings.TrimSpace(in.Me) == "" {
			in.Me = prev.Me
		}
		cp := in
		c.Behaviors = &cp
	}); err != nil {
		return err
	}
	s.initBehaviors()
	return nil
}

// NormalizeMeLabel turns a People label into the stored Me slug. Empty clears.
func NormalizeMeLabel(label string) (string, error) {
	if strings.TrimSpace(label) == "" {
		return "", nil
	}
	slug := usercanon.Slugify(label)
	if slug == "" || slug == usercanon.DefaultUser {
		return "", fmt.Errorf("pick a person")
	}
	return slug, nil
}

// SetMe records which household member is the operator (People card "Me").
// Empty label clears it; everyone enrolled is then just a friend.
func (s *Service) SetMe(label string) error {
	slug, err := NormalizeMeLabel(label)
	if err != nil {
		return err
	}
	return s.config.WithLockSave(func(c *config.Config) {
		if c.Behaviors == nil {
			d := config.DefaultBehaviors()
			c.Behaviors = &d
		}
		c.Behaviors.Me = slug
	})
}

func (s *Service) IsMeeting() bool {
	s.behaviorsMu.Lock()
	defer s.behaviorsMu.Unlock()
	return s.behaviors.meeting
}

func (s *Service) SetMeeting(on bool) error {
	if on {
		return s.applyMeeting("meeting-on")
	}
	return s.clearMeeting("meeting-off")
}

func (s *Service) applyMeeting(reason string) error {
	slog.Info("behaviors: meeting on", "component", "device", "reason", reason)
	_ = hal.StopAudio()
	_ = hal.StopServoTracking()
	_ = hal.PostRaw("/voice/mute", "{}")
	_ = hal.PostRaw("/speaker/mute", "{}")
	_ = hal.PostRaw("/camera/disable", "{}")
	s.behaviorsMu.Lock()
	s.behaviors.meeting = true
	s.behaviorsMu.Unlock()
	return nil
}

func (s *Service) clearMeeting(reason string) error {
	slog.Info("behaviors: meeting off", "component", "device", "reason", reason)
	_ = hal.PostRaw("/voice/unmute", "{}")
	_ = hal.PostRaw("/speaker/unmute", "{}")
	_ = hal.PostRaw("/camera/enable", "{}")
	s.behaviorsMu.Lock()
	s.behaviors.meeting = false
	s.behaviorsMu.Unlock()
	return nil
}

func (s *Service) FireMorningBrief(reason string) error {
	b := s.config.BehaviorsOrDefault()
	if !b.MorningBrief.Enabled && reason != "manual" {
		return fmt.Errorf("morning brief is off")
	}
	if s.IsQuiet() || s.IsMeeting() {
		return fmt.Errorf("quiet or meeting — brief skipped")
	}
	if s.agentGateway == nil || !s.agentGateway.IsReady() {
		return fmt.Errorf("agent not ready")
	}
	msg := buildBriefPrompt(b)
	if _, err := s.agentGateway.SendChatMessage(msg); err != nil {
		return err
	}
	day := s.clockNow().Format("2006-01-02")
	s.behaviorsMu.Lock()
	s.behaviors.lastBriefDay = day
	s.behaviorsMu.Unlock()
	saveBriefDay(day)
	slog.Info("behaviors: morning brief fired", "component", "device", "reason", reason)
	return nil
}

func buildBriefPrompt(b config.Behaviors) string {
	var parts []string
	if b.MorningBrief.Weather {
		parts = append(parts, "weather")
	}
	if b.MorningBrief.Calendar {
		parts = append(parts, "today's calendar")
	}
	if b.MorningBrief.Email {
		parts = append(parts, "overnight email (read-only, never send)")
	}
	if b.MorningBrief.Habits {
		parts = append(parts, "one habit beat")
	}
	if b.Wearables.Enabled && b.Wearables.Provider != "" && b.Wearables.Provider != "none" {
		parts = append(parts, b.Wearables.Provider+" recovery if a connector exists")
	}
	include := "weather"
	if len(parts) > 0 {
		include = strings.Join(parts, ", ")
	}
	budget := b.MorningBrief.MaxSeconds
	if budget <= 0 {
		budget = 40
	}
	channel := "Speak it out loud"
	if !b.MorningBrief.Speak {
		channel = "Do not speak — Telegram/chat only"
	}
	if b.MorningBrief.Telegram {
		channel += ". Also send a short Telegram copy if a known user's telegram_id is in context"
	}
	return "[companion:morning-brief] Isolated morning briefing. Follow morning-brief/SKILL.md. Include: " +
		include + ". Budget ~" + fmt.Sprintf("%d", budget) + " seconds spoken. " + channel +
		". Read-only — never send mail or write calendar. Do not write MEMORY.md this turn."
}

func (s *Service) StartPomodoro() error {
	b := s.config.BehaviorsOrDefault()
	if !b.Pomodoro.Enabled {
		return fmt.Errorf("pomodoro is off")
	}
	now := s.clockNow()
	s.behaviorsMu.Lock()
	s.behaviors.pomodoro = pomodoroRuntime{
		running: true,
		phase:   "work",
		endsAt:  now.Add(time.Duration(b.Pomodoro.WorkMin) * time.Minute),
	}
	s.behaviorsMu.Unlock()
	slog.Info("behaviors: pomodoro work started", "component", "device", "min", b.Pomodoro.WorkMin)
	return nil
}

func (s *Service) StopPomodoro() {
	s.behaviorsMu.Lock()
	s.behaviors.pomodoro = pomodoroRuntime{}
	s.behaviorsMu.Unlock()
}

func (s *Service) StartBehaviorsLoop(ctx context.Context) {
	s.initBehaviors()
	if day := loadBriefDay(); day != "" {
		s.behaviorsMu.Lock()
		s.behaviors.lastBriefDay = day
		s.behaviorsMu.Unlock()
	}
	t := time.NewTicker(behaviorsTickInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.tickBehaviors()
		}
	}
}

func (s *Service) tickBehaviors() {
	now := s.clockNow()
	b := s.config.BehaviorsOrDefault()
	if b.MorningBrief.Enabled && !s.IsQuiet() && !s.IsMeeting() {
		s.behaviorsMu.Lock()
		already := s.behaviors.lastBriefDay == now.Format("2006-01-02")
		s.behaviorsMu.Unlock()
		if !already && inBriefWindow(now, b.MorningBrief) {
			if err := s.FireMorningBrief("schedule"); err != nil {
				slog.Warn("behaviors: morning brief failed", "component", "device", "error", err)
			}
		}
	}
	s.tickPomodoro(now, b)
}

func (s *Service) tickPomodoro(now time.Time, b config.Behaviors) {
	s.behaviorsMu.Lock()
	p := s.behaviors.pomodoro
	s.behaviorsMu.Unlock()
	if !p.running || now.Before(p.endsAt) {
		return
	}
	nextPhase := "break"
	mins := b.Pomodoro.BreakMin
	prompt := "[companion:pomodoro] Work block is over. Follow pomodoro/SKILL.md — invite a short break. One or two sentences."
	if p.phase == "break" {
		nextPhase = "work"
		mins = b.Pomodoro.WorkMin
		prompt = "[companion:pomodoro] Break is over. Follow pomodoro/SKILL.md — invite them back to the desk. One or two sentences."
	}
	s.behaviorsMu.Lock()
	s.behaviors.pomodoro = pomodoroRuntime{
		running: true,
		phase:   nextPhase,
		endsAt:  now.Add(time.Duration(mins) * time.Minute),
	}
	s.behaviorsMu.Unlock()
	if s.agentGateway != nil && s.agentGateway.IsReady() && !s.IsQuiet() && !s.IsMeeting() {
		if _, err := s.agentGateway.SendChatMessage(prompt); err != nil {
			slog.Warn("behaviors: pomodoro prompt failed", "component", "device", "error", err)
		}
	}
}

func inBriefWindow(now time.Time, mb config.MorningBrief) bool {
	wantM, ok := parseHHMM(mb.At)
	if !ok {
		return false
	}
	nowM := now.Hour()*60 + now.Minute()
	if nowM < wantM || nowM > wantM+2 {
		return false
	}
	return dayAllowed(now.Weekday(), mb.Days)
}

func nextBriefTime(now time.Time, mb config.MorningBrief) time.Time {
	h, m, ok := splitHHMM(mb.At)
	if !ok {
		return time.Time{}
	}
	for i := 0; i < 8; i++ {
		day := now.AddDate(0, 0, i)
		t := time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, now.Location())
		if !t.After(now) {
			continue
		}
		if dayAllowed(t.Weekday(), mb.Days) {
			return t
		}
	}
	return time.Time{}
}
