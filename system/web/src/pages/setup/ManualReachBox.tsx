import { useState } from "react";
import { C, INPUT_STYLE } from "@/components/setup/shared";

// ManualReachBox — last-resort recovery shown on the "joining Wi-Fi…" screen
// when the join is dragging on AND we never captured the device's LAN IP (the
// AP died before the status poller could read it). Without a lan_ip there's no
// address to auto-redirect to, so we let the operator type the IP they find in
// their router's admin page and open it directly. Carries the original query
// params so the new host rehydrates state + re-auth, mirroring the auto-redirect.
export function ManualReachBox({ carrySearch }: { carrySearch: string }) {
  const [ip, setIp] = useState("");
  const trimmed = ip.trim();
  // Loose IPv4 shape check — enough to avoid opening obviously-bad input without
  // being strict about ranges (the device's router-assigned IP is whatever DHCP
  // gave it). Empty stays disabled.
  const looksValid = /^\d{1,3}(\.\d{1,3}){3}$/.test(trimmed);
  const open = () => {
    if (!looksValid) return;
    window.location.href = `http://${trimmed}${window.location.pathname}${carrySearch}`;
  };
  return (
    <div style={{
      marginTop: 18, paddingTop: 16,
      borderTop: `1px solid ${C.border}`, textAlign: "left",
    }}>
      <div style={{ fontSize: 13, color: C.textDim, marginBottom: 8, lineHeight: 1.5 }}>
        Taking a while? Your device is likely on your Wi-Fi now. Find its IP in
        your router's admin page, then reconnect to your home Wi-Fi and open it
        here:
      </div>
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <input
          value={ip}
          onChange={(e) => setIp(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Enter") open(); }}
          placeholder="192.168.1.x"
          inputMode="decimal"
          autoComplete="off"
          style={{
            ...INPUT_STYLE,
            flex: 1,
            fontFamily: "ui-monospace, monospace",
          }}
        />
        <button
          type="button"
          onClick={open}
          disabled={!looksValid}
          className="lm-btn lm-btn-primary"
          style={{ padding: "9px 16px", flexShrink: 0 }}
        >
          Open →
        </button>
      </div>
    </div>
  );
}
