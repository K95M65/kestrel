// Pure, UI-free helpers for the Setup page. Kept out of the component + the
// controller hook so both can import them without pulling in React state.

// SetupMode controls which sections render. Initial = AP/offline (hide
// online-only enrollments + tests), Continue = LAN/online (the device can hit
// APIs, so Voice/Face enroll + TTS preview become available).
export type SetupMode = "initial" | "continue";

// Go playground/validator returns errors shaped like:
//   "Key: 'SetupRequest.SSID' Error:Field validation for 'SSID' failed on the
//    'required' tag\nKey: 'SetupRequest.LLMAPIKey' Error:Field validation …"
// Surface that as a human-readable list of missing fields so operators don't
// see what looks like a stack trace. Falls through unchanged when the message
// doesn't match the validator format (other backend errors stay as-is).
const FIELD_LABELS: Record<string, string> = {
  SSID: "Wi-Fi name",
  Password: "Wi-Fi password",
  LLMAPIKey: "AI Brain API key",
  LLMBaseURL: "AI Brain URL",
  DeviceID: "Device ID",
};

export function normaliseSetupError(message: string): string {
  const matches = [...message.matchAll(/Field validation for '(\w+)' failed on the '(\w+)' tag/g)];
  if (matches.length === 0) return message;
  const missing: string[] = [];
  const other: string[] = [];
  for (const [, field, tag] of matches) {
    const label = FIELD_LABELS[field] ?? field;
    (tag === "required" ? missing : other).push(label);
  }
  const parts: string[] = [];
  if (missing.length > 0) parts.push(`Missing: ${missing.join(", ")}.`);
  if (other.length > 0) parts.push(`Invalid: ${other.join(", ")}.`);
  parts.push("Re-open Setup from the companion app, or add ?debug=true to enter them manually.");
  return parts.join(" ");
}
