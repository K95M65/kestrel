/** Home / Quiet hours Sleep now becomes Wake when the body is already quiet. */

import type { SleepStatus } from "./api";

export function isRobotQuiet(sleeping?: boolean | null, emotion?: string | null): boolean {
  if (sleeping === true) return true;
  if (sleeping === false) return false;
  return (emotion ?? "").trim().toLowerCase() === "sleepy";
}

export function sleepToggleKind(quiet: boolean): "sleep" | "wake" {
  return quiet ? "wake" : "sleep";
}

export function sleepToggleLabel(quiet: boolean): string {
  return quiet ? "Wake now" : "Sleep now";
}

export function withSleeping(status: SleepStatus | null, sleeping: boolean): SleepStatus {
  if (status) return { ...status, sleeping };
  return {
    sleeping,
    scheduled: false,
    schedule: { enabled: false, sleep_at: "23:00", wake_at: "07:00" },
  };
}
