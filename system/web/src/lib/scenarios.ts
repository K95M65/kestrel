/** OS-owned use catalog. Brains consume the skills; they do not own this list. */

import { UNSUPPORTED, type GuideCaps } from "./guideWalk.ts";
import type { RecipeService } from "./lifeRecipes.ts";

export type ScenarioId = "chat" | "web" | "news" | "music" | "spotify" | "stories" | "look" | "dance";

export type ScenarioStepId = "intro" | "connect" | "buddy" | "tools" | "try" | "done";

export type ScenarioMcp = {
  name: string;
  title: string;
  url: string;
};

export type Scenario = {
  id: ScenarioId;
  title: string;
  line: string;
  why: string;
  category: "chat" | "web" | "news" | "media" | "see";
  /** SKILL.md names both runtimes already read from the workspace. */
  skills: string[];
  services: RecipeService[];
  mcp: ScenarioMcp[];
  tools: { weather?: boolean; time?: boolean; search?: boolean };
  briefWeather?: boolean;
  buddy: "off" | "optional" | "required";
  kids: boolean;
  need: (c: GuideCaps) => boolean;
  tryPrompt: string;
  tryHint: string;
  honest: string;
  /** Extra words sidebar search matches — id/title/line are already included. */
  tags: string[];
  /** Behaviors this use turns on when you finish setup. */
  flip?: { stories?: boolean; look?: boolean; dance?: boolean };
};

const TELEGRAM: RecipeService = {
  id: "telegram",
  title: "Telegram",
  why: "Text it from your phone. Bot token from @BotFather, then your user id from @userinfobot.",
  kind: "channel",
  auth: "token",
};

/** Public Pollen Gradio MCP Spaces. Optional — weather/news skills also work without them. */
export const SCENARIO_MCP = {
  weather: {
    name: "weather",
    title: "Weather",
    url: "https://pollen-robotics-reachy-mini-weather-tool.hf.space/gradio_api/mcp/sse",
  },
  search: {
    name: "search",
    title: "Web search",
    url: "https://pollen-robotics-reachy-mini-search-tool.hf.space/gradio_api/mcp/sse",
  },
  time: {
    name: "time",
    title: "Time",
    url: "https://pollen-robotics-reachy-mini-time-tool.hf.space/gradio_api/mcp/sse",
  },
} as const satisfies Record<string, ScenarioMcp>;

export const SCENARIOS: Scenario[] = [
  {
    id: "chat",
    title: "Chat from your phone",
    line: "Text it on Telegram when you are not at the desk.",
    why: "Same conversation as Talk, on your phone. Slack and Discord stay under Device → Channels if you need those later.",
    category: "chat",
    skills: [],
    services: [TELEGRAM],
    mcp: [],
    tools: {},
    buddy: "off",
    kids: true,
    need: () => true,
    tryPrompt: "When I message you on Telegram, what should I say?",
    tryHint: "After the bot is linked, send it a line from your phone. You can also try one here.",
    honest: "This is Telegram. It does not log you into iMessage or WhatsApp.",
    tags: ["telegram", "phone", "text", "imessage", "whatsapp", "discord", "slack"],
  },
  {
    id: "web",
    title: "Websites on your computer",
    line: "Open a site or app on the computer next to it.",
    why: "Needs Kestrel Buddy on that computer (Mac, Windows, or Linux). Then: “open Gmail”, “search for …”, “join this Meet”.",
    category: "web",
    skills: ["computer-use"],
    services: [],
    mcp: [],
    tools: {},
    buddy: "required",
    kids: false,
    need: (c) => c.companion,
    tryPrompt: "Open Gmail on my computer",
    tryHint: "That opens the site on the paired computer, not in this browser.",
    honest: "It drives the computer. It does not browse the web by itself.",
    tags: ["gmail", "browser", "website", "chrome", "computer", "buddy", "mac", "windows", "linux"],
  },
  {
    id: "news",
    title: "News and weather",
    line: "Ask “what’s the weather?” or “what’s in the news?”",
    why: "Weather from a public forecast. Headlines from public news feeds. The morning brief can speak weather too.",
    category: "news",
    skills: ["weather", "news", "morning-brief"],
    services: [],
    mcp: [SCENARIO_MCP.weather, SCENARIO_MCP.search, SCENARIO_MCP.time],
    tools: { weather: true, time: true, search: true },
    briefWeather: true,
    buddy: "off",
    kids: true,
    need: (c) => c.audio,
    tryPrompt: "What's the weather in Paris?",
    tryHint: "A named city gets a real forecast. Headlines: “what's in the news?”",
    honest: "Headlines are public feeds, not a paywalled paper. It will not invent a forecast.",
    tags: ["weather", "forecast", "headlines", "news", "rain"],
  },
  {
    id: "music",
    title: "Music on the speaker",
    line: "Play a song out loud.",
    why: "YouTube through this body's speaker. Say a track, or “play something chill.”",
    category: "media",
    skills: ["music"],
    services: [],
    mcp: [],
    tools: {},
    buddy: "off",
    kids: true,
    need: (c) => c.audio && c.media,
    tryPrompt: "Play a short upbeat song",
    tryHint: "You should hear it from the robot, not the computer. Say “stop” when you have heard enough.",
    honest: "This is YouTube on the speaker. Spotify is a separate use — it opens the computer app.",
    tags: ["youtube", "song", "speaker", "play", "music"],
  },
  {
    id: "spotify",
    title: "Spotify on your computer",
    line: "Open Spotify on the computer, not the robot speaker.",
    why: "Needs Kestrel Buddy. Then: “open Spotify”, “play this on Spotify.”",
    category: "media",
    skills: ["spotify", "computer-use"],
    services: [],
    mcp: [],
    tools: {},
    buddy: "required",
    kids: false,
    need: (c) => c.companion,
    tryPrompt: "Open Spotify on my computer",
    tryHint: "Spotify has to be installed on that computer. Playback stays there.",
    honest: "There is no Spotify login on the robot. The computer app is what plays.",
    tags: ["spotify", "playlist", "mac", "windows", "linux", "buddy"],
  },
  {
    id: "stories",
    title: "Stories",
    line: "A short tale out loud, time-capped.",
    why: "Bedtime or “tell me a story.” Gentle when the kids profile is on.",
    category: "media",
    skills: ["stories"],
    services: [],
    mcp: [],
    tools: {},
    buddy: "off",
    kids: true,
    need: (c) => c.audio,
    tryPrompt: "Tell me a very short story",
    tryHint: "One short chapter. Say stop whenever you want.",
    honest: "It makes the tale up, or reads one it knows. Not an audiobook app.",
    tags: ["bedtime", "story", "tale", "kids"],
    flip: { stories: true },
  },
  {
    id: "look",
    title: "Look at this",
    line: "One photo when you ask — the stream does not stay on.",
    why: "“What do you see?” takes a still. Privacy stays camera-on-demand.",
    category: "see",
    skills: ["camera"],
    services: [],
    mcp: [],
    tools: {},
    buddy: "off",
    kids: true,
    need: (c) => c.vision,
    tryPrompt: "What do you see right now?",
    tryHint: "Hold something in front of its camera. It takes one still, then stops.",
    honest: "This is a snapshot, not a live stream.",
    tags: ["camera", "photo", "see", "snapshot", "vision"],
    flip: { look: true },
  },
  {
    id: "dance",
    title: "Dance to a song",
    line: "Music on the speaker plus a groove.",
    why: "Name a track, or just say dance. Off in the office pack.",
    category: "media",
    skills: ["dance", "music"],
    services: [],
    mcp: [],
    tools: {},
    buddy: "off",
    kids: true,
    need: (c) => c.motion && c.media,
    tryPrompt: "Dance for a few seconds",
    tryHint: "You should hear a track and see it move. Say stop to freeze.",
    honest: "The groove is a built-in move, not a choreographed routine.",
    tags: ["dance", "groove", "boogie", "song"],
    flip: { dance: true },
  },
];

