"""The Autonomous OS architecture figure — one drawing, house tokens.

    python3 docs/architecture/build_figures.py      # writes autonomous-stack.svg

then render it to PNG at 1.5× with headless Chrome (see DIAGRAMS notes in
build output) and look at the result before committing. READMEs embed the PNG.

The figure is a layered stack read top-down, and every layer names the folder
that implements it, so the drawing and the repository tree map 1:1. Green does
work, purple decides or holds state, coral is a terminal — the bodies the whole
stack exists to drive. Nothing is coloured for emphasis.
"""
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).parent))
from figs import (Fig, H, PAD, ARROW, LABEL_TEXT, PURPLE_TEXT, GREEN_TEXT, CORAL_TEXT,
                  NODE_FS, LABEL_FS, box_w, text_w)

HERE = pathlib.Path(__file__).parent

# ---- layout ---------------------------------------------------------------
LEFT = 96           # left caption column starts here (the act arrow runs at x=40)
COL = 620           # x where the first node begins (must clear the widest left caption)
RIGHT_PAD = 125     # gap between the last node and the right caption column
GAP = 18            # gap between sibling nodes in a row
ROW = 112           # centre-to-centre distance between rows
SUB = 84            # centre-to-centre distance between the two lines of one row


def row_width(labels):
    return sum(box_w(l) for l in labels) + GAP * (len(labels) - 1)


class Stack(Fig):
    """A Fig that knows how to lay out one row of the stack."""

    def caption_left(self, cy, name, folder=None, file=None, note=None):
        """Layer name, then the folder that implements it and the file that governs it."""
        self.parts.append(
            f'<text x="{LEFT}" y="{cy + 2:.0f}" fill="{LABEL_TEXT}" '
            f'font-size="{NODE_FS}" font-weight="600">{name}</text>')
        line = []
        if folder:
            line.append(f'<tspan fill="{LABEL_TEXT}">{folder}</tspan>')
        if file:
            line.append(f'<tspan fill="{PURPLE_TEXT}"> · {file}</tspan>')
        if line:
            plain = (folder or "") + (f" · {file}" if file else "")
            assert LEFT + len(plain) * (NODE_FS - 2) * 0.60 < COL - 24, \
                f"left caption {plain!r} runs under the node column — raise COL"
            self.parts.append(
                f'<text x="{LEFT}" y="{cy + 34:.0f}" font-size="{NODE_FS - 2}" '
                f'font-family="SF Mono, Menlo, Consolas, monospace">{"".join(line)}</text>')
        if note:
            ny = cy + 66 if line else cy + 34
            self.parts.append(
                f'<text x="{LEFT}" y="{ny:.0f}" fill="{ARROW}" font-size="{LABEL_FS}">{note}</text>')

    def caption_right(self, x, cy, *lines, colour=None):
        for i, ln in enumerate(lines):
            self.parts.append(
                f'<text x="{x:.0f}" y="{cy + 9 + i * 31:.0f}" fill="{colour or LABEL_TEXT}" '
                f'font-size="{LABEL_FS}" font-family="SF Mono, Menlo, Consolas, monospace">'
                f'{ln}</text>')
            self._saw(x + text_w(ln, LABEL_FS) * 1.15, cy + 9 + i * 31)

    def row(self, cy, labels, kind, x0=COL, term=False, dashed=()):
        x = x0
        for l in labels:
            w = box_w(l) if not term else max(H + 8, box_w(l) - 8)
            cx = x + w / 2
            if term:
                self.term(cx, cy, l, w=w, dashed=(l in dashed))
            else:
                self.box(cx, cy, l, kind=kind, w=w, dashed=(l in dashed))
            x += w + GAP
        return x - GAP  # right edge of the row

    def vlabel(self, x, y1, y2, text, up=False):
        """A vertical arrow along a margin with its caption rotated to run beside it."""
        if up:
            self.arrow(x, y2, x, y1)
        else:
            self.arrow(x, y1, x, y2)
        cy = (y1 + y2) / 2
        tx = x - 34 if not up else x + 34
        rot = 90 if not up else -90
        self.parts.append(
            f'<text x="{tx:.0f}" y="{cy:.0f}" fill="{LABEL_TEXT}" font-size="{NODE_FS}" '
            f'text-anchor="middle" transform="rotate({rot} {tx:.0f} {cy:.0f})">{text}</text>')


