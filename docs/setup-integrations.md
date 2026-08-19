# Setup integrations to add

Notes from the Reachy Mini install (2026-08-18). Product work for **setup**, not a live robot patch. Goal: the installer and dash should match OpenCode’s `/connect` — sign in with the account you already pay for, then the same login should work for whatever agent runtime you deploy (OpenClaw **or** Hermes).

Today setup is still **paste an API key + base URL + model** (`system/web/src/components/setup/LLMSection.tsx`). That is the gap.

---

## 1. Provider login (OpenCode-parity)

OpenCode stores credentials in `~/.local/share/opencode/auth.json` via `/connect`. Autonomous OS should own the same flows on a headless robot: **device-code / OAuth in the dash**, token file on disk, refresh in the background, then write whatever the chosen runtime needs.

Do this once, then **emit both**:

| Runtime | Where the credential lands |
|---|---|
| **OpenClaw** | `/root/config/config.json` `llm_*` + `/root/.openclaw/openclaw.json` `models.providers.autonomous` |
| **Hermes** | Hermes env / `~/.hermes/.env` (today Hermes is compile-time constants in `runtimes/hermes/constants.go` — that has to become per-unit config first) |

Subscription OAuth must **not** be treated as a console `xai-` / `sk-` key. Token refresh is a systemd timer or os-server loop, not “paste again at setup”.

### 1.1 Grok OAuth (started, not productized)

Code already in-tree, **not** on the setup page:

- Go library: `system/grokauth/` — OpenCode’s xAI plugin (`packages/opencode/src/plugin/xai.ts`): public Grok-CLI client id, RFC 8628 device code, refresh rotation. Bearer works at `https://api.x.ai/v1`.
- One-off Reachy helper: `grok-login/grok_login.py` + systemd timer. Writes `/root/config/grok-oauth.json`, then copies the access token into `config.json` / OpenClaw. **Not** wired into `/setup`, Settings → AI Brain, or Hermes.

Still needed:

- Setup + Settings UI: “Sign in with Grok” → show user code + `https://auth.x.ai/oauth2/device` (or verification_uri).
- os-server: start device-code, poll, persist, refresh before expiry, restart gateway when the access token rotates.
- Same token file consumed by OpenClaw **and** a Hermes xAI / SuperGrok provider.
- Model picker that lists Grok models after login, default something current (do not hardcode a stale id).
- Be explicit in the UI: this uses the **Grok / SuperGrok subscription**, not xAI console prepaid API.

### 1.2 Cloudflare Workers AI (and AI Gateway)

Missing entirely. OpenCode `/connect` has both:

| Connect as | Operator gives | Models |
|---|---|---|
| **Workers AI** | Cloudflare **Account ID** + API token (Workers AI REST API) | Workers AI catalog, including **Kimi** (`@cf/moonshotai/kimi-k2.5` and later). No separate Moonshot key if they use CF. |
| **AI Gateway** | Account ID + Gateway ID + token | Unified OpenAI/Anthropic/Workers AI endpoint; Unified Billing optional |

Wire as OpenAI-compatible `llm_base_url` for OpenClaw, and as a Hermes `cloudflare` / `workers-ai` provider. Keep account id + token out of `config.json` logs. Employee / paid CF plans must not be treated as the public free-tier limit (voice work already hit that confusion).

### 1.3 Everything else OpenCode `/connect` has that setup does not

`domain.ListProviders` is a small API-key list (OpenAI, Anthropic, Gemini, xAI, Groq, …). OpenCode’s directory is much larger. Priority for a robot brain:

