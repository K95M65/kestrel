# Hive (Buzz)

Two robots on this Wi-Fi can hear each other in Talk. Device → Channels → **Talk to another robot**.

1. Turn it on. Check **This robot hosts**.
2. Copy or scan the join address (QR on that card).
3. On the other body, paste that address and Save.
4. Send a test line. It shows up in Talk as `[buzz]`.

This is the pairing-code idea from Apple Home / Google Home, for robots talking to robots — not for adding a bulb.

A full [Block Buzz](https://github.com/block/buzz) relay (Nostr + Postgres) is a different wire. Optional later: `scripts/provision/buzz-relay.sh` on a machine with Docker. Do not paste a Block `ws://…:3000` URL here.

# Matter

House does not become an Apple Home accessory (that needs CHIP certificates we do not ship).

Turn on **House → Behaviors → Home Assistant** (URL + long-lived token). Paste the pairing code from the box under **Add a Matter accessory**. The robot asks HA to commission the bulb / lock / sensor. Then the [home-assistant](house) skill can talk to it.

# Sign in with Apple

Apple will not complete Sign in on a LAN `http://` page. On the desk, use [Google](accounts). Developer fields are Advanced (`?debug=true`).
