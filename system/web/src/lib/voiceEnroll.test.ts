import assert from "node:assert/strict";
import { canStartVoiceEnroll, voiceEnrollTarget } from "./voiceEnroll.ts";

assert.equal(canStartVoiceEnroll(""), false);
assert.equal(canStartVoiceEnroll("   "), false);
assert.equal(canStartVoiceEnroll(null), false);
assert.equal(canStartVoiceEnroll(undefined), false);
assert.equal(canStartVoiceEnroll("Leo"), true);

assert.equal(voiceEnrollTarget("  Leo  "), "leo");
assert.throws(() => voiceEnrollTarget(""), /Pick a person first/);
assert.throws(() => voiceEnrollTarget(null), /Pick a person first/);

console.log("voiceEnroll: ok");
