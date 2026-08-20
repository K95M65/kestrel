# Kestrel Host

Linux (or later a Mac Mini): webcam + speaker, **no motors**. Develop here.
Promote to the Wireless Reachy when a slice is done.

This Mac already has [Lima](https://lima-vm.io) (`limactl`). That is the VM.

## Once — Linux VM

```bash
limactl start --name kestrel template://ubuntu
limactl shell kestrel
```

On the guest, first boot only: nginx, Python 3.12, the repo (or rsync), then
HAL in simulate mode and os-server. `DEVICE_TYPE=kestrel-host`,
`HAL_BOARD=sim`, `HAL_SIMULATE=1`. Pass a USB camera into Lima later if you
want a real image; until then the camera is virtual.

After that, treat it like any other unit:

```bash
# from this repo, on the Mac
make os-build web-build
make sideload TARGET=lima-kestrel
```

`TARGET` is any `user@host` you can ssh to — the VM, or `pollen@10.10.2.160`.
It is not hardcoded to the Pi.

## Virtual devices

On that Linux machine you add what you have. No motors in this `ROBOT.md`.
A pan/tilt or a Reachy is a **different** `robots/<id>/`, not a fork of the OS.

## Mac Mini (later)

Same body shape: display, webcam, mic, speaker, no servos. Darwin is a new
board entry (`HAL_SIM_MEDIA=host` or native capture). Same rooms, same
sideload of web + os-server once those binaries are macOS-native. Do not
clone the CM4 image onto it.
