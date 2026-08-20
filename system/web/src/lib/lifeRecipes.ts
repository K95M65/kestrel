/** OS-owned life packs. Brains consume skills/connectors; they do not own this catalog. */

export type LifeId = "desk" | "family" | "kids" | "office";

export type RecipeServiceId = "google_calendar" | "gmail" | "telegram";

export type RecipeService = {
  id: RecipeServiceId;
  title: string;
  why: string;
  kind: "connector" | "channel";
  /** pat = Gmail app password. ical = Google Calendar secret address. token = Telegram. */
  auth: "pat" | "ical" | "token";
};

export type LifeRecipe = {
  id: LifeId;
  title: string;
  line: string;
  services: RecipeService[];
  policy: { draft_not_send: boolean; kids: boolean };
  buddy: "off" | "optional";
};

const CALENDAR: RecipeService = {
  id: "google_calendar",
  title: "Calendar",
  why: "Used in the morning brief. Paste the secret iCal address from Google Calendar.",
  kind: "connector",
  auth: "ical",
};
const GMAIL: RecipeService = {
  id: "gmail",
  title: "Gmail",
  why: "Overnight mail in the brief — drafts only, it never sends.",
  kind: "connector",
  auth: "pat",
};
const TELEGRAM: RecipeService = {
  id: "telegram",
  title: "Telegram",
  why: "Text it when you are not at the desk.",
  kind: "channel",
  auth: "token",
};

export const LIFE_RECIPES: Record<LifeId, LifeRecipe> = {
  desk: {
    id: "desk",
    title: "Just me",
    line: "A companion for one person — briefing, memory, dance, a hello when you sit down.",
    services: [CALENDAR, GMAIL, TELEGRAM],
    policy: { draft_not_send: true, kids: false },
    buddy: "optional",
  },
  family: {
    id: "family",
    title: "Family",
    line: "Kitchen presence — greet whoever walks in, meals, stories if asked.",
    services: [CALENDAR, TELEGRAM],
    policy: { draft_not_send: true, kids: false },
    buddy: "optional",
  },
  kids: {
    id: "kids",
    title: "Kids around",
    line: "Gentle and bounded. No mail, calendar, or the house. Stories on.",
    services: [],
    policy: { draft_not_send: true, kids: true },
    buddy: "off",
  },
  office: {
    id: "office",
    title: "Office",
    line: "Named greetings, morning brief, focus nag, pomodoro. Dance stays off.",
    services: [CALENDAR, GMAIL, TELEGRAM],
    policy: { draft_not_send: true, kids: false },
    buddy: "optional",
  },
};

export function recipeFor(id: string | null | undefined): LifeRecipe | null {
  if (!id) return null;
  return LIFE_RECIPES[id as LifeId] ?? null;
}

export function lifeHasConnect(id: string | null | undefined): boolean {
  const r = recipeFor(id);
  return !!r && r.services.length > 0;
}
