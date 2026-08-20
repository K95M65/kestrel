/** Capability-driven Guided Setup. Address ROBOT.md groups, never device ids. */

export const UNSUPPORTED = "Your current hardware does not support this feature";

export type GuideStepId =
  | "intro"
  | "name"
  | "talk"
  | "see"
  | "prove"
  | "preset"
  | "connect"
  | "mornings"
  | "extras"
  | "done";

export type GuideCaps = {
  audio: boolean;
  vision: boolean;
  motion: boolean;
  light: boolean;
  expression: boolean;
  presence: boolean;
  media: boolean;
  companion: boolean;
};

/** Fail-open while /api/system/info has not arrived — same as useCapabilities. */
export const OPEN_CAPS: GuideCaps = {
  audio: true,
  vision: true,
  motion: true,
  light: true,
  expression: true,
  presence: true,
  media: true,
  companion: true,
};

export function capsFromSet(set: Set<string> | null | undefined): GuideCaps {
  if (!set) return OPEN_CAPS;
  return {
    audio: set.has("audio"),
    vision: set.has("vision"),
    motion: set.has("motion"),
    light: set.has("light"),
    expression: set.has("expression"),
    presence: set.has("presence"),
    media: set.has("media"),
    companion: set.has("companion"),
  };
}

export function guideSteps(c: GuideCaps, connect = false): GuideStepId[] {
  const steps: GuideStepId[] = ["intro", "name", "talk"];
  if (c.vision) steps.push("see");
  if (c.motion || c.light || c.expression) steps.push("prove");
  steps.push("preset");
  if (connect) steps.push("connect");
  if (c.audio) steps.push("mornings");
  steps.push("extras", "done");
  return steps;
}

export type ExtraChip = {
  key: "greeter" | "dance" | "remember" | "look" | "kitchen" | "focus" | "pomodoro" | "stories" | "kids";
  label: string;
  need: (c: GuideCaps) => boolean;
};

export const EXTRA_CHIPS: ExtraChip[] = [
  { key: "greeter", label: "Greet people", need: (c) => c.presence || c.vision },
  { key: "dance", label: "Dance to songs", need: (c) => c.motion && c.media },
  { key: "remember", label: "Remember this", need: () => true },
  { key: "look", label: "Look at this", need: (c) => c.vision },
  { key: "kitchen", label: "Meals / kitchen", need: () => true },
  { key: "focus", label: "Phone nag", need: (c) => c.vision },
  { key: "pomodoro", label: "Pomodoro", need: (c) => c.audio },
  { key: "stories", label: "Stories", need: (c) => c.audio },
  { key: "kids", label: "Kids profile", need: () => true },
];

export function extrasFor(c: GuideCaps): ExtraChip[] {
  return EXTRA_CHIPS.filter((x) => x.need(c));
}

const PRESET_ROW_NEED: Record<string, (c: GuideCaps) => boolean> = {
  "Morning brief": (c) => c.audio,
  "Greet everyone": (c) => c.presence || c.vision,
  "Mail / calendar": () => true,
  "Dance": (c) => c.motion,
  "Stories": (c) => c.audio,
  "Focus / pomodoro": () => true,
};

export function presetRowsFor<T extends { label: string }>(c: GuideCaps, rows: T[]): T[] {
  return rows.filter((row) => (PRESET_ROW_NEED[row.label] ?? (() => true))(c));
}

/** House → Behaviors toggles: missing hardware is grey, not hidden. */
export function featureSupported(key: string, c: GuideCaps): boolean {
  switch (key) {
    case "morning_brief":
    case "stories":
    case "pomodoro":
      return c.audio;
    case "dance":
    case "layered_motion":
    case "marionette":
    case "radio":
      return c.motion;
    case "look":
    case "telepresence":
    case "privacy":
    case "focus":
      return c.vision;
    case "greeter":
      return c.presence || c.vision;
    case "doa":
      return c.audio && c.motion;
    case "hand_track":
      return c.vision && c.motion;
    case "presence":
      return c.motion || c.light;
    default:
      return true;
  }
}

export type ProveActId = "light" | "motion" | "emotion";

/** Sequential hello: ring (if any), look (if any), then a small expression. */
export function proveActs(c: GuideCaps): ProveActId[] {
  const out: ProveActId[] = [];
  if (c.light) out.push("light");
  if (c.motion) out.push("motion");
  if (c.expression) out.push("emotion");
  return out;
}

/** Fields a save/preset may legally leave on. Privacy stays on-demand even
 *  without a camera — turning it off would mean "stream may stay running". */
type Clampable = {
  dance?: { enabled: boolean };
  look?: { enabled: boolean };
  greeter?: { enabled: boolean };
  focus?: { enabled: boolean };
  stories?: { enabled: boolean };
  pomodoro?: { enabled: boolean };
  morning_brief?: { enabled: boolean };
  doa?: { enabled: boolean };
  layered_motion?: { enabled: boolean };
  marionette?: { enabled: boolean };
  radio?: { enabled: boolean };
  hand_track?: { enabled: boolean };
  telepresence?: { enabled: boolean };
  presence?: { idle_motion: boolean };
};

export function clampBehaviors<T extends Clampable>(cfg: T, c: GuideCaps): T {
  const next = { ...cfg };
  const off = (key: keyof Clampable, feat: string) => {
    if (featureSupported(feat, c)) return;
    const cur = next[key];
    if (cur && typeof cur === "object" && "enabled" in cur) {
      (next as Clampable)[key] = { ...cur, enabled: false } as never;
    }
  };
  off("dance", "dance");
  off("look", "look");
  off("greeter", "greeter");
  off("focus", "focus");
  off("stories", "stories");
  off("pomodoro", "pomodoro");
  off("morning_brief", "morning_brief");
  off("doa", "doa");
  off("layered_motion", "layered_motion");
  off("marionette", "marionette");
  off("radio", "radio");
  off("hand_track", "hand_track");
  off("telepresence", "telepresence");
  if (!featureSupported("presence", c) && next.presence) {
    next.presence = { ...next.presence, idle_motion: false };
  }
  return next;
}

export function ownsEnter(step: GuideStepId): boolean {
  return step === "talk" || step === "see" || step === "prove";
}

export function wideStep(step: GuideStepId): boolean {
  return step === "talk" || step === "see" || step === "prove" || step === "preset" || step === "connect";
}
