import { getSleep, wakeNow } from "./api";
import { isRobotQuiet } from "./sleepToggle";
import { HW } from "@/pages/monitor/types";

function wait(ms: number) {
  return new Promise<void>((r) => setTimeout(r, ms));
}

async function postHw(path: string, body?: unknown): Promise<void> {
  await fetch(`${HW}${path}`, {
    method: "POST",
    headers: body ? { "Content-Type": "application/json" } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
}

/** Wake from quiet hours / sleep so later HAL calls aren't no-ops. */
export async function wakeIfQuiet(onStatus?: (line: string) => void): Promise<boolean> {
  let sleeping = false;
  try {
    const st = await getSleep();
    sleeping = isRobotQuiet(st.sleeping, st.emotion);
  } catch {
    sleeping = false;
  }
  if (!sleeping) return false;

  onStatus?.("Waking up…");
  try {
    await wakeNow();
  } catch {
    /* still try the body — HAL may already be mid-wake */
  }
  await wait(400);
  try { await postHw("/servo/resume"); } catch { /* keep going */ }
  return true;
}

/** If the body is in quiet hours, wake it and look at the person so the camera isn't on the floor. */
export async function wakeAndPoseForPhoto(onStatus?: (line: string) => void): Promise<boolean> {
  const woke = await wakeIfQuiet(onStatus);
  if (!woke) return false;
  try { await postHw("/camera/enable"); } catch { /* keep going */ }

  onStatus?.("Looking at you…");
  try {
    await postHw("/servo/aim", { direction: "user", duration: 1.5 });
  } catch { /* pose is best-effort */ }
  await wait(1600);
  try { await postHw("/servo/hold"); } catch { /* countdown can still run */ }
  return true;
}

export async function releasePhotoPose(): Promise<void> {
  try { await postHw("/servo/resume"); } catch { /* ignore */ }
}
