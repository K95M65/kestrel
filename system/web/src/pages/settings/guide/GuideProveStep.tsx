import { useState } from "react";
import { hwUrl } from "@/lib/api";
import type { BodyCopy } from "@/lib/bodyProfile";
import { proveActs, type GuideCaps } from "@/lib/guideWalk";
import { wakeIfQuiet } from "@/lib/preparePhoto";

type Act = { id: string; label: string; run: () => Promise<void> };

function wait(ms: number) {
  return new Promise<void>((r) => setTimeout(r, ms));
}

async function postJson(path: string, body: unknown): Promise<void> {
  const r = await fetch(hwUrl(path), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!r.ok) throw new Error(path);
}

export function GuideProveStep({
  copy, caps, onTried,
}: {
  copy: BodyCopy;
  caps: GuideCaps;
  onTried: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [status, setStatus] = useState<string | null>(null);
  const [log, setLog] = useState<{ id: string; ok: boolean; detail: string }[]>([]);
  const [error, setError] = useState<string | null>(null);

  const acts: Act[] = proveActs(caps).map((id) => {
    if (id === "light") {
      return {
        id,
        label: copy.proveLight || "Glow",
        run: async () => {
          await postJson("/led/solid", { color: [255, 168, 64], transient: true });
          await wait(500);
        },
      };
    }
    if (id === "motion") {
      return {
        id,
        label: copy.proveMotion || "Look at you",
        run: async () => {
          await postJson("/servo/aim", { direction: "user", duration: 1.5 });
          await wait(1600);
        },
      };
    }
    return {
      id: "emotion",
      label: copy.proveEmotion || "A small hello",
      run: async () => {
        await postJson("/emotion", { emotion: "greeting", intensity: 0.6 });
        await wait(700);
      },
    };
  });

  async function runAll() {
    if (busy || acts.length === 0) return;
    setBusy(true);
    setError(null);
    setStatus(null);
    try {
      const woke = await wakeIfQuiet((line) => setStatus(line));
      if (woke) await wait(400);
    } catch {
      /* prove still tries */
    }
    setStatus(null);
    const next: { id: string; ok: boolean; detail: string }[] = [];
    for (const act of acts) {
      try {
        await act.run();
        next.push({ id: act.id, ok: true, detail: act.label });
      } catch {
        next.push({ id: act.id, ok: false, detail: act.label });
      }
      setLog([...next]);
    }
    if (next.some((x) => x.ok)) onTried();
    if (next.every((x) => !x.ok)) {
      setError("Couldn't move or light up. You can continue and try this later from Home.");
    }
    setBusy(false);
  }

  return (
    <>
      <p className="lm-guide-lead">{copy.proveLead}</p>
      <ul className="lm-guide-prove">
        {acts.map((a) => {
          const row = log.find((x) => x.id === a.id);
          return (
            <li key={a.id} className={"lm-guide-prove-row" + (row ? (row.ok ? " lm-guide-prove-row--ok" : " lm-guide-prove-row--bad") : "")}>
              {a.label}
              {row ? (row.ok ? " — done" : " — skipped") : ""}
            </li>
          );
        })}
      </ul>
      {status && <div className="lm-guide-ok">{status}</div>}
      {error && <div className="lm-guide-err">{error}</div>}
      {log.some((x) => x.ok) && !busy && (
        <div className="lm-guide-ok">That's this body saying hello.</div>
      )}
      <button
        type="button"
        className="lm-guide-primary"
        style={{ marginTop: 12 }}
        disabled={busy || acts.length === 0}
        onClick={() => void runAll()}
      >
        {busy ? "Hello…" : log.length ? "Try again" : "Say hello"}
      </button>
    </>
  );
}
