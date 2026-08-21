package skillcontext

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.autonomous.ai/os/system/server/config"
)

var (
	behaviorsMu sync.RWMutex
	behaviorsFn func() config.Behaviors
)

// SetBehaviorsSource registers the live companion pack. device.ProvideService
// installs it; tests may swap it.
func SetBehaviorsSource(fn func() config.Behaviors) {
	behaviorsMu.Lock()
	behaviorsFn = fn
	behaviorsMu.Unlock()
}

// CurrentBehaviors returns the live pack or compiled defaults.
func CurrentBehaviors() config.Behaviors {
	behaviorsMu.RLock()
	fn := behaviorsFn
	behaviorsMu.RUnlock()
	if fn == nil {
		return config.DefaultBehaviors()
	}
	b := fn()
	b.FillDefaults()
	return b
}

// behaviorsDigest is the compact block skills read. Secrets never go here.
type behaviorsDigest struct {
	MorningBrief        bool   `json:"morning_brief"`
	Remember            bool   `json:"remember"`
	Dance               bool   `json:"dance"`
	DanceQuery          string `json:"dance_query,omitempty"`
	DraftNotSend        bool   `json:"draft_not_send"`
	Ask                 string `json:"ask,omitempty"`
	Room                string `json:"room,omitempty"`
	OwnerEmail          string `json:"owner_email,omitempty"`
	Claimed             bool   `json:"claimed,omitempty"`
	CameraOnDemand      bool   `json:"camera_on_demand"`
	FaceFollowAfterWake bool   `json:"face_follow_after_wake"`
	IdleMotion          bool   `json:"idle_motion"`
	DoA                 bool   `json:"doa"`
	LayeredMotion       bool   `json:"layered_motion"`
	Focus               bool   `json:"focus"`
	PhoneNag            bool   `json:"phone_nag"`
	FocusCooldownMin    int    `json:"focus_cooldown_min,omitempty"`
	Kids                bool   `json:"kids"`
	KidsSessionMin      int    `json:"kids_session_min,omitempty"`
	Greeter             bool   `json:"greeter"`
	GreeterNamedOnly    bool   `json:"greeter_named_only"`
	Look                bool   `json:"look"`
	Kitchen             bool   `json:"kitchen"`
	LunchStart          string `json:"lunch_start,omitempty"`
	LunchEnd            string `json:"lunch_end,omitempty"`
	DinnerStart         string `json:"dinner_start,omitempty"`
	DinnerEnd           string `json:"dinner_end,omitempty"`
	HomeAssistant       bool   `json:"home_assistant"`
	HAURL               string `json:"ha_url,omitempty"`
	Marionette          bool   `json:"marionette"`
	Weather             bool   `json:"weather"`
	TimeTool            bool   `json:"time_tool"`
	Search              bool   `json:"search"`
	HandTrack           bool   `json:"hand_track"`
	Radio               bool   `json:"radio"`
	Telepresence        bool   `json:"telepresence"`
	Stories             bool   `json:"stories"`
	StoriesMaxMin       int    `json:"stories_max_min,omitempty"`
	Pomodoro            bool   `json:"pomodoro"`
	Wearables           bool   `json:"wearables"`
	WearableProvider    string `json:"wearable_provider,omitempty"`
}

// BuildBehaviorsContext is the `[behaviors: {...}]` block every user-facing
// turn should carry so skills can honor the dashboard pack without a tool call.
func BuildBehaviorsContext() string {
	b := CurrentBehaviors()
	h := CurrentHousehold()
	kids := b.Kids.Enabled || KidsBound()
	d := behaviorsDigest{
		MorningBrief:        b.MorningBrief.Enabled,
		Remember:            b.Remember.Enabled,
		Dance:               b.Dance.Enabled,
		DanceQuery:          b.Dance.DefaultQuery,
		DraftNotSend:        b.Connectors.DraftNotSend,
		Ask:                 b.Connectors.Ask,
		Room:                h.Room,
		OwnerEmail:          h.OwnerEmail,
		Claimed:             h.Claimed,
		CameraOnDemand:      b.Privacy.CameraOnDemand,
		FaceFollowAfterWake: b.Privacy.FaceFollowAfterWake,
		IdleMotion:          b.Presence.IdleMotion,
		DoA:                 b.DoA.Enabled,
		LayeredMotion:       b.LayeredMotion.Enabled,
		Focus:               b.Focus.Enabled,
		PhoneNag:            b.Focus.PhoneNag,
		FocusCooldownMin:    b.Focus.CooldownMin,
		Kids:                kids,
		KidsSessionMin:      b.Kids.SessionMin,
		Greeter:             b.Greeter.Enabled,
		GreeterNamedOnly:    b.Greeter.NamedOnly,
		Look:                b.Look.Enabled,
		Kitchen:             b.Kitchen.Enabled,
		LunchStart:          b.Kitchen.LunchStart,
		LunchEnd:            b.Kitchen.LunchEnd,
		DinnerStart:         b.Kitchen.DinnerStart,
		DinnerEnd:           b.Kitchen.DinnerEnd,
		HomeAssistant:       b.HomeAssistant.Enabled,
		HAURL:               b.HomeAssistant.URL,
		Marionette:          b.Marionette.Enabled,
		Weather:             b.Tools.Weather,
		TimeTool:            b.Tools.Time,
		Search:              b.Tools.Search,
		HandTrack:           b.HandTrack.Enabled,
		Radio:               b.Radio.Enabled,
		Telepresence:        b.Telepresence.Enabled,
		Stories:             b.Stories.Enabled,
		StoriesMaxMin:       b.Stories.MaxMin,
		Pomodoro:            b.Pomodoro.Enabled,
		Wearables:           b.Wearables.Enabled,
		WearableProvider:    b.Wearables.Provider,
	}
	body, err := json.Marshal(d)
	if err != nil {
		return ""
	}
	return "\n[behaviors: " + string(body) + "]"
}

func kitchenWindowMinutes() (lunchStart, lunchEnd, dinnerStart, dinnerEnd int, custom bool) {
	b := CurrentBehaviors()
	if !b.Kitchen.Enabled {
		return 0, 0, 0, 0, false
	}
	ls, ok1 := hhmmMinutes(b.Kitchen.LunchStart)
	le, ok2 := hhmmMinutes(b.Kitchen.LunchEnd)
	ds, ok3 := hhmmMinutes(b.Kitchen.DinnerStart)
	de, ok4 := hhmmMinutes(b.Kitchen.DinnerEnd)
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return 0, 0, 0, 0, false
	}
	return ls, le, ds, de, true
}

func hhmmMinutes(s string) (int, bool) {
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
		return 0, false
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// KitchenWindowFor is the kitchen-pack override of the historic 11:30–13:30 /
// 18:30–20:30 meal windows. ok is false when kitchen is off or times are bad.
func KitchenWindowFor(now time.Time) (window string, ok bool) {
	ls, le, ds, de, custom := kitchenWindowMinutes()
	if !custom {
		return "", false
	}
	mins := now.Hour()*60 + now.Minute()
	if inWrap(mins, ls, le) {
		return "lunch", true
	}
	if inWrap(mins, ds, de) {
		return "dinner", true
	}
	return "", true
}

func inWrap(now, start, end int) bool {
	if start == end {
		return false
	}
	if start < end {
		return now >= start && now < end
	}
	return now >= start || now < end
}