| Provider | Auth OpenCode uses | Notes for us |
|---|---|---|
| **Kimi / Moonshot** | API key; also via Workers AI | Direct `kimi-coding` / `platform.kimi.ai` **and** CF-hosted Kimi. Hermes already documents `KIMI_API_KEY` — Autonomous setup does not. |
| **Anthropic Claude Pro/Max** | Browser OAuth | Device-code or a phone-app deep link; headless robot cannot open a browser. |
| **ChatGPT / OpenAI Codex** | OAuth (Codex CLI style) | We already have a `codex` runtime; setup still wants a pasted key. |
| **Google Gemini CLI / Antigravity** | OAuth | Listed in `ListProviders`, no login UI. |
| **GitHub Copilot** | OAuth | Listed, no login UI. |
| **OpenCode Zen / OpenCode Go** | API key after site login | Listed as `opencode`. |
| **OpenRouter, Groq, Cerebras, Mistral, Z.AI, Vertex, Vercel AI Gateway** | API key | Easy: add to the picker with the right default base URL + model. |
| **Amazon Bedrock, Azure, local Ollama** | keys / profile / LAN URL | Ollama BYO already works if you type the URL; give it a first-class “Local (Ollama / LM Studio)” choice. |

Implementation shape (one pattern, many providers):

1. Provider catalog with `auth: api_key | oauth | device_code | cloudflare_workers | byo_url`.
2. Setup step + Settings → AI Brain both use it.
3. Credential store on the device (mode 600), not in git, not in MQTT payloads in logs.
4. **Presync / switch_runtime** copies that store into OpenClaw `openclaw.json` **or** Hermes `.env` depending on `agent_runtime`.
5. Hermes must grow per-unit provider config before any of this works on a Hermes image.

---

## 2. Dash link for Autonomous Buddy (Mac)

The Overview **Autonomous Buddy (Mac)** card (`BuddyCard.tsx`) can start a pairing code and revoke. It says “Install Autonomous Buddy on your Mac” and **does not link to a build**.

On Reachy we had to: clone/build Swift, ad-hoc sign, copy to `/Applications`, then pair with a code that also required an **admin bearer** (`POST /api/buddy/pair/start`). That is the manual loop to kill.

Add on the card (unpaired state):

- **Download for Mac** → Setup Apps step + Buddy card (`GET /api/device/companion-apps`). Direct zip URL expects a GitHub Release asset `AutonomousBuddy.zip` on `K95M65/kestrel` (override with `KESTREL_GITHUB_REPO`).
- Short “how to pair”: install → Accessibility + Screen Recording if you want click/type → click Pair here → enter the code in the app.
- Optional: `reachy-xxxx.local` / LAN IP pre-filled copy button so the app does not have to discover via mDNS only.
- After pair: keep CONNECTED / OFFLINE; do not hide the download (reinstall / second Mac later).

Ship notes (or the link is a 404):

- Notarized or at least a documented “right-click → Open” ad-hoc build.
- Version in the dash vs `VERSION_AUTONOMOUS_BUDDY`.
- Login-item / `start-buddy.sh` so it comes back after reboot (LaunchAgent `exec` of an `LSUIElement` app does **not** open the WebSocket; that bit us).

---

## 3. Smaller setup gaps we hit on Reachy (parked)

Not the ask above, but they belong on the same list:

- Voice STT/TTS as first-class setup, not “reuse the LLM URL” (HAL now refuses that fallback).
- Exclusive wake phrase + follow-up window as a Settings control.
- Companion Mac as an optional speech/brain box (this house: `10.10.2.194:8787` talk/STT/TTS).
- Calendar / Gmail connectors during setup if Buddy is paired.
- Do not overwrite a custom `os-server` with the stock binary on OTA without a warning (we lost local builds that way).

---

## 4. Done vs not (this pass)

| Item | State |
|---|---|
| `system/grokauth` device-code + tests | In tree |
| Setup / Settings provider picker + Grok Sign in | **Kestrel:** `GET /api/device/llm-providers`, `POST /api/device/llm-oauth/start|poll`, AI Brain dropdown (Grok device-code, Kimi, CF Workers AI, Ollama, …) |
| Cloudflare Workers AI / Gateway / Kimi | Catalog + URL templates; paste token + account id |
| OpenCode-parity provider catalog + OAuth | Catalog landed; only **xAI** device-code is live. Copilot / Claude Pro / Gemini CLI still paste-token. |
| Hermes per-unit LLM config | **Not built** |
| Buddy download link on the dash | **Not built** |
