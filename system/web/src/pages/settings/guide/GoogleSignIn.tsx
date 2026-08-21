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
        Sign in with Google for mail and calendar. Needs a Google Cloud OAuth client
        of type <strong>TV and limited input</strong>. Sign in with Apple is not in this build.
      </p>
      {st?.connected && (
        <p className="lm-guide-lead" style={{ margin: "0 0 8px" }}>
          Signed in as {st.user_email || "Google"}.
        </p>
      )}
      {!st?.ready && (
        <>
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
        </>
      )}
      {st?.ready && !login && (
        <button type="button" className="lm-u-btn" disabled={!!busy} onClick={() => void start()}
          style={{ marginTop: 8 }}>
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
            <p className="lm-guide-lead" style={{ margin: "6px 0 0" }}>Enter that code, then return here.</p>
          </div>
        </div>
      )}
      {err && <p className="lm-guide-lead" style={{ color: "var(--lm-red)", margin: "8px 0 0" }}>{err}</p>}
    </div>
  );
}
