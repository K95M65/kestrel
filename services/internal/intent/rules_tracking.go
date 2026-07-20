// Vision-tracking rules — servo follow. Going through OpenClaw costs ~3-5s
// per "track the cup" command; local match → direct POST /servo/track keeps
// the latency under ~100ms so the camera starts following before the user has
// finished their next breath.
package intent

import (
	"fmt"
	"strings"

	"go.autonomous.ai/os/internal/device"
)

var trackingRules = []rule{
	{
		name:       "servo_track_stop",
		capability: device.CapMotion,
		match:      anyOf("stop tracking", "stop following", "stop watching", "stop track"),
		exec: func(string) *Result {
			post("/servo/track/stop", "")
			return &Result{TTSText: "Stopped tracking.", Actions: []string{"POST /servo/track/stop"}}
		},
	},
	{
		name:       "servo_track",
		capability: device.CapMotion,
		match: func(t string) bool {
			if !hasTrackVerb(t) {
				return false
			}
			return extractTrackTarget(t) != ""
		},
		exec: func(t string) *Result {
			target := extractTrackTarget(t)
			body := fmt.Sprintf(`{"target":["%s"]}`, target)
			post("/servo/track", body)
			return &Result{
				TTSText: fmt.Sprintf("Tracking %s.", target),
				Actions: []string{"POST /servo/track " + body},
			}
		},
	},
}

// hasTrackVerb returns true when the command contains a tracking verb. Phrased
// to require the verb early in the sentence so "I'd like to follow up on..." or
// "I can't watch that movie" don't trigger the camera.
func hasTrackVerb(t string) bool {
	for _, kw := range []string{"track ", "follow ", "watch "} {
		if strings.HasPrefix(t, kw) || strings.Contains(t, " "+kw) {
			return true
		}
	}
	return false
}

// trackTargets maps spoken nouns to the label sent to /servo/track. Pronouns
// resolve to "person"; ambiguous words ("me", "yourself") are explicit so a
// stray "watch yourself" still picks the right target.
var trackTargets = []struct {
	keywords []string
	label    string
}{
	{[]string{"face", "my face", "the face"}, "face"},
	{[]string{"hand", "my hand", "the hand"}, "hand"},
	{[]string{"me", "myself", "user", "us"}, "person"},
	{[]string{"person", "people", "human", "man", "woman", "the guy", "that guy"}, "person"},
	{[]string{"dog"}, "dog"},
	{[]string{"cat"}, "cat"},
	{[]string{"bird"}, "bird"},
	{[]string{"cup", "mug", "coffee cup"}, "cup"},
	{[]string{"bottle", "water bottle"}, "bottle"},
	{[]string{"phone", "smartphone", "cell phone", "mobile"}, "cell phone"},
	{[]string{"book"}, "book"},
	{[]string{"remote", "tv remote"}, "remote"},
	{[]string{"laptop", "computer"}, "laptop"},
	{[]string{"keyboard"}, "keyboard"},
	{[]string{"mouse"}, "mouse"},
	{[]string{"teddy", "teddy bear", "stuffed animal"}, "teddy bear"},
	{[]string{"ball"}, "sports ball"},
	{[]string{"backpack", "bag"}, "backpack"},
	{[]string{"chair"}, "chair"},
	{[]string{"clock"}, "clock"},
	{[]string{"scissors"}, "scissors"},
	{[]string{"banana"}, "banana"},
	{[]string{"apple"}, "apple"},
	{[]string{"orange"}, "orange"},
}

// extractTrackTarget pulls the COCO/face label from a tracking command. Returns
// "" when nothing matches — caller should then fall through to OpenClaw which
// can use YOLOWorld open-vocab for less common nouns.
func extractTrackTarget(t string) string {
	for _, e := range trackTargets {
		for _, kw := range e.keywords {
			if strings.Contains(t, kw) {
				return e.label
			}
		}
	}
	return ""
}
