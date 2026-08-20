package config

import (
	"fmt"
	"regexp"
	"strings"
)

// Behaviors is the operator-facing companion pack: morning brief, memory,
// dance-to-song, privacy, kids, kitchen, Home Assistant, and the rest of
// the desk-robot jobs. Nil on disk means DefaultBehaviors().
type Behaviors struct {
	// Onboarded is true after the operator finishes or skips the guided
	// setup. False/absent → Overview offers the interactive configuration.
	Onboarded bool `json:"onboarded,omitempty"`

	// Me is the household member who is the operator — their People card
	// is labeled Me; everyone else enrolled is a Friend. Empty = unset.
	// Folder-safe label, same slug as /root/local/users/<me>/.
	Me string `json:"me,omitempty"`

	MorningBrief  MorningBrief  `json:"morning_brief"`
	Remember      Remember      `json:"remember"`
	Dance         Dance         `json:"dance"`
	Privacy       Privacy       `json:"privacy"`
	Connectors    ConnectorGate `json:"connectors"`
	Presence      PresenceIdle  `json:"presence"`
	DoA           FeatureFlag   `json:"doa"`
	LayeredMotion FeatureFlag   `json:"layered_motion"`
	Focus         FocusCoach    `json:"focus"`
	Kids          KidsProfile   `json:"kids"`
	Greeter       Greeter       `json:"greeter"`
	Look          LookAtThis    `json:"look"`
	Kitchen       Kitchen       `json:"kitchen"`
	HomeAssistant HomeAssistant `json:"home_assistant"`
	Marionette    FeatureFlag   `json:"marionette"`
	Tools         AmbientTools  `json:"tools"`
	HandTrack     FeatureFlag   `json:"hand_track"`
	Radio         FeatureFlag   `json:"radio"`
	Telepresence  FeatureFlag   `json:"telepresence"`
	Stories       Stories       `json:"stories"`
	Pomodoro      Pomodoro      `json:"pomodoro"`
	Wearables     Wearables     `json:"wearables"`
}

type FeatureFlag struct {
	Enabled bool `json:"enabled"`
}

type MorningBrief struct {
	Enabled    bool   `json:"enabled"`
	At         string `json:"at"`
	Days       []int  `json:"days,omitempty"`
	Speak      bool   `json:"speak"`
	Telegram   bool   `json:"telegram"`
	Weather    bool   `json:"weather"`
	Calendar   bool   `json:"calendar"`
	Email      bool   `json:"email"`
	Habits     bool   `json:"habits"`
	MaxSeconds int    `json:"max_seconds"`
}

type Remember struct {
	Enabled  bool `json:"enabled"`
	MaxItems int  `json:"max_items"`
}

type Dance struct {
	Enabled      bool   `json:"enabled"`
	DefaultQuery string `json:"default_query"`
}

type Privacy struct {
	CameraOnDemand      bool `json:"camera_on_demand"`
	FaceFollowAfterWake bool `json:"face_follow_after_wake"`
}

type ConnectorGate struct {
	DraftNotSend bool `json:"draft_not_send"`
}

type PresenceIdle struct {
	IdleMotion bool `json:"idle_motion"`
}

type FocusCoach struct {
	Enabled     bool `json:"enabled"`
	PhoneNag    bool `json:"phone_nag"`
	CooldownMin int  `json:"cooldown_min"`
}

type KidsProfile struct {
	Enabled    bool `json:"enabled"`
	SessionMin int  `json:"session_min"`
}

type Greeter struct {
	Enabled   bool `json:"enabled"`
	NamedOnly bool `json:"named_only"`
}

type LookAtThis struct {
	Enabled bool `json:"enabled"`
}

type Kitchen struct {
	Enabled     bool   `json:"enabled"`
	LunchStart  string `json:"lunch_start"`
	LunchEnd    string `json:"lunch_end"`
	DinnerStart string `json:"dinner_start"`
	DinnerEnd   string `json:"dinner_end"`
}

