import assert from "node:assert/strict";
import { isRobotQuiet, sleepToggleKind, sleepToggleLabel, withSleeping } from "./sleepToggle.ts";

assert.equal(isRobotQuiet(false), false);
assert.equal(isRobotQuiet(false, "happy"), false);
assert.equal(isRobotQuiet(true), true);
assert.equal(isRobotQuiet(true, "greeting"), true);
assert.equal(isRobotQuiet(false, "sleepy"), false);
assert.equal(isRobotQuiet(null, "sleepy"), true);
assert.equal(isRobotQuiet(undefined, " Sleepy "), true);
assert.equal(isRobotQuiet(undefined, undefined), false);

assert.equal(sleepToggleKind(false), "sleep");
assert.equal(sleepToggleKind(true), "wake");
assert.equal(sleepToggleLabel(false), "Sleep now");
assert.equal(sleepToggleLabel(true), "Wake now");

const seeded = withSleeping(null, true);
assert.equal(seeded.sleeping, true);
assert.equal(withSleeping(seeded, false).sleeping, false);

console.log("sleepToggle: ok");
