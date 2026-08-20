import assert from "node:assert/strict";
import { speakerMutedFromVoice } from "./speakerMute.ts";

assert.equal(speakerMutedFromVoice(undefined), undefined);
assert.equal(speakerMutedFromVoice(null), undefined);
assert.equal(speakerMutedFromVoice({}), undefined);
assert.equal(speakerMutedFromVoice({ speaker_muted: true }), true);
assert.equal(speakerMutedFromVoice({ speaker_muted: false }), false);

console.log("speakerMute: ok");
