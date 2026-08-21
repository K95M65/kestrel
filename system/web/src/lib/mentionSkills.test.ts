import assert from "node:assert/strict";
import { applySkillMention, filterSkills, mentionQuery } from "./mentionSkills.ts";

assert.equal(applySkillMention("hello"), "hello");
assert.equal(applySkillMention("@news what happened"), "[use-skill: news] what happened");
assert.deepEqual(mentionQuery("please @ne", 10), { at: 7, q: "ne" });
assert.equal(mentionQuery("hello there"), null);

const hits = filterSkills(
  [{ name: "news" }, { name: "music" }, { name: "skill-creator" }],
  "ne",
);
assert.equal(hits.length, 1);
assert.equal(hits[0].name, "news");
console.log("mentionSkills: ok");