function fold(s: string): string {
  return s.toLowerCase().replace(/[-–—\s]/g, "");
}

/** Sidebar search haystack for one use. */
export function scenarioSearchHay(s: Scenario): string {
  return [s.id, s.title, s.line, s.category, ...s.tags].join(" ");
}

export function scenariosMatching(q: string): Scenario[] {
  const nq = fold(q.trim());
  if (!nq) return [];
  return SCENARIOS.filter((s) => fold(scenarioSearchHay(s)).includes(nq));
}

export function scenarioFor(id: string | null | undefined): Scenario | null {
  if (!id) return null;
  return SCENARIOS.find((s) => s.id === id) ?? null;
}

export function scenariosFor(c: GuideCaps): Scenario[] {
  return SCENARIOS.filter((s) => s.need(c));
}

export function scenarioSteps(s: Scenario): ScenarioStepId[] {
  const steps: ScenarioStepId[] = ["intro"];
  if (s.services.length > 0) steps.push("connect");
  if (s.buddy === "required") steps.push("buddy");
  if (s.mcp.length > 0 || Object.keys(s.tools).length > 0) steps.push("tools");
  steps.push("try", "done");
  return steps;
}

export type ScenarioReadyCtx = {
  caps: GuideCaps;
  kids: boolean;
  telegram: boolean;
  buddyPaired: boolean;
  toolsOn: boolean;
};

export type ScenarioStatus = "ready" | "setup" | "needs-computer" | "kids" | "unsupported";

export function scenarioStatus(s: Scenario, ctx: ScenarioReadyCtx): ScenarioStatus {
  if (!s.need(ctx.caps)) return "unsupported";
  if (!s.kids && ctx.kids) return "kids";
  if (s.buddy === "required" && !ctx.buddyPaired) return "needs-computer";
  if (s.services.some((svc) => svc.id === "telegram") && !ctx.telegram) return "setup";
  if ((s.mcp.length > 0 || Object.keys(s.tools).length > 0) && !ctx.toolsOn) return "setup";
  return "ready";
}

export function scenarioStatusLabel(st: ScenarioStatus): string {
  switch (st) {
    case "ready":
      return "Ready";
    case "setup":
      return "Set up";
    case "needs-computer":
      return "Needs a computer";
    case "kids":
      return "Off while kids profile is on";
    case "unsupported":
      return UNSUPPORTED;
  }
}
