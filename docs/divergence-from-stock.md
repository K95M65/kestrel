# Divergence from stock Autonomous OS

This tree is **Kestrel** (`K95M65/kestrel`) plus local product work for the
desk Reachy Mini. Stock is **Autonomous OS**
([`autonomous-ai/autonomous-os`](https://github.com/autonomous-ai/autonomous-os)).

When stock ships a bugfix, we do **not** `software-update` the stock `os-server`
or web bundle over this unit. We fetch `upstream`, see if the fix lands in a
file we overlaid, and port it by hand.

```bash
git fetch upstream
./scripts/check-upstream-divergence.sh
```

Watch list: [`scripts/upstream-watch-paths.txt`](../scripts/upstream-watch-paths.txt).
Add a path there whenever we overlay another stock file.

Last `git fetch upstream` in this log: **2026-08-19**, stock `ce3c8618`
(`Merge remote-tracking branch 'origin/main'`). Merge-base with our HEAD was
`667cef92` — kestrel diverged early, so the script lists a long stock history.
Treat **recent** stock SHAs as the ones to port; older ones may already be in
kestrel under a different hash.

Worth a look on current stock (not ported here yet):

| Stock | What |
|---|---|
| `d6852b08` | Copy Device ID on plain-HTTP (we have a copy button — check if their fix still applies) |
| `329831bc` | Sync `llm_api_key` rotation into `openclaw.json` |
| `3686c4db` | Standalone `/wifi` portal in AP mode |
| `4c38f35b` | UI button to enable + restart the gateway |
| `eb74eeb3` | Settings shows accepted wake phrases (we replaced this with name-family chips) |

Remotes in this clone:

| Remote | Repo |
|---|---|
| `origin` | `https://github.com/K95M65/kestrel.git` |
| `upstream` | `https://github.com/autonomous-ai/autonomous-os.git` |

---

## Why this exists

Stock is a device-agnostic robot OS (Lamp / Intern / Reachy Mini as *bodies*).
This product is **Kestrel / Desk Companion** on one Reachy Mini Wireless
(`10.10.2.160`). Chrome, onboarding, identity, and SOUL are ours. HAL, Pollen
daemon, OpenClaw, and most of os-server stay stock unless listed below.

OTA from the public feed will **overwrite** a side-loaded `os-server` /
`system/web/dist`. This unit was last side-loaded from this tree (see
[`robots/reachy-mini/docs/deploy-this-unit.md`](../robots/reachy-mini/docs/deploy-this-unit.md)).
Do not promote GCS `min_version` for these versions unless we intend the whole
fleet to get the overlay.

---

## Overlay catalog

Grouped by why we changed it, not by commit. Update this table when we add a
divergence. The check script only cares about paths; this table is for humans.

### Brand / chrome (stock is “Autonomous”)

| What we did | Stock does | Files |
|---|---|---|
| Tab icon is the Kestrel mark (ReachyMark twin) | Autonomous λ `favicon.ico` | `system/web/public/favicon.svg`, `favicon-32.png`, `apple-touch-icon.png`, `system/web/index.html` |
| Document title `Kestrel · …` | `Autonomous Setup` / Autonomous | `index.html`, `useDocumentTitle.ts` |
| Sidebar / login lockup **Kestrel / Desk Companion** | Autonomous / device type | `ReachyMark.tsx`, `Login.tsx`, `monitor/index.tsx` |
| Public rooms: Talk, Home, House (People first), Device | Monitor + Settings dump | `monitor/types.ts` `NAV`, `PUBLIC_SECTIONS` |
| Ocean theme (amber → `#3368A0`) | Stock lamp amber | `index.css` |

### Identity, wake, voice

| What we did | Stock does | Files |
|---|---|---|
| `PUT /api/device/identity` writes name, `NewSession` + name ping with in-flight gate so Talk cannot overlap | Name lives in IDENTITY.md only; session keeps the old “I am …” | `system/device/identity.go`, runtime `*identity.go` |
| Public wake chips = `hey {name}` family only | Merges `hey autonomous` + device-type aliases | `config_update.go`, `i18n/chitchat.go` |
| Test Voice toasts mute / busy (HAL 409) | Fire-and-forget, silent if muted | `system/server/voice.go`, `hal/client.go`, `TTSSection.tsx` |
| SOUL defers to IDENTITY.md `**Name:**` | Hardcoded “You are **Reachy**” | `robots/reachy-mini/SOUL.md` |

### Guided setup / companion pack

| What we did | Stock does | Files |
|---|---|---|
| 8-step guide: name → spoken hello → live MJPEG enroll → preset table → mornings → extras → You’re set | No product guide, or older “Let’s try this desk” 9-step | `BehaviorsOnboarding.tsx`, `guide/GuideTalkStep.tsx`, `guide/GuideSeeStep.tsx`, `guide/LiveFaceEnroll.tsx` |
| House → People is a contact book (live add-a-friend, photo/voice on the person) | Face + My Voice as Device tabs plus a household dump | `FaceOwnersSection.tsx`, `face-owners/EnrollModal.tsx`, `ContactVoiceBar.tsx`, `lib/voiceEnroll.ts` |
| Rename waits out NewSession; Talk chat is refused while reset is in flight | Session keeps the old name; next chat dropped busy | `system/device/identity.go`, sensing `handler.go` |
| `?guide=1` consumed once | Sticky query reopens the modal on every House click | `BehaviorsSection.tsx` |
| Behaviors API + presets (Just me / Family / Kids / Office), locked-down privacy default | Not in stock os-server | `system/device/behaviors.go`, `behaviorsModel.ts`, `server/config/behaviors.go` |
| Skills: morning-brief, greeter, remember, stories, kitchen, pomodoro, focus-coach, body-play, home-assistant | Absent or lamp-only | `skills/*` |

### This body (Reachy Mini Wireless)

| What we did | Stock does | Files |
|---|---|---|
| Hardware notes (CM4 CM4104016, no Pi 5 HAT) | Generic reachy-mini ROBOT.md | `robots/reachy-mini/docs/hardware.md`, `ROBOT.md` |
| Side-load plan for `10.10.2.160` | OTA from public feed | `robots/reachy-mini/docs/deploy-this-unit.md` |
| Product shots in README | Pollen artwork only | `robots/reachy-mini/images/app/` |

### House Kestrel commits already on `origin` (pre-overlay)

These are in kestrel `main` and may or may not be upstream. Re-check with the
script after `git fetch`. Notable:

- Quiet hours on the dashboard
- Grok device login / LLM provider catalog
- Exclusive wake + Mac talk hook
- Plugin install / Grok token refresh
- Reachy dance / emotions / cameraman apps
- Companion download links

---

## How to take a stock fix

1. `git fetch upstream`
2. `./scripts/check-upstream-divergence.sh`
3. If a stock SHA touches a watch path:
   - Read the stock patch (`git show <sha> -- path`)
   - Port the *fix*, keep our product behavior
   - Add a line under the matching section above if the overlay grew
4. If the stock SHA is outside the watch list, it is safe to merge or ignore
   independently (usual git).
5. Never `software-update os-server` / `web` from the public feed on this unit
   unless the overlay is in that build.

---

## This unit

| | |
|---|---|
| Host | `10.10.2.160` (`reachy-mini`, user `pollen`) |
| Overlay | os-server **0.1.20**, web **0.1.20** (People contact book + favicon) |
| Rollback | `/root/bootstrap/rollback/os-server.0.1.19` |
| Do not touch | Pollen daemon, HAL 0.1.12 unless a listed overlay needs it |

Working queue: [`docs/product-work-queue.md`](product-work-queue.md).
