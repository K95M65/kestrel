import { useCallback, useEffect, useState } from "react";
import { QRCodeSVG } from "qrcode.react";
import { INPUT_STYLE, LABEL_STYLE } from "@/components/setup/shared";
import {
  getApple, getBuzz, sayBuzz, setAppleClient, setBuzz, startAppleOAuth,
  type AppleStatus, type BuzzStatus,
} from "@/lib/api";

function hiveJoinFallback(): string {
  if (typeof window === "undefined") return "";
  return window.location.origin.replace(/^http/, "ws") + "/api/buzz/ws";
}

export function HiveCard() {
  const [buzz, setSt] = useState<BuzzStatus | null>(null);
  const [apple, setApple] = useState<AppleStatus | null>(null);
  const [relay, setRelay] = useState("");
  const [host, setHost] = useState(false);
  const [on, setOn] = useState(false);
  const [say, setSay] = useState("");
  const [sid, setSid] = useState("");
  const [team, setTeam] = useState("");
  const [kid, setKid] = useState("");
  const [p8, setP8] = useState("");
  const [ret, setRet] = useState("");
  const [copied, setCopied] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const isDebug = typeof window !== "undefined" && new URLSearchParams(window.location.search).get("debug") === "true";

  const reload = useCallback(async () => {
    try {
      const b = await getBuzz();
      setSt(b);
      setOn(b.enabled);
      setHost(b.host);
      setRelay(b.relay_url || "");
    } catch { /* */ }
    try {
      const a = await getApple();
      setApple(a);
      if (a.return_url) setRet(a.return_url);
    } catch { /* */ }
  }, []);
  useEffect(() => { void reload(); }, [reload]);

  const join = (buzz?.join_url || hiveJoinFallback()).trim();

  function copyJoin() {
    if (!join) return;
    const done = () => { setCopied(true); window.setTimeout(() => setCopied(false), 1600); };
    if (navigator.clipboard?.writeText) {
      void navigator.clipboard.writeText(join).then(done).catch(() => {});
      return;
    }
    done();
  }

  return (
    <div style={{ marginTop: 22, paddingTop: 18, borderTop: "1px solid var(--lm-border)" }}>
      <div style={{ fontSize: 13, fontWeight: 650, marginBottom: 4 }}>Talk to another robot</div>
      <p className="lm-guide-lead">
        Two robots on this Wi-Fi can hear each other in Talk. One hosts. The other pastes the join address.
        This is not adding a bulb to Apple Home or Google Home.
      </p>
      {buzz && (
        <p className="lm-guide-lead">{buzz.ready ? `Connected · ${buzz.peers} other robot(s)` : "Off until you save."}</p>
      )}
      <label style={{ display: "flex", gap: 8, alignItems: "center", fontSize: 13, margin: "8px 0" }}>
        <input type="checkbox" checked={on} onChange={(e) => setOn(e.target.checked)} /> On
      </label>
      <label style={{ display: "flex", gap: 8, alignItems: "center", fontSize: 13, margin: "8px 0" }}>
        <input type="checkbox" checked={host} onChange={(e) => setHost(e.target.checked)} /> This robot hosts
      </label>
      {host && join && (
        <div style={{ display: "flex", gap: 12, alignItems: "center", flexWrap: "wrap", margin: "8px 0 12px" }}>
          <QRCodeSVG value={join} size={88} marginSize={1} bgColor="transparent" fgColor="currentColor" title={join} />
          <div>
            <p className="lm-guide-lead" style={{ margin: "0 0 6px" }}>Share this with the other robot, like a pairing code.</p>
            <button type="button" className="lm-u-btn" onClick={copyJoin}
              style={{ fontSize: 12, fontFamily: "ui-monospace, monospace" }}>
              {copied ? "Copied" : join.replace(/^wss?:\/\//, "")}
            </button>
          </div>
        </div>
      )}
      {!host && (
        <>
          <label htmlFor="buzz-url" style={LABEL_STYLE}>Join address from the host</label>
          <input id="buzz-url" value={relay} onChange={(e) => setRelay(e.target.value)}
            placeholder="ws://10.10.2.160/api/buzz/ws" style={INPUT_STYLE} />
        </>
      )}
      <button type="button" className="lm-u-btn" disabled={!!busy} style={{ marginTop: 8 }}
        onClick={() => {
          setBusy("buzz");
          setBuzz({ enabled: on, host, relay_url: host ? "" : relay })
            .then(() => reload())
            .catch((e: Error) => setErr(e.message))
            .finally(() => setBusy(null));
        }}>{busy === "buzz" ? "…" : "Save"}</button>
      <div style={{ display: "flex", gap: 8, marginTop: 10 }}>
        <input value={say} onChange={(e) => setSay(e.target.value)} placeholder="Say something the other robot can hear"
          style={{ ...INPUT_STYLE, flex: 1 }} />
        <button type="button" className="lm-u-btn" disabled={!!busy || !say.trim()}
          onClick={() => { setBusy("say"); sayBuzz(say).then(() => setSay("")).catch((e: Error) => setErr(e.message)).finally(() => setBusy(null)); }}>
          Send
        </button>
      </div>

      {isDebug && (
        <>
          <div style={{ fontSize: 13, fontWeight: 650, margin: "22px 0 4px" }}>Sign in with Apple (Advanced)</div>
          <p className="lm-guide-lead">{apple?.hint || "Needs an Apple Developer Services ID and https:// return URL."}</p>
          {apple?.user_email && <p className="lm-guide-lead">Signed in as {apple.user_email}.</p>}
          <label htmlFor="ap-sid" style={LABEL_STYLE}>Services ID</label>
          <input id="ap-sid" value={sid} onChange={(e) => setSid(e.target.value)} placeholder="com.example.kestrel" style={INPUT_STYLE} />
          <label htmlFor="ap-team" style={LABEL_STYLE}>Team ID</label>
          <input id="ap-team" value={team} onChange={(e) => setTeam(e.target.value)} placeholder="ABCD1234" style={INPUT_STYLE} />
          <label htmlFor="ap-kid" style={LABEL_STYLE}>Key ID</label>
          <input id="ap-kid" value={kid} onChange={(e) => setKid(e.target.value)} placeholder="KEYID" style={INPUT_STYLE} />
          <label htmlFor="ap-p8" style={LABEL_STYLE}>.p8 private key</label>
          <textarea id="ap-p8" value={p8} onChange={(e) => setP8(e.target.value)} placeholder="-----BEGIN PRIVATE KEY-----" rows={4} style={INPUT_STYLE} />
          <label htmlFor="ap-ret" style={LABEL_STYLE}>HTTPS return URL</label>
          <input id="ap-ret" value={ret} onChange={(e) => setRet(e.target.value)}
            placeholder="https://…/api/auth/apple/callback" style={INPUT_STYLE} />
          <button type="button" className="lm-u-btn" disabled={!!busy} style={{ marginTop: 8 }}
            onClick={() => {
              setBusy("apple");
              setAppleClient({ services_id: sid, team_id: team, key_id: kid, private_key: p8, return_url: ret })
                .then((a) => { setApple(a); setP8(""); })
                .catch((e: Error) => setErr(e.message))
                .finally(() => setBusy(null));
            }}>{busy === "apple" ? "…" : "Save Apple client"}</button>
          {apple?.ready && (
            <button type="button" className="lm-u-btn" disabled={!!busy} style={{ marginTop: 8, marginLeft: 8 }}
              onClick={() => {
                setBusy("siwa");
                startAppleOAuth()
                  .then((r) => { window.open(r.url, "_blank", "noopener"); })
                  .catch((e: Error) => setErr(e.message))
                  .finally(() => setBusy(null));
              }}>{busy === "siwa" ? "…" : "Sign in with Apple"}</button>
          )}
        </>
      )}
      {err && <p className="lm-guide-lead" style={{ color: "var(--lm-red)" }}>{err}</p>}
    </div>
  );
}
