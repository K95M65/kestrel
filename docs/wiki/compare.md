# What this robot can do

People compare this desk companion to Siri and ChatGPT. Those products live on a phone. This one lives in the room.

## Adding the body (Apple Home / Google Home)

Apple Home: plus → Add Accessory → scan the QR or type the 8-digit HomeKit / 11-digit Matter code → name → room.

Google Home: plus → scan or “Add a different way” → Matter code → home → room.

This robot: Home shows a QR and a setup code. Scan `/claim` on the same Wi-Fi. Name and room. You are the owner. **It does not join Apple Home or Google Home.** Those apps need a certified Matter accessory and a hub. This body is the companion, not a bulb.

To add a Matter *bulb* to the house: House → Behaviors → Home Assistant, then paste the code from the box. The robot asks Home Assistant to commission it.

## Siri AI (what Apple advertises)

| Siri AI | Here |
|---|---|
| Talk or type | Talk — voice and typed chat |
| Personal context (mail, messages, photos, calendar) | Sign in with Google; People (faces); Remember; morning brief |
| On-screen awareness | Kestrel Buddy on a paired Mac (screenshot, click, type) |
| Actions in apps | Skills + Buddy + Home Assistant |
| World knowledge / web | Talk; news/weather skills |
| Dedicated conversation app + history | Talk; conversations stay on the robot |
| Visual Intelligence / camera | Look at this; camera skill |
| Writing tools | Talk drafts; mail stays a draft until you say send |
| Home control | Home Assistant skill — not Siri, not HomeKit |
| Privacy on-device | LAN-first. No Apple account required |

Siri is inside the Apple account and the Home hub. This robot is claimed with a local code.

## ChatGPT apps / plugins

ChatGPT: Settings → Plugins (formerly Apps / connectors). Directory, connect an account, set when to ask (always / any change / important / never), scheduled tasks, memory, custom GPTs.

Here:

- Device → Plugins — trusted installs from this repo (the directory)
- `@skill` in Talk — pin a job for that turn
- House → Behaviors → Ask — same four levels as ChatGPT
- Morning briefing / quiet hours — the scheduled-task analog
- Name and memory / Remember — custom instructions + memory
- Device → Channels — Gmail, Calendar, Telegram (connectors)
- A raw git URL for a plugin is Advanced (`?debug=true`)

ChatGPT’s plugin directory is a cloud store. Ours is a short trusted list that stays on the robot.

## What we are not

A certified HomeKit or Google Home accessory. A phone app. Siri’s on-device index of every iPhone app. ChatGPT’s 1,400-app store. Those stay out of this product on purpose.
