// Owner-facing robot name. IDENTITY.md / agent_name start as the device class
// ("lamp", "autonomous"); those are not a chosen name.

const GENERIC = /^(autonomous|friend|lamp|dog|intern|intern-v2|reachy|reachy-mini|reachy mini)$/i;

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