type HomeAssistant struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	// Token is write-only. GET never returns it; empty PUT keeps the stored value.
	Token string `json:"token,omitempty"`
}

type AmbientTools struct {
	Weather bool `json:"weather"`
	Time    bool `json:"time"`
	Search  bool `json:"search"`
}

type Stories struct {
	Enabled bool `json:"enabled"`
	MaxMin  int  `json:"max_min"`
}

type Pomodoro struct {
	Enabled  bool `json:"enabled"`
	WorkMin  int  `json:"work_min"`
	BreakMin int  `json:"break_min"`
}

type Wearables struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider,omitempty"`
}

var behaviorsHHMM = regexp.MustCompile(`^([01]?\d|2[0-3]):([0-5]\d)$`)

// DefaultBehaviors is the first-boot / missing-key pack. Opt-in for
// scheduled jobs; on for memory, dance, draft-not-send, greeter, look,
// kitchen times (matching the historic wellbeing meal windows).
func DefaultBehaviors() Behaviors {
	return Behaviors{
		MorningBrief: MorningBrief{
			At: "07:30", Speak: true, Telegram: true,
			Weather: true, Calendar: true, Email: true, Habits: true,
			MaxSeconds: 40,
		},
		Remember:   Remember{Enabled: true, MaxItems: 200},
		Dance:      Dance{Enabled: true, DefaultQuery: "upbeat dance pop"},
		Privacy:    Privacy{CameraOnDemand: true, FaceFollowAfterWake: true},
		Connectors: ConnectorGate{DraftNotSend: true},
		Presence:   PresenceIdle{IdleMotion: true},
		Focus:      FocusCoach{PhoneNag: true, CooldownMin: 15},
		Kids:       KidsProfile{SessionMin: 30},
		Greeter:    Greeter{Enabled: true},
		Look:       LookAtThis{Enabled: true},
		Kitchen: Kitchen{
			Enabled:     true,
			LunchStart:  "11:30",
			LunchEnd:    "13:30",
			DinnerStart: "18:30",
			DinnerEnd:   "20:30",
		},
		Tools:    AmbientTools{Weather: true, Time: true, Search: true},
		Stories:  Stories{MaxMin: 10},
		Pomodoro: Pomodoro{WorkMin: 25, BreakMin: 5},
	}
}

func (c *Config) BehaviorsOrDefault() Behaviors {
	if c == nil || c.Behaviors == nil {
		return DefaultBehaviors()
	}
	b := *c.Behaviors
	b.FillDefaults()
	return b
}

// FillDefaults writes zero-value knobs so a partial on-disk block still
// has usable times and limits. Enabled flags stay as stored.
func (b *Behaviors) FillDefaults() {
	d := DefaultBehaviors()
	if strings.TrimSpace(b.MorningBrief.At) == "" {
		b.MorningBrief.At = d.MorningBrief.At
	}
	if b.MorningBrief.MaxSeconds <= 0 {
		b.MorningBrief.MaxSeconds = d.MorningBrief.MaxSeconds
	}
	if b.Remember.MaxItems <= 0 {
		b.Remember.MaxItems = d.Remember.MaxItems
	}
	if strings.TrimSpace(b.Dance.DefaultQuery) == "" {
		b.Dance.DefaultQuery = d.Dance.DefaultQuery
	}
	if b.Focus.CooldownMin <= 0 {
		b.Focus.CooldownMin = d.Focus.CooldownMin
	}
	if b.Kids.SessionMin <= 0 {
		b.Kids.SessionMin = d.Kids.SessionMin
	}
	if strings.TrimSpace(b.Kitchen.LunchStart) == "" {
		b.Kitchen.LunchStart = d.Kitchen.LunchStart
	}
	if strings.TrimSpace(b.Kitchen.LunchEnd) == "" {
		b.Kitchen.LunchEnd = d.Kitchen.LunchEnd
	}
	if strings.TrimSpace(b.Kitchen.DinnerStart) == "" {
		b.Kitchen.DinnerStart = d.Kitchen.DinnerStart
	}
	if strings.TrimSpace(b.Kitchen.DinnerEnd) == "" {
		b.Kitchen.DinnerEnd = d.Kitchen.DinnerEnd
	}
	if b.Stories.MaxMin <= 0 {
		b.Stories.MaxMin = d.Stories.MaxMin
	}
	if b.Pomodoro.WorkMin <= 0 {
		b.Pomodoro.WorkMin = d.Pomodoro.WorkMin
	}
	if b.Pomodoro.BreakMin <= 0 {
		b.Pomodoro.BreakMin = d.Pomodoro.BreakMin
	}
}

