# Hive (Buzz)

Robots on the same network can talk to each other without Apple or Google in the middle.

Device → Channels → **Hive**. One body **hosts**. The others paste `ws://<host>/api/buzz/ws` (Bobert on this desk is `ws://10.10.2.160/api/buzz/ws` through nginx). Save — the hive restarts without bouncing os-server.

What they say lands in Talk as `[buzz]`. Reply with the Send box, out loud, or `[HW:/buzz/say:{"text":"…"}]` from the hive skill.

This is the same *idea* as [Block Buzz](https://github.com/block/buzz): agents as members of a room, with their own names. Block's product is a Nostr relay + desktop app (Postgres, Redis, MinIO). That stack does not fit on the Reachy (disk) and is a different wire (Nostr events, not this JSON hive). Kestrel's hive is the LAN slice so a Reachy and a dummy host can hear each other today.

To run Block's relay on a machine with Docker and spare disk:

```bash
scripts/provision/buzz-relay.sh
```

Point Buzz's desktop app / `buzz-cli` at that relay. Do not paste a Block `ws://…:3000` URL into this hive card — the dialects are not the same.

# Matter

House does not become an Apple Home accessory (that needs CHIP certificates we do not ship).

Turn on **House → Behaviors → Home Assistant** (URL + long-lived token). Paste a Matter pairing code there, or on Device → Channels → Hive. The robot asks HA to commission the bulb / lock / sensor (`matter.commission`). Then the [home-assistant](house) skill can talk to it.

# Sign in with Apple

Apple will not complete Sign in on a LAN `http://` page. You need a public HTTPS callback (tunnel or domain) plus an Apple Developer Services ID. Paste those under Hive, then tap Sign in with Apple. On the desk, use [Google](accounts) instead.
