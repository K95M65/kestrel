import { defaultBehaviors, type BehaviorsConfig } from "@/lib/api";

export const DAYS = [
  { n: 0, label: "Sun" }, { n: 1, label: "Mon" }, { n: 2, label: "Tue" },
  { n: 3, label: "Wed" }, { n: 4, label: "Thu" }, { n: 5, label: "Fri" }, { n: 6, label: "Sat" },
];

export type PresetId = "desk" | "family" | "kids" | "office";

export const PRESETS: { id: PresetId; title: string; line: string }[] = [
  { id: "desk", title: "Just me", line: "A companion for one person — briefing, memory, dance, a hello when you sit down." },
  { id: "family", title: "Family", line: "Kitchen presence — greet whoever walks in, meals, stories if asked." },
  { id: "kids", title: "Kids around", line: "Gentle and bounded. No mail, calendar, or the house. Stories on." },
  { id: "office", title: "Office", line: "Named greetings, morning brief, focus nag, pomodoro. Dance stays off." },
];

/** What each starting personality actually turns on. Shown as a second pane after a pick. */
export const PRESET_COMPARE: { label: string; values: Record<PresetId, string> }[] = [
  { label: "Morning brief", values: { desk: "on", family: "on", kids: "off", office: "on" } },
  { label: "Greet everyone", values: { desk: "yes", family: "yes", kids: "yes", office: "named only" } },
  { label: "Mail / calendar", values: { desk: "on", family: "on", kids: "off", office: "on" } },
  { label: "Dance", values: { desk: "on", family: "on", kids: "on", office: "off" } },
  { label: "Stories", values: { desk: "off", family: "on", kids: "on", office: "off" } },
  { label: "Focus / pomodoro", values: { desk: "off", family: "off", kids: "off", office: "on" } },
];

export function mergeBehaviors(raw?: BehaviorsConfig | null): BehaviorsConfig {
  const d = defaultBehaviors();
  if (!raw) return d;
  return {
    ...d, ...raw,
    me: raw.me ?? d.me,
    morning_brief: { ...d.morning_brief, ...raw.morning_brief },
    remember: { ...d.remember, ...raw.remember },
    dance: { ...d.dance, ...raw.dance },
    privacy: { ...d.privacy, ...raw.privacy },
    connectors: { ...d.connectors, ...raw.connectors },
    presence: { ...d.presence, ...raw.presence },
    doa: { ...d.doa, ...raw.doa },
    layered_motion: { ...d.layered_motion, ...raw.layered_motion },
    focus: { ...d.focus, ...raw.focus },
    kids: { ...d.kids, ...raw.kids },
    greeter: { ...d.greeter, ...raw.greeter },
    look: { ...d.look, ...raw.look },
    kitchen: { ...d.kitchen, ...raw.kitchen },
    home_assistant: { ...d.home_assistant, ...raw.home_assistant, token: "" },
    marionette: { ...d.marionette, ...raw.marionette },
    tools: { ...d.tools, ...raw.tools },
    hand_track: { ...d.hand_track, ...raw.hand_track },
    radio: { ...d.radio, ...raw.radio },
    telepresence: { ...d.telepresence, ...raw.telepresence },
    stories: { ...d.stories, ...raw.stories },
    pomodoro: { ...d.pomodoro, ...raw.pomodoro },
    wearables: { ...d.wearables, ...raw.wearables },
  };
}

export function applyPreset(base: BehaviorsConfig, id: PresetId): BehaviorsConfig {
  const b = mergeBehaviors(base);
  const ha = b.home_assistant;
  const next = mergeBehaviors(defaultBehaviors());
  next.onboarded = b.onboarded;
  next.me = b.me;
  next.home_assistant = ha;
  next.connectors.draft_not_send = true;
  next.privacy.camera_on_demand = true;
  next.privacy.face_follow_after_wake = true;
  next.presence.idle_motion = true;
  next.remember.enabled = true;
  next.look.enabled = true;
  next.kitchen.enabled = true;
  next.tools = { weather: true, time: true, search: true };

  if (id === "desk") {
    next.morning_brief.enabled = true;
    next.morning_brief.at = "07:30";
    next.dance.enabled = true;
    next.greeter.enabled = true;
    next.greeter.named_only = false;
    next.kids.enabled = false;
    next.stories.enabled = false;
    next.focus.enabled = false;
    next.pomodoro.enabled = false;
  }
  if (id === "family") {
    next.morning_brief.enabled = true;
    next.morning_brief.at = "08:00";
    next.dance.enabled = true;
    next.greeter.enabled = true;
    next.greeter.named_only = false;
    next.stories.enabled = true;
    next.kids.enabled = false;
    next.focus.enabled = false;
    next.pomodoro.enabled = false;
  }
  if (id === "kids") {
    next.morning_brief.enabled = false;
    next.morning_brief.email = false;
    next.morning_brief.calendar = false;
    next.dance.enabled = true;
    next.greeter.enabled = true;
    next.kids.enabled = true;
    next.kids.session_min = 30;
    next.stories.enabled = true;
    next.focus.enabled = false;
    next.pomodoro.enabled = false;
    next.home_assistant.enabled = false;
  }
  if (id === "office") {
    next.morning_brief.enabled = true;
    next.morning_brief.at = "08:30";
    next.dance.enabled = false;
    next.greeter.enabled = true;
    next.greeter.named_only = true;
    next.kids.enabled = false;
    next.stories.enabled = false;
    next.focus.enabled = true;
    next.focus.phone_nag = true;
    next.pomodoro.enabled = true;
  }
  return next;
}

