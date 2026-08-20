// Capture product screenshots for the Kestrel README.
// Usage (from autonomous-os/, Vite proxied to the robot):
//   LAMP_PROXY=http://10.10.2.160 npm --prefix system/web run dev
//   OS_SESSION_FILE=/tmp/kestrel-os-session \
//     BASE=http://127.0.0.1:5173 OUT=robots/reachy-mini/images/app \
//     npx --yes playwright-core node robots/reachy-mini/scripts/wiki-screenshots.mjs
//
// Auth: OS_SESSION cookie (httpOnly session) or ADMIN_PASSWORD.
// Guided setup is walked for screenshots then closed with X — never Finish.
import { chromium } from "playwright-core";
import fs from "node:fs";
import path from "node:path";

const BASE = process.env.BASE || "http://127.0.0.1:5173";
const OUT = process.env.OUT || path.resolve("robots/reachy-mini/images/app");
const PASSWORD = process.env.ADMIN_PASSWORD || "";
const SESSION =
  process.env.OS_SESSION ||
  (process.env.OS_SESSION_FILE && fs.existsSync(process.env.OS_SESSION_FILE)
    ? fs.readFileSync(process.env.OS_SESSION_FILE, "utf8").trim()
    : "");

if (!PASSWORD && !SESSION) {
  console.error("ADMIN_PASSWORD or OS_SESSION(_FILE) is required");
  process.exit(1);
}

fs.mkdirSync(OUT, { recursive: true });

const browser = await chromium.launch({ channel: "chrome", headless: true });

async function shot(page, name) {
  const dest = path.join(OUT, name);
  await page.waitForTimeout(500);
  await page.screenshot({ path: dest, fullPage: false });
  console.log("wrote", dest);
}

const loginPage = await browser.newPage({
  viewport: { width: 1440, height: 900 },
  colorScheme: "light",
});
await loginPage.goto(`${BASE}/login`, { waitUntil: "networkidle" });
await loginPage.waitForSelector("text=Sign in");
await shot(loginPage, "01-login.png");
await loginPage.close();

const context = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  colorScheme: "light",
});
if (SESSION) {
  await context.addCookies([
    { name: "os_session", value: SESSION, url: BASE },
  ]);
}
const page = await context.newPage();
const hideToasts = async () => {
  await page.addStyleTag({
    content: "[data-sonner-toaster],[data-sonner-toast],.toaster{display:none!important}",
  }).catch(() => {});
};
page.on("load", () => { void hideToasts(); });

if (PASSWORD && !SESSION) {
  await page.goto(`${BASE}/login`, { waitUntil: "networkidle" });
  await page.locator("input[type='password']").fill(PASSWORD);
  await page.getByRole("button", { name: "Sign in" }).click();
  await page.waitForURL(/\/monitor/, { timeout: 20000 });
} else {
  await page.goto(`${BASE}/monitor`, { waitUntil: "networkidle" });
  await page.waitForURL(/\/monitor/, { timeout: 20000 });
}
await page.waitForTimeout(1200);
await shot(page, "02-home.png");

await page.goto(`${BASE}/monitor#chat`, { waitUntil: "networkidle" });
await page.waitForSelector("text=Chat with");
await page.waitForTimeout(600);
await shot(page, "03-talk.png");

await page.goto(`${BASE}/monitor#face-owners`, { waitUntil: "networkidle" });
await page.waitForSelector("text=Who");
await page.waitForTimeout(800);
await shot(page, "04-people.png");

await page.goto(`${BASE}/monitor#camera`, { waitUntil: "networkidle" });
await page.waitForSelector("text=What the robot sees");
await page.waitForTimeout(800);
await shot(page, "05-camera.png");

await page.goto(`${BASE}/setting#general`, { waitUntil: "networkidle" });
await page.waitForSelector("text=General");
await page.waitForTimeout(800);
await shot(page, "06-general.png");

await page.goto(`${BASE}/setting#uses`, { waitUntil: "networkidle" });
await page.waitForSelector("text=What it can do");
await page.waitForTimeout(800);
await shot(page, "11-uses.png");

await page.goto(`${BASE}/setting#behaviors`, { waitUntil: "networkidle" });
await page.waitForSelector("text=Start guided setup");
await page.waitForTimeout(600);
await shot(page, "12-behaviors.png");

await page.getByRole("button", { name: "Start guided setup" }).click();
await page.waitForSelector("text=Let's try this");
await shot(page, "07-guide-intro.png");

await page.getByRole("button", { name: "Start", exact: true }).click();
await page.waitForSelector("text=What should we call");
await shot(page, "08-guide-name.png");

let currentName = "Buddy";
try {
  const cfg = await page.evaluate(async () => {
    const r = await fetch("/api/device/config");
    const j = await r.json();
    return j.data || {};
  });
  if (cfg.agent_name) currentName = String(cfg.agent_name);
} catch {
  /* keep default */
}
await page.locator("#guide-robot-name").fill(currentName);
await page.getByRole("button", { name: "Next" }).click();
await page.waitForSelector(`text=Talk to`);
await page.waitForTimeout(400);
await shot(page, "09-guide-talk.png");

await page.getByRole("button", { name: "Next" }).click();
await page.waitForTimeout(800);
await shot(page, "10-guide-see.png");

const nextBtn = page.getByRole("button", { name: "Next" });
if (await nextBtn.isVisible().catch(() => false)) {
  await nextBtn.click();
  await page.waitForTimeout(600);
  await shot(page, "14-guide-prove.png");
}
if (await nextBtn.isVisible().catch(() => false)) {
  await nextBtn.click();
  await page.waitForTimeout(600);
  await shot(page, "15-guide-preset.png");
  const justMe = page.getByRole("button", { name: /Just me/i });
  if (await justMe.isVisible().catch(() => false)) {
    await justMe.click();
    await page.waitForTimeout(400);
    await shot(page, "15-guide-preset.png");
  }
}
if (await nextBtn.isVisible().catch(() => false)) {
  await nextBtn.click();
  await page.waitForTimeout(500);
  await shot(page, "16-guide-life.png");
}

await page.getByRole("button", { name: "Close" }).click({ timeout: 2000 }).catch(async () => {
  await page.locator('[role="dialog"]').locator("button").first().click().catch(() => {});
  await page.keyboard.press("Escape");
});
await page.waitForTimeout(400);

await browser.close();
