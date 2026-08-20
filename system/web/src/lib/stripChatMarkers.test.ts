import assert from "node:assert/strict";
import { isInternalAgentDoc, stripChatMarkers } from "./stripChatMarkers.ts";

assert.equal(
  stripChatMarkers("Hi, I'm Bobert [excited]"),
  "Hi, I'm Bobert",
);
assert.equal(
  stripChatMarkers('Hello /emotion name="happy" intensity="0.8" there'),
  "Hello there",
);
assert.equal(
  stripChatMarkers('[HW:/emotion:{"emotion":"curious","intensity":0.7}] Looking at you.'),
  "Looking at you.",
);
assert.equal(
  stripChatMarkers("[Lights off right away!](HW:/led/off:{}) Done."),
  "Lights off right away! Done.",
);
assert.equal(
  stripChatMarkers("[HW:/led/off](HW:/led/solid:{})"),
  "",
);
assert.equal(
  stripChatMarkers("I logged it in [MEMORY.md](/root/.openclaw/workspace/MEMORY.md)."),
  "I logged it in.",
);
assert.equal(
  stripChatMarkers("See /root/.openclaw/workspace/SKILL.md for that."),
  "See for that.",
);
assert.equal(
  stripChatMarkers("Keep this photo /root/.openclaw/media/hal-snapshots/snap_1.jpg please."),
  "Keep this photo /root/.openclaw/media/hal-snapshots/snap_1.jpg please.",
);
assert.equal(
  stripChatMarkers("[thinking] One moment.\n\nNow I have it."),
  "One moment.\n\nNow I have it.",
);
assert.equal(stripChatMarkers("/emotion"), "");
assert.equal(isInternalAgentDoc("/root/.openclaw/workspace/MEMORY.md"), true);
assert.equal(isInternalAgentDoc("/root/.openclaw/workspace/skills/foo/SKILL.md"), true);
assert.equal(isInternalAgentDoc("/root/.openclaw/media/hal-snapshots/snap_1.jpg"), false);
assert.equal(isInternalAgentDoc("/root/.openclaw/workspace/notes-MEMORY.md"), false);

console.log("stripChatMarkers: ok");
