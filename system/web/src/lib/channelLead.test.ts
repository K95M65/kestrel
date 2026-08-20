import assert from "node:assert/strict";
import { channelLead } from "./channelLead.ts";

const s = channelLead();
assert.match(s, /Telegram/);
assert.match(s, /not iMessage/i);
assert.doesNotMatch(s, /OpenClaw|Hermes/);

console.log("channelLead: ok");
