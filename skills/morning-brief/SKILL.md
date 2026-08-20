---
name: morning-brief
description: Spoken morning briefing. Use when the message is tagged [companion:morning-brief], or they say "brief me", "what's today", "morning briefing".
---

# Morning brief

Isolated job. Speak one short briefing, then stop. Do not write MEMORY.md. Never send email or create calendar events.

Read `[behaviors: ...]` first. If `morning_brief` is false and this is not a manual `[companion:morning-brief]` fire, reply `NO_REPLY`.

## Include (only flags that are true)

| Flag | What to pull |
|---|---|
| `weather` | One-line local weather. Follow `weather/SKILL.md` (MCP if present, else Open-Meteo). Skip if that fetch fails. |
| `calendar` | Today's events via `connectors/SKILL.md` (read-only). |
| `email` | 3-line overnight inbox via connectors (read-only). Unread count + anything urgent. |
| `habits` | One beat from wellbeing/habit if logs exist. |
| `wearables` | Recovery/sleep from `wearable_provider` only if a connector exists. |

Kids profile (`kids: true`) → skip email, calendar, wearables. Weather + a kind hello only.

## Shape

`[HW:/emotion:{"emotion":"happy","intensity":0.6}]` then **20–40 seconds** spoken (honor `max_seconds` if present in the prompt). 1–3 short paragraphs. No bullets out loud. Match the owner's language.

If `draft_not_send` is true (default) you MUST NOT send or write anything.

Telegram copy: only if the prompt asked and `[user_info]` has `telegram_id` — `[HW:/dm:{"telegram_id":"<id>"}]` with a 5-line text version.

If connectors are missing, skip that slice and still brief the rest. Never invent events or mail.
