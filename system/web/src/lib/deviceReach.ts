/** URLs another computer on the LAN can use to open this robot. */

export type ReachInput = {
  /** STA IPv4 from /api/system/network. */
  ip?: string | null;
  /** config device_id / GetDeviceMac form, e.g. "reachy-mini-a1b2". */
  hostId?: string | null;
  /** Current page origin — used only to copy a non-default port. */
  origin?: string | null;
};

export type ReachUrls = {
  /** Best URL to share. LAN IP first — many routers drop mDNS. */
  primary: string;
  lan: string;
  mdns: string;
};

const HOST_ID = /^[a-z0-9]+(?:-[a-z0-9]+)*-[0-9a-f]{4}$/;

export function mdnsHost(hostId?: string | null): string {
  const h = (hostId || "").trim().toLowerCase();
  if (!HOST_ID.test(h)) return "";
  return `${h}.local`;
}

export function originPort(origin?: string | null): string {
  if (!origin) return "";
  try {
    const p = new URL(origin).port;
    if (!p || p === "80" || p === "443") return "";
    return `:${p}`;
  } catch {
    return "";
  }
}

export function isLoopbackOrigin(origin?: string | null): boolean {
  if (!origin) return false;
  try {
    const h = new URL(origin).hostname;
    return h === "localhost" || h === "127.0.0.1" || h === "[::1]" || h === "::1";
  } catch {
    return false;
  }
}

function httpUrl(host: string, origin?: string | null): string {
  if (!host) return "";
  return `http://${host}${originPort(origin)}`;
}

export function deviceReach(input: ReachInput): ReachUrls {
  const mdnsName = mdnsHost(input.hostId);
  const ip = (input.ip || "").trim();
  const lan = ip ? httpUrl(ip, input.origin) : "";
  const mdns = mdnsName ? httpUrl(mdnsName, input.origin) : "";
  const origin = (input.origin || "").replace(/\/$/, "");
  const primary = lan || mdns || (!isLoopbackOrigin(origin) ? origin : "");
  return { primary, lan, mdns };
}
