import assert from "node:assert/strict";
import {
  OPEN_CAPS,
  capsFromSet,
  clampBehaviors,
  extrasFor,
  featureSupported,
  guideSteps,
  ownsEnter,
  presetRowsFor,
  proveActs,
  UNSUPPORTED,
} from "./guideWalk.ts";
import { bodyCopy, inferredBodyId } from "./bodyProfile.ts";

const lamp = capsFromSet(new Set(["audio", "vision", "motion", "light", "expression", "presence", "media"]));
const reachy = capsFromSet(new Set(["audio", "vision", "motion", "expression", "media"]));
const intern = capsFromSet(new Set(["audio", "light", "media"]));

const lampSteps = guideSteps(lamp);
const reachySteps = guideSteps(reachy);
const internSteps = guideSteps(intern);

assert.deepEqual(lampSteps, ["intro", "name", "talk", "see", "prove", "preset", "mornings", "extras", "done"]);
assert.deepEqual(reachySteps, lampSteps, "Reachy and Lamp share the same walk when both have cam+motion+audio");
assert.ok(guideSteps(reachy, true).includes("connect"));
assert.ok(!guideSteps(reachy, false).includes("connect"));
assert.ok(!internSteps.includes("see"), "Intern has no camera — skip enroll");
assert.ok(internSteps.includes("prove"), "Intern has a ring — prove the body");
assert.ok(!internSteps.includes("mornings") === false, "Intern has audio — mornings stay");
assert.deepEqual(internSteps, ["intro", "name", "talk", "prove", "preset", "mornings", "extras", "done"]);

const ROWS = [
  { label: "Morning brief" },
  { label: "Greet everyone" },
  { label: "Mail / calendar" },
  { label: "Dance" },
  { label: "Stories" },
  { label: "Focus / pomodoro" },
];
assert.ok(presetRowsFor(reachy, ROWS).some((r) => r.label === "Dance"));
assert.ok(!presetRowsFor(intern, ROWS).some((r) => r.label === "Dance"));
assert.ok(!presetRowsFor(intern, ROWS).some((r) => r.label === "Greet everyone"));
assert.equal(presetRowsFor(lamp, ROWS).length, ROWS.length);
assert.deepEqual(extrasFor(lamp).map((x) => x.key), extrasFor(reachy).map((x) => x.key));
assert.deepEqual(presetRowsFor(lamp, ROWS).map((r) => r.label), presetRowsFor(reachy, ROWS).map((r) => r.label));

assert.ok(extrasFor(intern).every((x) => x.key !== "dance" && x.key !== "look" && x.key !== "greeter"));
assert.ok(extrasFor(reachy).some((x) => x.key === "dance"));
assert.ok(extrasFor(reachy).some((x) => x.key === "look"));

assert.equal(featureSupported("dance", intern), false);
assert.equal(featureSupported("dance", lamp), true);
assert.equal(featureSupported("look", intern), false);
assert.equal(featureSupported("greeter", intern), false);
assert.equal(featureSupported("greeter", reachy), true); // vision
assert.equal(featureSupported("radio", reachy), true);
assert.equal(featureSupported("radio", intern), false);
assert.equal(UNSUPPORTED, "Your current hardware does not support this feature");

assert.equal(inferredBodyId("reachy-mini", reachy), "reachy-mini");
assert.equal(inferredBodyId("", reachy), "reachy-mini");
assert.equal(inferredBodyId("", lamp), "lamp");
assert.equal(inferredBodyId("", intern), "intern-v2");

const rc = bodyCopy("reachy-mini", reachy);
assert.match(rc.introLead, /ears/i);
assert.doesNotMatch(rc.introLead, /ring/i);
assert.match(rc.sleep, /ears fold/i);

const lc = bodyCopy("lamp", lamp);
assert.match(lc.introLead, /ring/i);
assert.match(lc.sleep, /limp/i);

const ic = bodyCopy("intern-v2", intern);
assert.doesNotMatch(ic.introLead, /let it see you/i);
assert.match(ic.proveLead, /ring/i);

assert.equal(ownsEnter("talk"), true);
assert.equal(ownsEnter("name"), false);
assert.deepEqual(guideSteps(OPEN_CAPS), lampSteps);

assert.deepEqual(proveActs(lamp), ["light", "motion", "emotion"]);
assert.deepEqual(proveActs(reachy), ["motion", "emotion"]);
assert.deepEqual(proveActs(intern), ["light"]);

assert.equal(inferredBodyId("", lamp, false), "");
assert.equal(bodyCopy("", lamp, false).introTitle, "Let's try this robot");
assert.doesNotMatch(bodyCopy("", lamp, false).introLead, /the ring is its face/i);
assert.match(lc.welcomeLine, /ring/i);
assert.match(rc.welcomeLine, /camera/i);
assert.doesNotMatch(rc.welcomeLine, /ring/i);
assert.match(ic.welcomeLine, /talk/i);
assert.doesNotMatch(ic.welcomeLine, /camera/i);
assert.match(rc.talkLead("Buddy"), /head and ears/i);
assert.match(lc.talkLead("Buddy"), /ring/i);
assert.match(rc.faceLead, /desk robot/i);
assert.match(lc.faceLead, /desk lamp/i);

const internOn = clampBehaviors({
  dance: { enabled: true },
  look: { enabled: true },
  greeter: { enabled: true },
  focus: { enabled: true },
  stories: { enabled: true },
  pomodoro: { enabled: true },
  morning_brief: { enabled: true },
  doa: { enabled: true },
  layered_motion: { enabled: true },
  marionette: { enabled: true },
  radio: { enabled: true },
  hand_track: { enabled: true },
  telepresence: { enabled: true },
  presence: { idle_motion: true },
}, intern);
assert.equal(internOn.dance.enabled, false);
assert.equal(internOn.look.enabled, false);
assert.equal(internOn.greeter.enabled, false);
assert.equal(internOn.stories.enabled, true);
assert.equal(internOn.presence.idle_motion, true); // ring counts as presence
assert.equal(internOn.radio.enabled, false);

const reachyOn = clampBehaviors({
  dance: { enabled: true },
  look: { enabled: true },
  greeter: { enabled: true },
  presence: { idle_motion: true },
}, reachy);
assert.equal(reachyOn.dance.enabled, true);
assert.equal(reachyOn.look.enabled, true);

assert.equal(featureSupported("privacy", intern), false);
assert.equal(featureSupported("privacy", reachy), true);

console.log("guideWalk: ok");
