import { useCallback, useEffect, useState } from "react";
import { QRCodeSVG } from "qrcode.react";
import { INPUT_STYLE, LABEL_STYLE } from "@/components/setup/shared";
import {
  getGoogleStatus, pollGoogleOAuth, setGoogleClient, startGoogleOAuth,
  type GoogleStatus,
} from "@/lib/api";

export function GoogleSignIn({ onChanged }: { onChanged?: () => void }) {
  const [st, setSt] = useState<GoogleStatus | null>(null);
  const [clientId, setClientId] = useState("");
  const [secret, setSecret] = useState("");
  const [showClient, setShowClient] = useState(false);
  const [busy, setBusy] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [login, setLogin] = useState<{
    user_code: string; device_code: string; verification_uri: string; verification_uri_complete?: string;
  } | null>(null);

  const reload = useCallback(async () => {
    try { setSt(await getGoogleStatus()); } catch { /* leave */ }
  }, []);

  useEffect(() => { void reload(); }, [reload]);

  useEffect(() => {
    if (!login) return;
    let stop = false;
    const tick = async () => {
      while (!stop) {
        await new Promise((r) => setTimeout(r, 4000));
        if (stop) return;
        try {
          const p = await pollGoogleOAuth(login.device_code);
          if (!p.pending) {
            setLogin(null);
            await reload();
            onChanged?.();
            return;
          }
        } catch {
          /* keep polling until expiry */
        }
      }
    };
    void tick();
    return () => { stop = true; };
  }, [login, onChanged, reload]);

  async function saveClient() {
    setBusy("client");
    setErr(null);
    try {
      setSt(await setGoogleClient(clientId, secret));
      setSecret("");
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Could not save client.");
    } finally {
      setBusy(null);
    }
  }

  async function start() {
    setBusy("start");
    setErr(null);
    try {
      setLogin(await startGoogleOAuth());
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Could not start Google sign-in.");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="lm-guide-extra" style={{ marginTop: 8 }}>
      <p className="lm-guide-lead" style={{ margin: "0 0 8px" }}>
        Use your Google account for mail and calendar. Same idea as signing in on a TV:
        a code on this page, you confirm on your phone. This does not add the robot to Google Home.
      </p>
      {st?.connected && (
        <p className="lm-guide-lead" style={{ margin: "0 0 8px" }}>
          Signed in as {st.user_email || "Google"}.
        </p>
      )}
      {st?.ready && !login && (
        <button type="button" className="lm-u-btn" disabled={!!busy} onClick={() => void start()}
          style={{ marginTop: 4 }}>
          {busy === "start" ? "Starting…" : st.connected ? "Sign in again" : "Sign in with Google"}
        </button>
      )}
      {login && (
        <div style={{ marginTop: 12, display: "flex", gap: 12, alignItems: "center", flexWrap: "wrap" }}>
          <QRCodeSVG value={login.verification_uri_complete || login.verification_uri} size={88} marginSize={1}
            bgColor="transparent" fgColor="currentColor" title={login.verification_uri} />
          <div>
            <div style={{ fontSize: 18, fontWeight: 700, letterSpacing: 2, fontFamily: "ui-monospace, monospace" }}>
              {login.user_code}
            </div>
            <a href={login.verification_uri} target="_blank" rel="noreferrer" className="lm-wiki-link">
              {login.verification_uri.replace(/^https?:\/\//, "")}
            </a>
            <p className="lm-guide-lead" style={{ margin: "6px 0 0" }}>Open that page, enter the code, then come back.</p>
          </div>
        </div>
      )}
      {!st?.ready && (
        <>
          <p className="lm-guide-lead" style={{ margin: "8px 0" }}>
            Skip for now and paste an app password or iCal address below. Sign-in with Google needs a
            one-time TV client from Google Cloud.
          </p>
          <button type="button" className="lm-wiki-link" style={{ background: "none", border: 0, padding: 0, cursor: "pointer" }}
            onClick={() => setShowClient(!showClient)}>
            {showClient ? "Hide Google Cloud client" : "I have a Google Cloud TV client"}
          </button>
          {showClient && (
            <div style={{ marginTop: 8 }}>
              <label htmlFor="g-cid" style={LABEL_STYLE}>OAuth client ID</label>
              <input id="g-cid" value={clientId} onChange={(e) => setClientId(e.target.value)}
                placeholder="….apps.googleusercontent.com" autoComplete="off" style={INPUT_STYLE} />
              <label htmlFor="g-sec" style={LABEL_STYLE}>Client secret</label>
              <input id="g-sec" type="password" value={secret} onChange={(e) => setSecret(e.target.value)}
                placeholder={st?.has_secret ? "saved" : ""} autoComplete="off" style={INPUT_STYLE} />
              <button type="button" className="lm-u-btn" disabled={!!busy || !clientId.trim()}
                onClick={() => void saveClient()} style={{ marginTop: 8 }}>
                {busy === "client" ? "Saving…" : "Save client"}
              </button>
            </div>
          )}
        </>
      )}
      <p className="lm-guide-lead" style={{ margin: "12px 0 0" }}>
        Sign in with Apple needs a public https:// page. On this desk, use Google.
      </p>
      {err && <p className="lm-guide-lead" style={{ color: "var(--lm-red)", margin: "8px 0 0" }}>{err}</p>}
    </div>
  );
}
