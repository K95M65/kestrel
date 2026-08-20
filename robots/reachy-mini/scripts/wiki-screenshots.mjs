// Capture product screenshots for the Reachy Mini README / wiki.
// Usage (from repo, Vite proxied to the robot, admin password in env):
//   ADMIN_PASSWORD=… BASE=http://127.0.0.1:5173 OUT=robots/reachy-mini/images/app \
//     npx --yes playwright-core node robots/reachy-mini/scripts/wiki-screenshots.mjs
import { chromium } from "playwright-core";
import fs from "node:fs";
import path from "node:path";

const BASE = process.env.BASE || "http://127.0.0.1:5173";
const OUT = process.env.OUT || path.resolve("robots/reachy-mini/images/app");
const PASSWORD = process.env.ADMIN_PASSWORD || "";

if (!PASSWORD) {
  console.error("ADMIN_PASSWORD is required");
  process.exit(1);
}

fs.mkdirSync(OUT, { recursive: true });

const browser = await chromium.launch({ channel: "chrome", headless: true });
const page = await browser.newPage({
  viewport: { width: 1440, height: 900 },
  colorScheme: "light",
});

async function shot(name) {
  const dest = path.join(OUT, name);
  await page.waitForTimeout(400);
  await page.screenshot({ path: dest, fullPage: false });
  console.log("wrote", dest);
}

await page.goto(`${BASE}/login`, { waitUntil: "networkidle" });
await page.waitForSelector("text=Sign in");
await shot("01-login.png");

await page.locator("input[type='password']").fill(PASSWORD);
await page.getByRole("button", { name: "Sign in" }).click();
await page.waitForURL(/\/monitor/, { timeout: 20000 });
await page.waitForTimeout(1200);
await shot("02-home.png");

await page.goto(`${BASE}/monitor#chat`, { waitUntil: "networkidle" });
await page.waitForSelector("text=Chat with");
await page.waitForTimeout(600);
await shot("03-talk.png");

await page.goto(`${BASE}/monitor#face-owners`, { waitUntil: "networkidle" });
await page.waitForSelector("text=Who");
await page.waitForTimeout(800);
await shot("04-people.png");

await page.goto(`${BASE}/monitor#camera`, { waitUntil: "networkidle" });
await page.waitForSelector("text=What the robot sees");
await page.waitForTimeout(800);
await shot("05-camera.png");

await page.goto(`${BASE}/setting#general`, { waitUntil: "networkidle" });
await page.waitForSelector("text=General");
await page.waitForTimeout(800);
await shot("06-general.png");

await page.goto(`${BASE}/setting#behaviors`, { waitUntil: "networkidle" });
await page.waitForSelector("text=Start guided setup");
await page.getByRole("button", { name: "Start guided setup" }).click();
await page.waitForSelector("text=Let's try this robot");
await shot("07-guide-intro.png");
await page.getByRole("button", { name: "Start", exact: true }).click();
await page.waitForSelector("text=What should we call");
await shot("08-guide-name.png");
await page.locator("#guide-robot-name").fill("Buddy");
await page.getByRole("button", { name: "Next" }).click();
await page.waitForSelector("text=Talk to Buddy");
await shot("09-guide-talk.png");
await page.getByRole("button", { name: "Next" }).click();
await page.waitForSelector("text=Who is this?");
await page.waitForTimeout(800);
await shot("10-guide-see.png");

await browser.close();
console.log("done");
