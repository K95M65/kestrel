import { useEffect, useRef, useState } from "react";
import { checkInternet, getCurrentNetwork } from "@/lib/api";

// Confirms — from the DEVICE's actual live state, not from ephemeral form
// input — whether the device is already on home Wi-Fi. This is what lets the
// Setup page mark the Wi-Fi step "done" after the AP→STA join reloads the page
// on the new LAN-IP origin with empty React state.
//
// Why not read the saved SSID from /api/device/config? That endpoint is
// admin-gated and a fresh device has no admin session yet, so the reloaded
// page gets a 401 and can't rehydrate. Both signals used here are public:
//   - checkInternet()      → the device has an uplink. The setup AP has no
//                            uplink, so internet == the device left AP mode
//                            and joined a real network.
//   - getCurrentNetwork()  → the SSID wlan0 is presently associated with
//                            (`iwgetid -r`). Non-empty == station mode.
// We require BOTH: internet alone can't name the network, and an associated
// SSID alone could theoretically be the device's own AP hotspot. Together they
// are unambiguous proof the device is on home Wi-Fi, matching the same
// internet-based signal SetupGate uses to pick continue vs initial mode.
//
// Returns:
//   wifiConnected — true once the device is confirmed on home Wi-Fi.
//   currentSsid   — the associated SSID, so the caller can prefill the Wi-Fi
//                   picker instead of showing an empty "choose your Wi-Fi".
//   checking      — true while the FIRST probe is still in flight (nothing
//                   decided yet). The caller shows a skeleton during this window
//                   so the Wi-Fi step doesn't flash "Choose your Wi-Fi" before
//                   we know the device is already connected. Flips false after
//                   the first probe resolves — later background retries (for the
//                   DHCP-lease race) don't re-raise it, so the picker stays
//                   interactive instead of re-blanking under the operator.
//
// Polls a few times because wlan0 may not have finished associating / gotten a
// DHCP lease in the first instant after the reload; once confirmed it stops.
export function useWifiConnected(): { wifiConnected: boolean; currentSsid: string; checking: boolean } {
  const [wifiConnected, setWifiConnected] = useState(false);
  const [currentSsid, setCurrentSsid] = useState("");
  // Starts true: the very first render is "we don't know yet", which is exactly
  // the skeleton state. Cleared once the first probe resolves either way.
  const [checking, setChecking] = useState(true);
  const doneRef = useRef(false);

  useEffect(() => {
    let cancelled = false;
    let attempt = 0;
    let timer: number | undefined;

    const check = async () => {
      attempt += 1;
      try {
        const [online, current] = await Promise.all([
          checkInternet().catch(() => false),
          getCurrentNetwork().catch(() => null),
        ]);
        if (cancelled) return;
        const ssid = current?.ssid ?? "";
        if (online && ssid) {
          doneRef.current = true;
          setCurrentSsid(ssid);
          setWifiConnected(true);
          setChecking(false);
          return; // confirmed — stop polling
        }
      } catch {
        /* transient — retry below */
      }
      if (cancelled || doneRef.current) return;
      // First probe resolved without a confirmed connection: stop blocking the
      // UI with the skeleton and show the picker. Background retries below still
      // run silently in case the DHCP lease is just landing.
      setChecking(false);
      // A handful of retries covers the brief window where the interface is
      // associated but the DHCP lease / route isn't up yet. After that we stop
      // and leave wifiConnected=false; the form still works via manual input.
      if (attempt < 5) timer = window.setTimeout(check, 1500);
    };

    check();
    return () => { cancelled = true; if (timer) window.clearTimeout(timer); };
    // Mount-only, like the other setup hooks.
  }, []);

  return { wifiConnected, currentSsid, checking };
}
