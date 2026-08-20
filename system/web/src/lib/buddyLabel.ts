/** Label the paired computer from the buddy's os_version string. */

export function buddyOSLabel(osVersion?: string | null): string {
  const s = (osVersion || "").toLowerCase();
  if (s.includes("windows")) return "Windows";
  if (s.includes("linux") || s.includes("ubuntu") || s.includes("debian") || s.includes("fedora") || s.includes("arch")) {
    return "Linux";
  }
  if (s.includes("darwin") || s.includes("macos") || s.includes("mac os") || s.includes("os x")) {
    return "macOS";
  }
  return osVersion?.trim() ? "Computer" : "Computer";
}

export function buddyKind<T extends { kind?: string; id?: string }>(apps: T[]): T[] {
  return apps.filter((a) => a.kind === "buddy" || (a.id || "").startsWith("autonomous-buddy"));
}
