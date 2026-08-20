/** Live station-network line for Device → Wi-Fi. */

export function formatWifiNow(cur: {
  ssid?: string;
  signal?: number;
  linkRate?: number;
} | null | undefined): string {
  const ssid = (cur?.ssid || "").trim();
  if (!ssid) return "Not on a station network.";
  const bits = [ssid];
  if (cur?.signal) bits.push(`${cur.signal} dBm`);
  if (cur?.linkRate) bits.push(`${cur.linkRate} Mbps`);
  return bits.join(" · ");
}