export type FeatureKey =
  | "morning_brief" | "remember" | "kitchen" | "pomodoro" | "wearables"
  | "dance" | "presence" | "doa" | "layered_motion" | "hand_track"
  | "marionette" | "radio" | "greeter" | "look"
  | "privacy" | "connectors" | "kids" | "stories" | "focus"
  | "home_assistant" | "tools" | "telepresence";

export const FEATURE_GROUPS: { id: string; title: string; keys: FeatureKey[] }[] = [
  { id: "daily", title: "Daily", keys: ["morning_brief", "remember", "kitchen", "pomodoro", "wearables"] },
  { id: "body", title: "Body", keys: ["dance", "presence", "doa", "layered_motion", "hand_track", "marionette", "radio", "greeter", "look"] },
  { id: "privacy", title: "Privacy & family", keys: ["privacy", "connectors", "kids", "stories", "focus"] },
  { id: "home", title: "Home & extras", keys: ["home_assistant", "tools", "telepresence"] },
];

export function isFeatureOn(cfg: BehaviorsConfig, key: FeatureKey): boolean {
  switch (key) {
    case "morning_brief": return cfg.morning_brief.enabled;
    case "remember": return cfg.remember.enabled;
    case "kitchen": return cfg.kitchen.enabled;
    case "pomodoro": return cfg.pomodoro.enabled;
    case "wearables": return cfg.wearables.enabled;
    case "dance": return cfg.dance.enabled;
    case "presence": return cfg.presence.idle_motion;
    case "doa": return cfg.doa.enabled;
    case "layered_motion": return cfg.layered_motion.enabled;
    case "hand_track": return cfg.hand_track.enabled;
    case "marionette": return cfg.marionette.enabled;
    case "radio": return cfg.radio.enabled;
    case "greeter": return cfg.greeter.enabled;
    case "look": return cfg.look.enabled;
    case "privacy": return cfg.privacy.camera_on_demand;
    case "connectors": return cfg.connectors.draft_not_send;
    case "kids": return cfg.kids.enabled;
    case "stories": return cfg.stories.enabled;
    case "focus": return cfg.focus.enabled;
    case "home_assistant": return cfg.home_assistant.enabled;
    case "tools": return cfg.tools.weather || cfg.tools.time || cfg.tools.search;
    case "telepresence": return cfg.telepresence.enabled;
  }
}

export function countOn(
  cfg: BehaviorsConfig,
  supported?: (key: FeatureKey) => boolean,
): { on: number; total: number } {
  const keys = FEATURE_GROUPS.flatMap((g) => g.keys).filter((k) => !supported || supported(k));
  const on = keys.filter((k) => isFeatureOn(cfg, k)).length;
  return { on, total: keys.length };
}

export function summaryLines(cfg: BehaviorsConfig): string[] {
  const out: string[] = [];
  if (cfg.morning_brief.enabled) out.push(`Morning brief at ${cfg.morning_brief.at}`);
  if (cfg.remember.enabled) out.push("Remembers what you tell it");
  if (cfg.dance.enabled) out.push("Dances to songs");
  if (cfg.greeter.enabled) out.push(cfg.greeter.named_only ? "Greets named friends" : "Greets people who walk in");
  if (cfg.kids.enabled) out.push("Kids profile on");
  if (cfg.connectors.draft_not_send) out.push("Drafts mail, never sends");
  if (cfg.privacy.camera_on_demand) out.push("Camera is one snapshot");
  if (cfg.focus.enabled) out.push("Phone / focus nag");
  if (cfg.pomodoro.enabled) out.push("Pomodoro timer");
  if (cfg.stories.enabled) out.push("Stories");
  if (cfg.kitchen.enabled) out.push("Kitchen / meals");
  if (cfg.home_assistant.enabled) out.push("Home Assistant");
  return out;
}

export function formatNext(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString(undefined, { weekday: "short", hour: "2-digit", minute: "2-digit" });
}

export function num(v: string, fallback: number): number {
  const n = parseInt(v, 10);
  return Number.isFinite(n) ? n : fallback;
}