def stack():
    f = Stack(1600, 1300)
    y = 30

    widest = 0

    # 0 — Apps: the surfaces a person touches. Android's top layer is System Apps;
    # ours is the app that adds a robot, installs skills and switches brains.
    y += 60
    labels = ["Autonomous app", "web setup", "live monitor"]
    f.caption_left(y, "Apps", "system/web/", note="add a robot · install skills · switch brains")
    r = f.row(y, labels, None, term=True); widest = max(widest, r)

    # 1 — Skills
    y += ROW
    skills_y = y
    labels = ["guard", "emotion", "face-enroll", "servo-tracking", "music", "+ 20 more", "your skill"]
    f.caption_left(y, "Skills", "skills/", "SKILL.md")
    r = f.row(y, labels, "green", dashed=("your skill",)); widest = max(widest, r)

    # 2 — Agentic Runtime
    y += ROW
    labels = ["OpenClaw", "Hermes", "PicoClaw", "OpenCode", "Codex", "Claude Code", "your brain"]
    f.caption_left(y, "Agentic runtime", "runtimes/", "SOUL.md")
    r = f.row(y, labels, "purple", dashed=("your brain",)); widest = max(widest, r)

    # 3 — System managers
    y += ROW
    top = ["server", "intent", "network", "monitor", "healthwatch", "ambient"]
    bot = ["device", "skills", "plugin", "statusled", "vision", "agent", "buddy", "bootstrap"]
    f.caption_left(y + SUB / 2, "System services", "system/")
    r1 = f.row(y, top, "purple"); r2 = f.row(y + SUB, bot, "purple")
    widest = max(widest, r1, r2)
    y += SUB

    # 3b — Realtime voice: hosted in HAL, answers small talk or hands the turn to the brain
    y += ROW
    labels = ["Gemini Live", "OpenAI Realtime", "Qwen"]
    f.caption_left(y, "Realtime voice", "hal/realtime/",
                   note="a voice turn lands here first — answer, or hand it up")
    r = f.row(y, labels, "purple"); widest = max(widest, r)

    # 4 — HAL capabilities, two lines
    y += ROW
    top = ["audio", "vision", "sensing", "presence", "motion", "light", "display"]
    bot = ["expression", "lifelike", "media", "connectivity", "companion", "system"]
    f.caption_left(y + SUB / 2, "Capabilities", "devices/contract/", "DEVICE.md",
                   note="served by hal/routes/ on :5001")
    r1 = f.row(y, top, "green")
    r2 = f.row(y + SUB, bot, "green")
    widest = max(widest, r1, r2)
    y += SUB

    # 5 — Safety gate (band)
    y += ROW
    f.caption_left(y, "Safety gate", "hal/safety/", "SAFETY.md")
    band_y = y

    # 6 — Drivers + boards
    y += ROW
    labels = ["motors", "rgb", "camera", "voice", "display", "sensing", "tracking", "media_owner", "your driver"]
    f.caption_left(y, "Drivers", "hal/drivers/")
    r = f.row(y, labels, "green", dashed=("your driver",)); widest = max(widest, r)

    y += SUB
    labels = ["Raspberry Pi 4", "Raspberry Pi 5", "Raspberry Pi CM4", "OrangePi 4 Pro", "your board"]
    f.caption_left(y, "Boards", "hal/board/boards.json")
    r = f.row(y, labels, "green", dashed=("your board",)); widest = max(widest, r)

    # 7 — Linux (band)
    y += ROW
    f.caption_left(y, "Linux", note="vendor kernel, not ours")
    linux_y = y

    # 8 — Bodies (terminals)
    y += ROW
    labels = ["Lamp", "Intern", "Reachy Mini", "Go2-W", "your robot"]
    f.caption_left(y, "Bodies", "devices/")
    r = f.row(y, labels, None, term=True, dashed=("Go2-W", "your robot")); widest = max(widest, r)
    body_y = y

    # bands span the widest row
    f.band(COL, widest, band_y, "brightness · quiet hours · explicit-move speed · thermal",
           kind="purple")
    f.band(COL, widest, linux_y, "Raspberry Pi OS · OrangePi Debian · the robot's own image", kind="green")

    # margin arrows: act down the left, sense up the right
    top_y = skills_y - H / 2 - 4
    f.vlabel(54, top_y, body_y + H / 2, "act — a skill writes [HW:/servo/aim]")
    f.vlabel(widest + 34, top_y, body_y + H / 2, "sense — events → intent → agent", up=True)

    return f.write(HERE / "autonomous-stack.svg")


if __name__ == "__main__":
    p = stack()
    print(p)
    print("render: \"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome\" --headless "
          "--disable-gpu --hide-scrollbars --force-device-scale-factor=1.5 "
          f"--window-size=W,H --screenshot={p.with_suffix('.png')} file://{p}")
