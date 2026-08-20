# Reachy Mini Wireless — compute and local AI

Measured on the desk unit (`10.10.2.160`, 2026-08-19) and matched against
Pollen / Seeed / Hugging Face datasheets.

## What is inside

| Item | Spec |
|---|---|
| Compute | Raspberry Pi **CM4 CM4104016** (Rev 1.1) |
| CPU | Broadcom BCM2711, 4× Cortex-A72 @ 1.5 GHz |
| RAM | **4 GB** LPDDR4 (~3.7 GiB visible) |
| Storage | **16 GB eMMC** (~14 GB filesystem; this unit ~12 GB used, ~1.7 GB free) |
| Wireless | On-module Wi-Fi + BT; dual-band patch antenna 2.79 dBi |
| Camera | Pi Camera Module 3 Wide, Sony IMX708, 12 MP, 120°, AF, CSI |
| Audio | 4× PDM MEMS mics (XMOS XVF3800 reSpeaker) + 5 W / 4 Ω speaker |
| Power | LiFePO4 2000 mAh, 6.4 V, 12.8 Wh; input 6.8–7.6 V |
| USB-C | **Data only.** Does not charge the robot. Thumb drive / dongle OK if bus-powered. |
| Body | 30 × 20 × 15.5 cm, 1.475 kg; 6-DOF Stewart head + 360° base + 2 antennas |

There is **no 40-pin GPIO header** and no exposed PCIe. The CM4 sits on Pollen’s
carrier inside the sealed body. A Raspberry Pi 5 HAT (AI HAT+, AI HAT+ 2, M.2
HAT) has nothing to plug into.

## Can we add local AI with a HAT?

**No worthwhile HAT path.** Official AI Kit / AI HAT+ / AI HAT+ 2 (Hailo-8L,
Hailo-8, Hailo-10H + 8 GB) are Pi 5 40-pin + PCIe boards. Stuffing one inside
Reachy is a chassis and power rewrite, not a weekend mod.

## Options that actually exist

### 1. Keep the brain on the Mac (what we already run)
The desk Mac already serves local models at `:11434`. Talk, briefs, and vision
descriptions should stay there. The CM4’s job is HAL, camera, mics, motors, and
the OS. This is the correct architecture for a 4 GB / 16 GB body.

### 2. USB accelerator for *vision*, not an LLM
USB-C is a real host port.

| Dongle | About | Useful for | Not useful for |
|---|---|---|---|
| Google Coral USB (Edge TPU, ~4 TOPS) | ~$60, USB 3, bus-powered | Face / person / object on-device | Any LLM |
| Hailo USB | No official USB Hailo. Hailo is M.2 / HAT. | — | — |

A Coral can take load off the Pi for “who is this” and greeter presence. It
will not run Grok, Llama, or Whisper-large. Power budget: Coral wants USB 3
current; the port is specified for thumb drives — measure before leaving it
plugged in on battery.

### 3. Swap the Compute Module (worth considering later)
The carrier is a standard CM4 socket.

| Module | RAM / flash | Why |
|---|---|---|
| CM4104016 (stock) | 4 GB / 16 GB | What shipped |
| CM4108032 | 8 GB / 32 GB | Same Wi-Fi CM4, more RAM + disk. Best *drop-in* if the board accepts it. Relieves the 88% disk, still not an LLM box. |
| CM5 (various) | up to 16 GB | 2–3× CPU. **Not a guaranteed drop-in** on this carrier (UART / firmware differences reported on Reachy Mini Wireless). Do not swap without a recovery image. |

An 8 GB / 32 GB CM4 is the only hardware upgrade I would call worthwhile on
this body: headroom for HAL + camera + a small vision model. It still will not
host the companion LLM.

### 4. Pi AI Camera (IMX500)
On-sensor NPU, CSI. Theoretically a camera-module swap. Different FOV / mount
than the current Camera v3 Wide in the face. High risk of a mechanical miss;
not a first experiment.

## Recommendation

Do not chase a HAT. Do not try to run the companion LLM on the 4 GB CM4 next to
HAL, OpenClaw, and Pollen.

1. **Now:** Mac (or other always-on box) as the brain. Already wired.
2. **If vision should stay local when the Mac is off:** Coral USB, after a
   current-draw check on this USB-C port.
3. **If disk/RAM becomes the pain:** CM4 8 GB / 32 GB swap, with a cloned eMMC
   image first.

Local LLM on the robot itself would mean a different body (Pi 5 class + HAT, or
an N100/Orin sidecar), not a hat clipped onto this one.
