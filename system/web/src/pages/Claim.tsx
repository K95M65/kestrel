import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { confirmClaim, getClaimPublic, type HouseholdPublic } from "@/lib/api";

export default function Claim() {
  const [params] = useSearchParams();
  const pinFromUrl = params.get("pin") || params.get("code") || "";
  const [st, setSt] = useState<HouseholdPublic | null>(null);
  const [name, setName] = useState("");
  const [room, setRoom] = useState("desk");
  const [pin, setPin] = useState(pinFromUrl);
  const [err, setErr] = useState<string | null>(null);
  const [done, setDone] = useState<HouseholdPublic | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    getClaimPublic().then((h) => {
      setSt(h);
      if (h.room) setRoom(h.room);
      if (!pinFromUrl && h.setup_pin) setPin(h.setup_pin);
    }).catch(() => {});
  }, [pinFromUrl]);

  async function submit() {
    setBusy(true);
    setErr(null);
    try {
      const next = await confirmClaim({
        pin,
        code: pin,
        name,
        room,
        role: st?.claimed ? "family" : "owner",
      });
      setDone(next);
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Could not claim.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div style={{ maxWidth: 420, margin: "48px auto", padding: 24, color: "var(--lm-text)" }}>
      <h1 style={{ fontSize: 22, margin: "0 0 8px" }}>
        {st?.claimed ? "Join this robot" : "Claim this robot"}
      </h1>
      <p style={{ fontSize: 14, color: "var(--lm-text-dim)", lineHeight: 1.5 }}>
        Same idea as adding a HomeKit accessory: a code on the robot, your name, a room.
        This stays on the local network. It does not join an Apple or Google Home.
      </p>
      {done ? (
        <p style={{ marginTop: 18, fontSize: 14 }}>
          {done.claimed ? `You're in${done.room ? ` — ${done.room}` : ""}. Open Talk on this network.` : "Done."}
        </p>
      ) : (
        <form onSubmit={(e) => { e.preventDefault(); void submit(); }} style={{ display: "flex", flexDirection: "column", gap: 10, marginTop: 18 }}>
          <label style={{ fontSize: 12 }}>
            Your name
            <input value={name} onChange={(e) => setName(e.target.value)} required
              style={{ display: "block", width: "100%", marginTop: 4, padding: 8, borderRadius: 8 }} />
          </label>
          <label style={{ fontSize: 12 }}>
            Room
            <input value={room} onChange={(e) => setRoom(e.target.value)}
              style={{ display: "block", width: "100%", marginTop: 4, padding: 8, borderRadius: 8 }} />
          </label>
          <label style={{ fontSize: 12 }}>
            Setup code
            <input value={pin} onChange={(e) => setPin(e.target.value)} required inputMode="numeric"
              style={{ display: "block", width: "100%", marginTop: 4, padding: 8, borderRadius: 8, fontFamily: "ui-monospace, monospace" }} />
          </label>
          {err && <div style={{ color: "var(--lm-red)", fontSize: 13 }}>{err}</div>}
          <button type="submit" disabled={busy || !name.trim()} className="lm-u-btn">
            {busy ? "…" : st?.claimed ? "Join" : "This is my robot"}
          </button>
        </form>
      )}
    </div>
  );
}
