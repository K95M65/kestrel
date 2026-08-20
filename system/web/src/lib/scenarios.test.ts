import assert from "node:assert/strict";
import { capsFromSet } from "./guideWalk.ts";
import {
  SCENARIOS,
  SCENARIO_MCP,
  scenarioFor,
  scenarioStatus,
  scenarioStatusLabel,
  scenarioSteps,
  scenariosFor,
  scenariosMatching,
} from "./scenarios.ts";

const reachy = capsFromSet(new Set([
  "audio", "vision", "motion", "expression", "media", "companion", "presence",
]));
const intern = capsFromSet(new Set(["audio", "light", "media", "companion"]));
const speaker = capsFromSet(new Set(["audio"]));

assert.equal(SCENARIOS.length, 8);
assert.deepEqual(SCENARIOS.map((s) => s.id), [
  "chat", "web", "news", "music", "spotify", "stories", "look", "dance",
]);
assert.equal(scenarioFor("chat")?.services[0]?.id, "telegram");
assert.equal(scenarioFor("web")?.buddy, "required");
assert.equal(scenarioFor("spotify")?.buddy, "required");
assert.equal(scenarioFor("music")?.buddy, "off");
assert.equal(scenarioFor("missing"), null);

assert.ok(scenariosFor(reachy).some((s) => s.id === "music"));
assert.ok(scenariosFor(reachy).some((s) => s.id === "web"));
assert.ok(scenariosFor(intern).some((s) => s.id === "music"));
assert.ok(scenariosFor(intern).some((s) => s.id === "web"), "Intern declares companion");
assert.ok(!scenariosFor(speaker).some((s) => s.id === "music"), "music needs media");
assert.ok(scenariosFor(speaker).some((s) => s.id === "news"));
assert.ok(!scenariosFor(speaker).some((s) => s.id === "web"));
assert.ok(scenariosFor(reachy).some((s) => s.id === "look"));
assert.ok(scenariosFor(reachy).some((s) => s.id === "dance"));
assert.ok(scenariosFor(intern).some((s) => s.id === "stories"));
assert.ok(!scenariosFor(intern).some((s) => s.id === "look"));
assert.ok(!scenariosFor(intern).some((s) => s.id === "dance"));
assert.ok(scenariosMatching("spotify").some((s) => s.id === "spotify"));
assert.ok(scenariosMatching("windows").some((s) => s.id === "web"));
assert.ok(scenariosMatching("telegram").some((s) => s.id === "chat"));
assert.ok(scenariosMatching("weather").some((s) => s.id === "news"));
assert.ok(scenariosMatching("bedtime").some((s) => s.id === "stories"));
assert.equal(scenariosMatching("xyzzy").length, 0);

assert.deepEqual(scenarioSteps(scenarioFor("music")!), ["intro", "try", "done"]);
assert.ok(scenarioSteps(scenarioFor("chat")!).includes("connect"));
assert.ok(scenarioSteps(scenarioFor("web")!).includes("buddy"));
assert.ok(scenarioSteps(scenarioFor("news")!).includes("tools"));
assert.ok(!scenarioSteps(scenarioFor("news")!).includes("buddy"));

const ready = {
  caps: reachy,
  kids: false,
  telegram: true,
  buddyPaired: true,
  toolsOn: true,
};
assert.equal(scenarioStatus(scenarioFor("chat")!, ready), "ready");
assert.equal(scenarioStatus(scenarioFor("chat")!, { ...ready, telegram: false }), "setup");
assert.equal(scenarioStatus(scenarioFor("web")!, { ...ready, buddyPaired: false }), "needs-computer");
assert.equal(scenarioStatusLabel("needs-computer"), "Needs a computer");
assert.equal(scenarioStatus(scenarioFor("spotify")!, { ...ready, kids: true }), "kids");
assert.equal(scenarioStatus(scenarioFor("web")!, { ...ready, kids: true }), "kids");
assert.equal(scenarioStatus(scenarioFor("music")!, { ...ready, kids: true }), "ready");
assert.equal(
  scenarioStatus(scenarioFor("web")!, { ...ready, caps: speaker }),
  "unsupported",
);
assert.equal(scenarioStatus(scenarioFor("news")!, { ...ready, toolsOn: false }), "setup");

assert.ok(scenarioFor("news")!.mcp.some((m) => m.url === SCENARIO_MCP.weather.url));
assert.match(scenarioFor("music")!.honest, /YouTube/i);
assert.match(scenarioFor("spotify")!.honest, /computer/i);
assert.ok(!scenarioFor("chat")!.why.toLowerCase().includes("hermes"));
assert.ok(!scenarioFor("chat")!.why.toLowerCase().includes("openclaw"));
assert.equal(scenarioFor("stories")!.flip?.stories, true);
assert.equal(scenarioFor("look")!.flip?.look, true);
assert.equal(scenarioFor("dance")!.flip?.dance, true);
assert.match(scenarioFor("news")!.tryPrompt, /Paris/i);

console.log("scenarios: ok");
