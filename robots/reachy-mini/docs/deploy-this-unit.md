# Deploy Kestrel on this Reachy Mini

Operational plan for the desk unit at `10.10.2.160` (`reachy-mini`, user `pollen`).
This is a **side-load onto the already-provisioned robot**, not a flash and not a
fleet OTA promote.

**Do not flash a golden image.** Pollen's daemon owns motors. Autonomous OS sits
on top.

## Current vs target (measured 2026-08-19)

| Component | On robot | Local tree | Action |
|-----------|----------|------------|--------|
| os-server | 0.1.19 | 0.1.20 | Replace binary, restart unit |
| web | 0.1.17 | 0.1.18 | Replace nginx `setup/` bundle |
| HAL | 0.1.12 | 0.1.12 | Leave running |
| bootstrap | 0.0.18 | — | Leave |
| OpenClaw | 2026.6.10 | — | Leave |
| Pollen daemon | active | — | Leave |
| Device profile | 0.0.10 | — | Leave |

os-server 0.1.20 resets the agent session on rename, surfaces muted/busy Test Voice,
and publishes name-family wake phrases. Web 0.1.18 is the 8-step guided setup
(live camera enroll, preset pane, locked privacy, “You’re set”).

Disk is tight (~1.7 GB free). Artifacts are ~25 MB + a few MB of web. Fine.

## Why not OTA

`software-update` pulls the public feed. These versions are not published there.
Promoting GCS would move the whole fleet. This unit gets a local arm64 build.

## Steps

1. **Backup** `/usr/local/bin/os-server` → `/root/bootstrap/rollback/os-server.previous`
2. **Build** on the Mac: `make os-build VERSION=0.1.19` and `make web-build`
3. **Copy** binary + `system/web/dist/` to the robot
4. **Install os-server**, `systemctl restart os-server`
5. **Install web** into `/usr/share/nginx/html/setup/`, `nginx -s reload`
6. **Smoke**
   - `os-server --version` → 0.1.19
   - `GET /api/device/behaviors` → 200 (admin cookie)
   - `PUT /api/device/identity` exists (not 404)
   - Home / Talk / Camera still work
   - Pollen daemon still active
7. **Rollback** if needed: `sudo software-update rollback os-server` or copy the
   backup back and restart

HAL, OpenClaw, and the Pollen daemon are not restarted.

## Applied 2026-08-19

Side-loaded on `10.10.2.160`:

- os-server **0.1.20** at `/usr/local/bin/os-server` (0.1.19 at `/root/bootstrap/rollback/os-server.0.1.19`)
- web **0.1.18** at `/usr/share/nginx/html/setup/`
- Smoke: login 200, `GET /api/device/behaviors` 200, `PUT /api/device/identity` validates names, HAL `/health` 200, OpenClaw connected, Pollen daemon still active

Product follow-ups from the desk walk: [`docs/product-work-queue.md`](../../../docs/product-work-queue.md).
Overlay vs stock Autonomous OS: [`docs/divergence-from-stock.md`](../../../docs/divergence-from-stock.md).

## Out of scope this pass

- Publishing to GCS / promoting `min_version`
- HAL Python (no delta vs 0.1.12)
- Re-running `spike.sh` (would re-pull OTA and could fight this side-load)
- Changing Grok keys, Wi-Fi, or the admin password
