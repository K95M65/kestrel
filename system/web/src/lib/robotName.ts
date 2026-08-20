// Owner-facing robot name. IDENTITY.md / agent_name start as the device class
// ("lamp", "autonomous"); those are not a chosen name.
//
// "Buddy" is the computer companion (Kestrel Buddy), not a default robot name.

const GENERIC = /^(autonomous|friend|lamp|dog|intern|intern-v2|reachy|reachy-mini|reachy mini)$/i;

/** Example on name fields — not Kestrel Buddy, so Talk never looks like the companion. */
export const EXAMPLE_ROBOT_NAME = "Luna";

export const IDENTITY_EVENT = "kestrel-identity";

export function publishRobotName(name: string) {
  if (typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent(IDENTITY_EVENT, { detail: name }));
}

export function displayRobotName(name?: string | null): string {
  const n = (name ?? "").trim();
  if (!n || GENERIC.test(n)) return "your robot";
  return n;
}

export function talkName(name?: string | null): string {
  const n = displayRobotName(name);
  return n === "your robot" ? "the robot" : n;
}

export function talkNameTitle(name?: string | null): string {
  const n = talkName(name);
  return n.charAt(0).toUpperCase() + n.slice(1);
}

export function defaultWakePhrase(name: string): string {
  const n = name.trim().toLowerCase();
  return n ? `hey ${n}` : "";
}

export function isNamedRobot(name?: string | null): boolean {
  return displayRobotName(name) !== "your robot";
}
