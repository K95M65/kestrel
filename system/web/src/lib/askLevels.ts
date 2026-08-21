/** ChatGPT-style connector permission. Default is important_actions. */

export const ASK_LEVELS = [
  { id: "always_ask", title: "Always ask", hint: "Ask before any mail or calendar read or write." },
  { id: "any_changes", title: "Any changes", hint: "Reads are fine. Ask before send, create, or delete." },
  { id: "important_actions", title: "Important actions", hint: "Default. Mail stays a draft. Ask before send." },
  { id: "never_ask", title: "Never ask", hint: "May send when you clearly say to." },
] as const;

export type AskLevel = (typeof ASK_LEVELS)[number]["id"];

export function normalizeAsk(raw?: string, draftNotSend = true): AskLevel {
  const v = (raw || "").trim().toLowerCase();
  if (v === "always_ask" || v === "any_changes" || v === "important_actions" || v === "never_ask") {
    return v;
  }
  return draftNotSend ? "important_actions" : "never_ask";
}

export function draftFromAsk(ask: AskLevel): boolean {
  return ask !== "never_ask";
}

export function claimUrl(origin: string, pin: string): string {
  const base = origin.replace(/\/$/, "");
  return `${base}/claim?pin=${encodeURIComponent(pin)}`;
}