func ValidateBehaviors(b Behaviors) error {
	b.FillDefaults()
	if err := requireHHMM(b.MorningBrief.At, "morning_brief.at"); err != nil {
		return err
	}
	if err := validDays(b.MorningBrief.Days); err != nil {
		return err
	}
	if b.MorningBrief.MaxSeconds < 15 || b.MorningBrief.MaxSeconds > 120 {
		return fmt.Errorf("morning_brief.max_seconds must be 15–120")
	}
	if b.Remember.MaxItems < 10 || b.Remember.MaxItems > 2000 {
		return fmt.Errorf("remember.max_items must be 10–2000")
	}
	if b.Focus.CooldownMin < 1 || b.Focus.CooldownMin > 180 {
		return fmt.Errorf("focus.cooldown_min must be 1–180")
	}
	if b.Kids.SessionMin < 5 || b.Kids.SessionMin > 180 {
		return fmt.Errorf("kids.session_min must be 5–180")
	}
	for _, pair := range [][3]string{
		{b.Kitchen.LunchStart, b.Kitchen.LunchEnd, "kitchen lunch"},
		{b.Kitchen.DinnerStart, b.Kitchen.DinnerEnd, "kitchen dinner"},
	} {
		if err := requireHHMM(pair[0], pair[2]+" start"); err != nil {
			return err
		}
		if err := requireHHMM(pair[1], pair[2]+" end"); err != nil {
			return err
		}
		if pair[0] == pair[1] {
			return fmt.Errorf("%s start and end must differ", pair[2])
		}
	}
	if b.HomeAssistant.Enabled {
		u := strings.TrimSpace(b.HomeAssistant.URL)
		if u == "" || !(strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")) {
			return fmt.Errorf("home_assistant.url must be http(s) when enabled")
		}
	}
	if b.Stories.MaxMin < 1 || b.Stories.MaxMin > 60 {
		return fmt.Errorf("stories.max_min must be 1–60")
	}
	if b.Pomodoro.WorkMin < 5 || b.Pomodoro.WorkMin > 90 {
		return fmt.Errorf("pomodoro.work_min must be 5–90")
	}
	if b.Pomodoro.BreakMin < 1 || b.Pomodoro.BreakMin > 30 {
		return fmt.Errorf("pomodoro.break_min must be 1–30")
	}
	switch strings.ToLower(strings.TrimSpace(b.Wearables.Provider)) {
	case "", "none", "oura", "whoop", "garmin":
	default:
		return fmt.Errorf("wearables.provider must be none, oura, whoop, or garmin")
	}
	return validDays(b.MorningBrief.Days)
}

func requireHHMM(v, field string) error {
	if !behaviorsHHMM.MatchString(strings.TrimSpace(v)) {
		return fmt.Errorf("%s must be HH:MM", field)
	}
	return nil
}

func validDays(days []int) error {
	for _, d := range days {
		if d < 0 || d > 6 {
			return fmt.Errorf("days must be 0–6 (Sunday–Saturday)")
		}
	}
	return nil
}

// RedactedForAPI copies b with Home Assistant token cleared.
func (b Behaviors) RedactedForAPI() Behaviors {
	out := b
	out.HomeAssistant.Token = ""
	return out
}
