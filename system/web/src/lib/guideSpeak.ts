import { testTTSVoice } from "@/lib/api";

/** Speak a short line through the robot during guided setup. Muted speaker
 *  or a missing TTS path is not a failure — the dashboard is the fallback. */
export async function speakGuide(text: string): Promise<"ok" | "muted" | "fail"> {
  const line = text.trim();
  if (!line) return "ok";
  try {
    await testTTSVoice("", { text: line });
    return "ok";
  } catch (err) {
    const msg = err instanceof Error ? err.message : "";
    if (/muted/i.test(msg)) return "muted";
    return "fail";
  }
}
