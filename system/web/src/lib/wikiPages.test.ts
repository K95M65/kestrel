import assert from "node:assert/strict";
import { readFileSync, readdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const wikiDir = join(here, "../../../../docs/wiki");
const files = readdirSync(wikiDir).filter((f) => f.endsWith(".md") && f !== "README.md");
const catalog = readFileSync(join(here, "wiki.ts"), "utf8");

assert.ok(files.includes("brains.md"), "docs/wiki/brains.md must exist");
assert.match(catalog, /slug: "brains"/);

const brains = readFileSync(join(wikiDir, "brains.md"), "utf8");
assert.match(brains, /OpenClaw/);
assert.match(brains, /coding CLI/i);
assert.match(brains, /MEMORY\.md/);
assert.match(brains, /Advanced/);

for (const f of files) {
  const slug = f.replace(/\.md$/, "");
  assert.match(catalog, new RegExp(`slug: "${slug}"`), `${f} must be in wiki.ts`);
}

console.log("wikiPages: ok");
