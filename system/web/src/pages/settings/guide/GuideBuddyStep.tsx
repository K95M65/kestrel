import { useCallback, useEffect, useRef, useState } from "react";
import { getCompanionApps, type CompanionApp } from "@/lib/api";
import { buddyKind, buddyOSLabel } from "@/lib/buddyLabel";
import { API } from "@/pages/monitor/types";

type BuddyStatus = {
  paired: boolean;
  connected?: boolean;
  name?: string;
  osVersion?: string;
};

function normalize(d: Record<string, unknown> | null): BuddyStatus {
  if (!d) return { paired: false };
  return {
    paired: Boolean(d.paired),
    connected: Boolean(d.connected),
    name: typeof d.name === "string" ? d.name : undefined,
    osVersion: typeof d.os_version === "string" ? d.os_version : undefined,
  };
}

export function GuideBuddyStep({
  onStatus, lead,
}: {
  onStatus?: (s: BuddyStatus) => void;
  lead?: string;
}) {
  const [status, setStatus] = useState<BuddyStatus | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [code, setCode] = useState<string | null>(null);
  const [codeExpiresAt, setCodeExpiresAt] = useState<number | null>(null);
  const [now, setNow] = useState(Date.now());
  const [busy, setBusy] = useState(false);
  const [apps, setApps] = useState<CompanionApp[]>([]);
  const codeBox = useRef<HTMLDivElement | null>(null);

  const fetchStatus = useCallback(async () => {
    try {
      const r = await fetch(`${API}/buddy/status`);
      const j = await r.json();
      if (j.status === 1) {
        const next = normalize(j.data);
        setStatus(next);
        onStatus?.(next);
        setError(null);
      } else {
        setError(j.message ?? "Could not check the computer pairing.");
      }
    } catch (e) {
      setError((e as Error).message);
    }
  }, [onStatus]);

  useEffect(() => {
    void fetchStatus();
    const id = setInterval(() => void fetchStatus(), 4000);
    return () => clearInterval(id);
  }, [fetchStatus]);

  useEffect(() => {
    getCompanionApps()
      .then((list) => setApps(buddyKind(list)))
      .catch(() => {});
  }, []);

  useEffect(() => {
    if (!codeExpiresAt) return;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [codeExpiresAt]);

  useEffect(() => {
    if (codeExpiresAt && now >= codeExpiresAt) {
      setCode(null);
      setCodeExpiresAt(null);
    }
  }, [now, codeExpiresAt]);

  useEffect(() => {
    if (status?.paired) {
      setCode(null);
      setCodeExpiresAt(null);
    }
  }, [status?.paired]);

  async function pair() {
    setBusy(true);
    setError(null);
    try {
      const r = await fetch(`${API}/buddy/pair/start`, { method: "POST" });
      const j = await r.json();
      if (j.status !== 1) {
        setError(j.message ?? "Could not start pairing.");
        return;
      }
      setCode(j.data.code);
      setCodeExpiresAt(Date.now() + j.data.expiresIn * 1000);
      setNow(Date.now());
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  const ttl = codeExpiresAt ? Math.max(0, Math.ceil((codeExpiresAt - now) / 1000)) : 0;

  return (
    <>
      <p className="lm-guide-lead">
        {lead ?? "Install Kestrel Buddy on this computer, then pair with a code. On a Mac, allow Accessibility if you want it to click and type."}
      </p>
      {status?.paired && (
        <div className="lm-guide-ok" style={{ marginTop: 0, marginBottom: 10 }}>
          {status.connected
            ? `${buddyOSLabel(status.osVersion)} paired${status.name ? ` — ${status.name}` : ""} and connected.`
            : `Computer paired${status.name ? ` — ${status.name}` : ""}. Open Kestrel Buddy if it shows offline.`}
        </div>
      )}
      {status && !status.paired && !code && (
        <div className="lm-guide-choices">
          {apps.map((app) => (
            <a key={app.id} className="lm-choice" href={app.direct_url || app.download_url} target="_blank" rel="noreferrer">
              <span className="lm-choice-dot" aria-hidden />
              <span>
                <span className="lm-choice-title">{app.platform}</span>
                <span className="lm-choice-line">{app.hint || "Install, then come back for a pairing code."}</span>
              </span>
            </a>
          ))}
          <button type="button" className="lm-guide-primary" disabled={busy} onClick={() => void pair()}>
            {busy ? "Generating…" : "Pair this computer"}
          </button>
        </div>
      )}
      {code && (
        <div ref={codeBox} className="lm-guide-extra">
          <p className="lm-guide-lead" style={{ marginBottom: 8 }}>
            Enter this code in Kestrel Buddy → Pair with device.
          </p>
          <button
            type="button"
            className="lm-guide-primary"
            style={{ fontFamily: "ui-monospace, monospace", letterSpacing: "0.18em", fontSize: 22 }}
            onClick={() => void navigator.clipboard?.writeText(code).catch(() => {})}
          >
            {code}
          </button>
          <p className="lm-guide-lead" style={{ marginTop: 8 }}>Expires in {ttl}s</p>
        </div>
      )}
      {error && <div className="lm-guide-err">{error}</div>}
    </>
  );
}
