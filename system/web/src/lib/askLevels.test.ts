import assert from "node:assert/strict";
import { claimUrl, draftFromAsk, normalizeAsk } from "./askLevels.ts";

assert.equal(normalizeAsk(undefined, true), "important_actions");
assert.equal(normalizeAsk("", false), "never_ask");
assert.equal(normalizeAsk("ALWAYS_ASK"), "always_ask");
assert.equal(draftFromAsk("never_ask"), false);
assert.equal(draftFromAsk("important_actions"), true);
assert.equal(claimUrl("http://10.10.2.160:8080", "12345678"), "http://10.10.2.160:8080/claim?pin=12345678");
console.log("askLevels: ok");
