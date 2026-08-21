# Plugins

Three different things share the word “plugin.” The rooms keep them apart.

**Skills** are markdown the brain reads (`skills/<name>/SKILL.md`). They stay on every body that has the hardware. In Talk, type `@news` (or another name) to pin that job for the turn. Hive, weather, connectors, home-assistant are skills — not installable apps.

**Robot apps** are what Device → Plugins lists. Trusted from this repo: dance, emotions, cameraman, phrase teacher. Install by name. They run as their own process (systemd) and talk to HAL. Start / stop from that list. Same idea as ChatGPT’s plugin directory, but the list is short and on the robot.

**Kestrel Buddy** is a computer app (Mac, Windows, Linux). Pair from Home. It is not a robot-app.

House → Behaviors → **Ask** is when mail and calendar may act (always / any change / important / never) — the same four levels ChatGPT uses for apps.

A raw git URL is Advanced (`?debug=true`). Install = full trust for that code. Hugging Face browse is parked.

Builder notes: [`docs/plugin-system.md`](../../plugin-system.md), [`docs/apps-and-tools.md`](../../apps-and-tools.md).
