import { Wifi, XCircle, CheckCircle2 } from "lucide-react";
import { C } from "@/components/setup/shared";
import { getInitialSearch } from "@/hooks/setup/useSetupUrlParams";
import { setupBridge } from "@/lib/setupBridge";
import { CopyAddress } from "./CopyAddress";
import { ManualReachBox } from "./ManualReachBox";

// Post-submit screen: shows progress while the device joins Wi-Fi, then a
// copyable IP / router hint for the operator to continue setup on the home
// network once the AP shuts down. Split out of Setup.tsx because it's the
// single largest JSX block and has three self-contained phase branches
// (connecting / connected / failed).
export function SetupProgressScreen({
  setupPhase, setupLanIP, setupErrorMsg, elapsed,
  deviceMdnsHost, deviceTypePrefix,
  onRetry,
}: {
  setupPhase: "connecting" | "connected" | "failed";
  setupLanIP: string;
  setupErrorMsg: string;
  elapsed: number;
  deviceMdnsHost: string;
  deviceTypePrefix: string;
  // Resets the wizard back to the Wi-Fi step after a failed join.
  onRetry: () => void;
}) {
  return (
    <div className="lm-card lm-fade-in" style={{
      padding: "32px 24px", textAlign: "center",
    }}>
      {setupPhase === "connecting" && (
        <>
          <div style={{ display: "flex", justifyContent: "center", marginBottom: 14 }}>
            <span className="lm-wifi-pulse" aria-hidden>
              <span className="lm-wifi-ring" />
              <span className="lm-wifi-ring lm-r2" />
              <span className="lm-wifi-ring lm-r3" />
              <span className="lm-wifi-icon">
                <Wifi size={26} strokeWidth={2} />
              </span>
            </span>
          </div>
          <div style={{ fontSize: 14.5, fontWeight: 600, color: C.amber, marginBottom: 8 }}>
            Your device is joining Wi-Fi
            <span className="lm-blink">.</span><span>.</span><span>.</span>
          </div>
          <div style={{ fontSize: 13, color: C.textDim, marginBottom: 14, lineHeight: 1.5 }}>
            This usually takes 10-30 seconds. Stay on this network.
          </div>
          {/* Indeterminate progress + elapsed counter: the join has no
              knowable %, so a sweeping bar signals "working" while the
              seconds give the wait a measured feel. */}
          <div className="lm-indeterminate" style={{ marginBottom: 7 }} />
          <div style={{ fontSize: 11, color: C.textMuted, marginBottom: setupLanIP ? 18 : 0 }}>
            Elapsed {elapsed}s
          </div>
          {/* Surface the raw-IP address NOW, while we still have a
              connection — once the operator switches to home Wi-Fi
              this page goes away and an un-copied address is lost.
              The auto-redirect on "connected" still handles the happy
              path; this is the safety net for when it doesn't land
              (AP dropping before the phase poll flips).
              IP-only by design: the .local name is unreliable on
              mDNS-blocking networks, so we show nothing until the
              backend's early-capture poll hands us a LAN IP.
              Toned down here vs. the "connected" screen: a single
              compact label + the copy field, no long paragraph, so it
              stays a quiet safety net rather than competing with the
              primary "joining…" message. */}
          {setupLanIP && (
            <div style={{
              marginTop: 4, paddingTop: 14,
              borderTop: `1px solid ${C.border}`,
              textAlign: "left",
            }}>
              <div style={{
                fontSize: 13, color: C.textDim, marginBottom: 6, lineHeight: 1.5,
              }}>
                This page disconnects when you rejoin home Wi-Fi — save
                this address to continue:
              </div>
              <CopyAddress url={`http://${setupLanIP}/setup`} />
            </div>
          )}

          {/* Last-resort recovery. If the join is taking a long time
              and we still never captured a LAN IP (the AP died before
              the poller could read it — the stranding bug's worst
              case), the operator has no auto path forward. After 25s
              we reveal a manual escape: look up the device IP in the
              router and open it directly. Only shows when there's no
              lan_ip to offer above, so it never competes with the
              happy-path copy field. */}
          {!setupLanIP && elapsed >= 25 && (
            <ManualReachBox carrySearch={getInitialSearch()} />
          )}
        </>
      )}

      {setupPhase === "connected" && (
        <>
          <div style={{ display: "flex", justifyContent: "center", marginBottom: 14 }}>
            <CheckCircle2 size={34} color={C.green} strokeWidth={1.75} aria-hidden />
          </div>
          <div style={{ fontSize: 14.5, fontWeight: 600, color: C.amber, marginBottom: 16 }}>
            Your device is online!
          </div>

          {/* IP path (only path): we redirect to the device's raw LAN
              IP, never its `.local` mDNS name — `.local` is unreliable
              on mDNS-blocking routers, whereas an IP resolves on every
              network. Shown once the backend's early-capture poll has
              handed us a LAN IP; otherwise we fall back to a
              router-admin hint so the operator can find the IP. */}
          {setupLanIP ? (
            <>
              {/* Action-first ordering: the one thing the user must do
                  now (rejoin home Wi-Fi, then Continue) leads, with the
                  primary button right under it. The IP address + router
                  fallback drop below a divider as a quiet safety-net for
                  when auto-redirect/Continue doesn't land — mirroring
                  the connecting screen's hierarchy. */}
              <div style={{ fontSize: 13, color: C.textDim, marginBottom: 16, lineHeight: 1.5 }}>
                Reconnect your computer to your home Wi-Fi, then click
                Continue.
              </div>
              <a
                // Carry the current pathname + query params so any
                // ?llm_api_key=… etc. from the OS server remain in scope on
                // the new host (redundant — the OS server already persisted
                // them via submit — but cheap and useful when the
                // operator re-runs setup with different overrides).
                // Force reload when the user is already on the device's
                // IP — otherwise the browser no-ops the same-URL click
                // and they stay stuck on the "Your device is online!"
                // screen even though the device is reachable in continue
                // mode now.
                href={`http://${setupLanIP}${window.location.pathname}${getInitialSearch()}`}
                onClick={(e) => {
                  setupBridge.continueClicked({ mdns_host: deviceMdnsHost });
                  if (window.location.hostname === setupLanIP) {
                    e.preventDefault();
                    window.location.reload();
                  }
                }}
                className="lm-btn lm-btn-primary"
                style={{
                  display: "inline-block", padding: "10px 22px",
                  textDecoration: "none",
                }}
              >
                Continue setup →
              </a>
              {/* Safety-net block: divider + the IP address and a
                  router-admin hint, toned down so it doesn't compete
                  with the Continue button above. */}
              <div style={{
                marginTop: 18, paddingTop: 16,
                borderTop: `1px solid ${C.border}`, textAlign: "left",
              }}>
                <div style={{ fontSize: 13, color: C.textDim, marginBottom: 6, lineHeight: 1.5 }}>
                  Or open this address once you're back on home Wi-Fi:
                </div>
                <CopyAddress url={`http://${setupLanIP}/setup`} />
                <div style={{ fontSize: 12, color: C.textMuted, marginTop: 8, lineHeight: 1.5 }}>
                  Can't reach it? Find your device's IP in your router's
                  admin page{deviceTypePrefix ? ` (look for "${deviceTypePrefix}")` : ""}.
                </div>
              </div>
            </>
          ) : (
            <div style={{ fontSize: 13, color: C.textDim, lineHeight: 1.5 }}>
              Your device is connected. Open your router's admin page to find
              the device's IP address{deviceTypePrefix ? ` (look for "${deviceTypePrefix}")` : ""}.
            </div>
          )}
        </>
      )}

      {setupPhase === "failed" && (
        <>
          <div style={{ display: "flex", justifyContent: "center", marginBottom: 14 }}>
            <XCircle size={34} color={C.red} strokeWidth={1.75} aria-hidden />
          </div>
          <div style={{ fontSize: 14.5, fontWeight: 600, color: C.red, marginBottom: 8 }}>
            Wi-Fi setup failed
          </div>
          <div style={{ fontSize: 13, color: C.textDim, marginBottom: 16, lineHeight: 1.5 }}>
            {setupErrorMsg || "Couldn't connect to the network you chose."}
          </div>

          {/* Actionable checklist. Wi-Fi join failures on these
              devices are overwhelmingly one of these three causes, so
              we spell them out instead of a generic "try again" —
              the 2.4GHz one in particular is non-obvious to most
              people and the single most common cause. */}
          <div style={{
            textAlign: "left", background: C.surface,
            border: `1px solid ${C.border}`, borderRadius: 8,
            padding: "12px 14px", marginBottom: 18, fontSize: 13,
            color: C.textDim, lineHeight: 1.6,
          }}>
            <div style={{ fontWeight: 600, color: C.text, marginBottom: 6 }}>
              Things to check:
            </div>
            <div>• Use a <strong style={{ color: C.text }}>2.4GHz</strong> Wi-Fi network — most devices can't join 5GHz.</div>
            <div>• Double-check the Wi-Fi password (it's case-sensitive).</div>
            <div>• Keep the device close to your router during setup.</div>
          </div>

          <button
            type="button"
            className="lm-btn lm-btn-primary"
            onClick={onRetry}
            style={{ padding: "9px 18px" }}
          >
            Back to Wi-Fi
          </button>
        </>
      )}
    </div>
  );
}
