import assert from "node:assert/strict";
import { LIFE_RECIPES, lifeHasConnect, recipeFor } from "./lifeRecipes.ts";

assert.equal(lifeHasConnect("desk"), true);
assert.equal(lifeHasConnect("office"), true);
assert.equal(lifeHasConnect("family"), true);
assert.equal(lifeHasConnect("kids"), false);
assert.equal(lifeHasConnect(null), false);

const desk = recipeFor("desk")!;
assert.ok(desk.services.some((s) => s.id === "gmail"));
assert.ok(desk.services.some((s) => s.id === "google_calendar"));
assert.ok(desk.services.some((s) => s.id === "telegram"));
assert.equal(desk.policy.draft_not_send, true);
assert.equal(desk.policy.kids, false);

const family = recipeFor("family")!;
assert.ok(!family.services.some((s) => s.id === "gmail"));
assert.ok(family.services.some((s) => s.id === "google_calendar"));

const kids = recipeFor("kids")!;
assert.equal(kids.services.length, 0);
assert.equal(kids.policy.kids, true);

const cal = desk.services.find((s) => s.id === "google_calendar")!;
assert.equal(cal.auth, "ical");
const gmail = desk.services.find((s) => s.id === "gmail")!;
assert.equal(gmail.auth, "pat");

assert.equal(LIFE_RECIPES.office.buddy, "optional");
assert.equal(LIFE_RECIPES.kids.buddy, "off");

console.log("lifeRecipes: ok");
