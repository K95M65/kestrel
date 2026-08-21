# Dummy host

A Linux VM (or any box with a camera, mic, and speaker) can run the same OS with no motors.

Kestrel Host is that body: audio and vision, no servo. HAL boots simulated until you pass through real media.

On this Mac, Lima VM `kestrel` serves the UI at `http://127.0.0.1:8080` (guest `192.168.5.15`). It runs OpenClaw like the desk unit. Sign-in password for that VM is local only.

To talk to the desk Reachy (Bobert): Device → Channels → **Talk to another robot**. Bobert hosts. Lima pastes `ws://10.10.2.160/api/buzz/ws`. Bobert cannot open Lima; Lima can open Bobert.

The desk robot is a different machine. Deploying here does not flash Reachy. See [Hive](hive).
