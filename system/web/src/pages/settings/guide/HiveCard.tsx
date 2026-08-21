import { useCallback, useEffect, useState } from "react";
import { INPUT_STYLE, LABEL_STYLE } from "@/components/setup/shared";
import {
  commissionMatter, getApple, getBuzz, getMatter, sayBuzz, setAppleClient, setBuzz, startAppleOAuth,
  type AppleStatus, type BuzzStatus, type MatterStatus,
} from "@/lib/api";

export function HiveCard() {
  const [buzz, setSt] = useState<BuzzStatus | null>(null);
  const [matter, setMatter] = useState<MatterStatus | null>(null);
  const [apple, setApple] = useState<AppleStatus | null>(null);
  const [relay, setRelay] = useState("");
  const [host, setHost] = useState(false);
  const [on, setOn] = useState(false);
  const [say, setSay] = useState("");
  const [code, setCode] = useState("");
  const [sid, setSid] = useState("");
  const [team, setTeam] = useState("");
  const [kid, setKid] = useState("");
  const [p8, setP8] = useState("");
  const [ret, setRet] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  const reload = useCallback(async () => {
    try {
      const b = await getBuzz();
      setSt(b);
      setOn(b.enabled);
      setHost(b.host);
      setRelay(b.relay_url || "");
    } catch { /* */ }
    try { setMatter(await getMatter()); } catch { /* */ }
    try {
      const a = await getApple();
      setApple(a);
      if (a.return_url) setRet(a.return_url);
    } catch { /* */ }
  }, []);
  useEffect(() => { void reload(); }, [reload]);

  return (
    <div style={{ marginTop: 22, paddingTop: 18, borderTop: "1px solid var(--lm-border)" }}>
      <div style={{ fontSize: 13, fontWeight: 650, marginBottom: 4 }}>Hive (Buzz)</div>
      <p className="lm-guide-lead">
        Other Kestrels on this network join this body the way agents join a Block Buzz room.
        Host here, or paste another unit's <code>ws://…/api/buzz/ws</code>. A full Block Buzz relay is optional later.
      </p>
      {buzz && (
        <p className="lm-guide-lead">{buzz.ready ? `Hive up · ${buzz.peers} peer(s)` : "Hive off until you save."}</p>
      )}
      <label style={{ display: "flex", gap: 8, alignItems: "center", fontSize: 13, margin: "8px 0" }}>
        <input type="checkbox" checked={on} onChange={(e) => setOn(e.target.checked)} /> On
      </label>
      <label style={{ display: "flex", gap: 8, alignItems: "center", fontSize: 13, margin: "8px 0" }}>
        <input type="checkbox" checked={host} onChange={(e) => setHost(e.target.checked)} /> This robot hosts
      </label>
      <label htmlFor="buzz-url" style={LABEL_STYLE}>Join URL (if not hosting)</label>
      <input id="buzz-url" value={relay} onChange={(e) => setRelay(e.target.value)}
        placeholder="ws://10.10.2.160/api/buzz/ws" style={INPUT_STYLE} />
      <button type="button" className="lm-u-btn" disabled={!!busy} style={{ marginTop: 8 }}
        onClick={() => {
          setBusy("buzz");
          setBuzz({ enabled: on, host, relay_url: relay })
            .then(() => reload())
            .catch((e: Error) => setErr(e.message))
            .finally(() => setBusy(null));
        }}>{busy === "buzz" ? "…" : "Save hive"}</button>
      <div style={{ display: "flex", gap: 8, marginTop: 10 }}>
        <input value={say} onChange={(e) => setSay(e.target.value)} placeholder="Tell the hive…" style={{ ...INPUT_STYLE, flex: 1 }} />
        <button type="button" className="lm-u-btn" disabled={!!busy || !say.trim()}
          onClick={() => { setBusy("say"); sayBuzz(say).then(() => setSay("")).catch((e: Error) => setErr(e.message)).finally(() => setBusy(null)); }}>
          Send
        </button>
      </div>

      <div style={{ fontSize: 13, fontWeight: 650, margin: "22px 0 4px" }}>Matter</div>
      <p className="lm-guide-lead">{matter?.hint || "Home Assistant commissions the accessory. This robot is not itself a Matter device."}</p>
      <input value={code} onChange={(e) => setCode(e.target.value)} placeholder="MT:… or 1111-222-333" style={INPUT_STYLE} />
      <button type="button" className="lm-u-btn" disabled={!!busy || !code.trim() || !matter?.ready} style={{ marginTop: 8 }}
        onClick={() => {
          setBusy("matter");
          commissionMatter(code).then(() => setCode("")).catch((e: Error) => setErr(e.message)).finally(() => setBusy(null));
        }}>{busy === "matter" ? "…" : "Add accessory"}</button>

      <div style={{ fontSize: 13, fontWeight: 650, margin: "22px 0 4px" }}>Sign in with Apple</div>
      <p className="lm-guide-lead">{apple?.hint || "Apple requires HTTPS. Use Google sign-in on the LAN."}</p>
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
      {err && <p className="lm-guide-lead" style={{ color: "var(--lm-red)" }}>{err}</p>}
    </div>
  );
}
