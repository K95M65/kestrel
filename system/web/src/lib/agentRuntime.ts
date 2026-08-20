/** Kind of a swappable agent backend. Companion loop vs coding CLI vs narrow channel. */

export type RuntimeKind = "companion" | "coding" | "telegram" | "unknown";

const COMPANION = new Set(["openclaw", "hermes"]);
const CODING = new Set(["codex", "claudecode", "opencode"]);
const TELEGRAM = new Set(["picoclaw"]);

export function runtimeKind(id: string): RuntimeKind {
  const n = id.trim().toLowerCase();
  if (COMPANION.has(n)) return "companion";
  if (CODING.has(n)) return "coding";
  if (TELEGRAM.has(n)) return "telegram";
  return "unknown";
}

export function runtimeSwitchWarning(id: string): string {
  switch (runtimeKind(id)) {
    case "coding":
      return "This is a coding CLI behind a bridge. Talk, morning brief, and hardware skills will not work the way they do on OpenClaw or Hermes.";
    case "telegram":
      return "PicoClaw is Telegram-only. Slack, Discord, and WhatsApp stay off.";
    default:
      return "";
  }
}
