import { useCallback, useEffect, useState } from "react";
import { INPUT_STYLE, LABEL_STYLE } from "@/components/setup/shared";
import { commissionMatter, getMatter, type MatterStatus } from "@/lib/api";

/** House → Home Assistant: add a bulb the way Apple Home / Google Home scan a code. */
export function MatterCard() {
  const [matter, setMatter] = useState<MatterStatus | null>(null);
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const reload = useCallback(async () => {
    try { setMatter(await getMatter()); } catch { /* */ }
  }, []);
  useEffect(() => { void reload(); }, [reload]);

  return (
    <div style={{ marginTop: 12 }}>
      <div style={{ fontSize: 13, fontWeight: 650, marginBottom: 4 }}>Add a Matter accessory</div>
      <p className="lm-guide-lead">
        Apple Home and Google Home scan a QR on the box. Here the robot asks Home Assistant to do that.
        This robot is not itself a HomeKit or Google Home device.
      </p>
      <ol className="lm-guide-lead" style={{ margin: "8px 0 10px", paddingLeft: 18 }}>
        <li>Turn Home Assistant on and save the URL + token above.</li>
        <li>Power the bulb or lock. Find the Matter code on the box (starts with MT: or looks like 1111-222-333).</li>
        <li>Paste it here. The robot commissions it into the house.</li>
      </ol>
      <p className="lm-guide-lead">{matter?.hint}</p>
      <label htmlFor="matter-code" style={LABEL_STYLE}>Pairing code</label>
      <input id="matter-code" value={code} onChange={(e) => setCode(e.target.value)}
        placeholder="MT:… or 1111-222-333" style={INPUT_STYLE} />
      <button type="button" className="lm-u-btn" disabled={busy || !code.trim() || !matter?.ready} style={{ marginTop: 8 }}
        onClick={() => {
          setBusy(true);
          setErr(null);
          commissionMatter(code)
            .then(() => { setCode(""); void reload(); })
            .catch((e: Error) => setErr(e.message))
            .finally(() => setBusy(false));
        }}>{busy ? "…" : "Add accessory"}</button>
      {err && <p className="lm-guide-lead" style={{ color: "var(--lm-red)" }}>{err}</p>}
    </div>
  );
}
