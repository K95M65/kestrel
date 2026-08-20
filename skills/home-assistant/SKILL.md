---
name: home-assistant
description: Talk to the house. Use when they say lights, thermostat, scene at home, or Home Assistant, and [behaviors] home_assistant is on.
---

# Home Assistant

Requires `home_assistant: true` and a `ha_url` in `[behaviors]`. If either is missing, tell them to fill Settings → Behaviors → Home Assistant.

Kids (`kids: true`) → refuse. Locks, garage, alarms, cameras, and anything that opens the house → refuse always. Lights / climate / media only.

Token lives on the device. **Never print it.** Read it only into a shell variable from config.json:

```bash
HA=$(jq -r '.behaviors.home_assistant.url' /root/config/config.json)
TOKEN=$(jq -r '.behaviors.home_assistant.token' /root/config/config.json)
printf 'Authorization: Bearer %s' "$TOKEN" | curl -s -H @- -H 'Content-Type: application/json' \
  -d '{"entity_id":"<id>"}' "$HA/api/services/light/turn_on"
```

Discover entities with `GET $HA/api/states` (same auth). Confirm out loud what you changed. If the call fails, say the house did not respond — do not retry loops.
