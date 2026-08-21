package skillcontext

import (
	"sync"

	"go.autonomous.ai/os/system/lib/usercanon"
	"go.autonomous.ai/os/system/server/config"
	"go.autonomous.ai/os/system/skillcontext/mood"
)

var (
	householdMu sync.RWMutex
	householdFn func() config.Household
)

// SetHouseholdSource registers the live household. device.ProvideService installs it.
func SetHouseholdSource(fn func() config.Household) {
	householdMu.Lock()
	householdFn = fn
	householdMu.Unlock()
}

// CurrentHousehold returns the live household or empty.
func CurrentHousehold() config.Household {
	householdMu.RLock()
	fn := householdFn
	householdMu.RUnlock()
	if fn == nil {
		return config.Household{}
	}
	return fn()
}

// RoleForLabel is the household role of a People slug, or empty.
func RoleForLabel(label string) string {
	return CurrentHousehold().RoleOf(label)
}

// KidsBound reports whether this turn should refuse mail/calendar/computer-use:
// the kids pack is on, or the person in front of the camera is a kid.
func KidsBound() bool {
	if CurrentBehaviors().Kids.Enabled {
		return true
	}
	u := mood.CurrentUser()
	if u == "" || u == usercanon.DefaultUser {
		return false
	}
	return RoleForLabel(u) == config.RoleKid
}
