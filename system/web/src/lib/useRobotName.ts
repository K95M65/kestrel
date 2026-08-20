import { useEffect, useState } from "react";
import { getDeviceConfig } from "@/lib/api";
import { IDENTITY_EVENT } from "@/lib/robotName";

/** Live IDENTITY / agent_name. Updates when Device → General or guided setup saves a name. */
export function useRobotName(): string {
  const [name, setName] = useState("");
  useEffect(() => {
    getDeviceConfig()
      .then((c) => setName(c.agent_name ?? ""))
      .catch(() => {});
    const on = (e: Event) => {
      const n = (e as CustomEvent<string>).detail;
      if (typeof n === "string") setName(n);
    };
    window.addEventListener(IDENTITY_EVENT, on);
    return () => window.removeEventListener(IDENTITY_EVENT, on);
  }, []);
  return name;
}
