/** Per-body Guided Setup copy. Flavor by ROBOT.md id; steps still come from caps. */

import type { GuideCaps } from "@/lib/guideWalk";

export type BodyCopy = {
  introTitle: string;
  introLead: string;
  kind: string;
  sleep: string;
  expression: string;
  proveLead: string;
  proveMotion: string;
  proveLight: string;
  proveEmotion: string;
  welcomeLine: string;
  talkLead: (name: string) => string;
  seeLead: (name: string) => string;
  faceLead: string;
};

const GENERIC: BodyCopy = {
  introTitle: "Let's try this robot",
  introLead: "Name it, talk to it, then pick how it lives here. Skip any try if it isn't answering.",
  kind: "robot",
  sleep: "Sleep is a do-not-disturb: it stays quiet until you wake it.",
  expression: "It shows how it feels with whatever this body has.",
  proveLead: "A quick hello from the body, so you know it can move or light up.",
  proveMotion: "It will look toward you.",
  proveLight: "The light will glow.",
  proveEmotion: "A small hello.",
  welcomeLine: "Name it, talk to it, then pick how it lives here.",
  talkLead: (name) =>
    name === "the robot"
      ? "The robot will say hello. Reply here if you don't hear it."
      : `Say hi to ${name}. If the speaker is muted, type here instead.`,
  seeLead: (name) =>
    `We'll add you as the first person. Stand in front of ${name}, take a photo, then say who you are.`,
  faceLead: "A photo so it can say hello by name. Take one here, or pick a file.",
};

const BY_TYPE: Record<string, Partial<BodyCopy>> = {
  lamp: {
    introTitle: "Let's try this lamp",
    introLead: "Name it, talk to it, let it see you. The ring is its face; the arm is how it nods.",
    kind: "desk lamp",
    sleep: "When it sleeps the arm goes limp and the ring dims.",
    expression: "It talks with its ring and a slow nod.",
    proveLead: "The arm looks toward you and the ring glows — a hello without words.",
    proveMotion: "The arm will look toward you.",
    proveLight: "The ring will glow warmly.",
    proveEmotion: "A small hello on the ring.",
    welcomeLine: "Name it, talk to it, try the camera and the ring. Then pick how it lives here.",
    talkLead: (name) => `Say hi to ${name}. The ring will glow when it answers — type here if the speaker is muted.`,
  },
  "reachy-mini": {
    introTitle: "Let's try this robot",
    introLead: "Name it, talk to it, let it see you. It talks with its head and ears, not a light.",
    kind: "desk robot",
    sleep: "When it sleeps the ears fold down and out of the way.",
    expression: "It talks with a head tilt and its ears.",
    proveLead: "The head will look toward you — a hello without words.",
    proveMotion: "The head will look toward you.",
    proveEmotion: "A small nod hello.",
    welcomeLine: "Name it, talk to it, try the camera. Then pick how it lives here.",
    talkLead: (name) => `Say hi to ${name}. Watch the head and ears — type here if the speaker is muted.`,
  },
  "intern-v2": {
    introTitle: "Let's try this intern",
    introLead: "Name it and talk to it. The ring is how it shows up — no camera, no arm.",
    kind: "desk intern",
    sleep: "When it sleeps the ring goes dark.",
    expression: "It talks with its ring.",
    proveLead: "The ring will glow — a hello without words.",
    proveLight: "The ring will glow warmly.",
    proveEmotion: "A small hello on the ring.",
    welcomeLine: "Name it and talk to it. Then pick how it lives here.",
    talkLead: (name) => `Say hi to ${name}. The ring will glow when it answers — type here if the speaker is muted.`,
  },
  "kestrel-host": {
    introTitle: "Let's try this computer",
    introLead: "Name it, talk to it, let it see you. Camera, mic, and speaker — no motors.",
    kind: "desk computer",
    sleep: "Sleep is a do-not-disturb: it stays quiet until you wake it.",
    expression: "It talks with its voice. There is no ring or head to watch.",
    proveLead: "A short hello from the speaker, so you know it can answer.",
    proveEmotion: "A short hello out loud.",
    welcomeLine: "Name it, talk to it, try the camera. Then pick how it lives here.",
    talkLead: (name) => `Say hi to ${name}. Type here if the speaker is muted.`,
  },
};

/** If the OS omits deviceType, infer a flavor from hardware — never a hardcoded id in the walk.
 *  Pass loaded=false until /api/system/info arrives so we don't flash Lamp copy on Reachy. */
export function inferredBodyId(deviceType: string, caps: GuideCaps, loaded = true): string {
  const id = (deviceType || "").trim().toLowerCase();
  if (id && BY_TYPE[id]) return id;
  if (!loaded) return "";
  if (caps.motion && caps.light) return "lamp";
  if (caps.motion && !caps.light) return "reachy-mini";
  if (caps.light && !caps.motion && !caps.vision) return "intern-v2";
  if (caps.audio && caps.vision && !caps.motion && !caps.light) return "kestrel-host";
  return id;
}

export function bodyCopy(deviceType: string, caps: GuideCaps, loaded = true): BodyCopy {
  const id = inferredBodyId(deviceType, caps, loaded);
  const spec = BY_TYPE[id] || {};
  const copy: BodyCopy = { ...GENERIC, ...spec };
  copy.faceLead = `A photo so this ${copy.kind} can say hello by name. Take one here, or pick a file.`;
  if (!caps.vision) {
    copy.introLead = copy.introLead.replace(/,?\s*let it see you\.?/i, ".").replace(/\.\./, ".");
  }
  if (!caps.light) {
    copy.proveLight = "";
  }
  if (!caps.motion) {
    copy.proveMotion = "";
  }
  return copy;
}
