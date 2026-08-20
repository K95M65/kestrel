import { useCallback, useEffect, useState } from "react";
import { INPUT_STYLE, LABEL_STYLE } from "@/components/setup/shared";
import {
  getServices, setCalendarICS, setGmailPAT, setTelegramChannel,
  type ServiceStatus,
} from "@/lib/api";
import type { RecipeService } from "@/lib/lifeRecipes";

export function GuideConnectStep({
  services, onStatus, lead,
}: {
  services: RecipeService[];
  onStatus?: (rows: ServiceStatus[]) => void;
  lead?: string;
}) {
  const [rows, setRows] = useState<ServiceStatus[]>([]);
  const [open, setOpen] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);

  const reload = useCallback(async () => {
    try {
      const next = await getServices();
      setRows(next);
      onStatus?.(next);
    } catch {
      /* leave previous */
    }
  }, [onStatus]);

  useEffect(() => { void reload(); }, [reload]);

  function rowFor(id: string): ServiceStatus | undefined {
    return rows.find((r) => r.id === id);
  }

  return (
    <>
      <p className="lm-guide-lead">
        {lead ?? "These accounts make the brief and remote chat work. Skip any you do not want. Mail stays a draft — it never sends from this device."}
      </p>
      <div className="lm-guide-choices">
        {services.map((svc) => {
          const st = rowFor(svc.id);
          const on = !!st?.connected;
          return (
            <div key={svc.id} className={"lm-choice" + (on ? " lm-choice--on" : "")}>
              <span className="lm-choice-dot" aria-hidden />
              <span style={{ flex: 1, minWidth: 0 }}>
                <button type="button" className="lm-feat-main" style={{ width: "100%", textAlign: "left" }}
                  onClick={() => setOpen(open === svc.id ? null : svc.id)}>
                  <span className="lm-choice-title">{svc.title}{on ? " — connected" : ""}</span>
                  <span className="lm-choice-line">{on && (st?.user_email || st?.label) ? (st.user_email || st.label) : svc.why}</span>
                </button>
                {open === svc.id && (
                  <ServiceFields
                    svc={svc}
                    connected={on}
                    busy={busy === svc.id}
                    error={err && open === svc.id ? err : null}
                    onSave={async (payload) => {
                      setBusy(svc.id);
                      setErr(null);
                      try {
                        let next: ServiceStatus[] = [];
                        if (svc.id === "gmail") next = await setGmailPAT(payload.email || "", payload.secret || "");
                        else if (svc.id === "google_calendar") next = await setCalendarICS(payload.secret || "");
                        else if (svc.id === "telegram") next = await setTelegramChannel(payload.secret || "", payload.userId || "");
                        setRows(next);
                        onStatus?.(next);
                        setOpen(null);
                      } catch (e) {
                        setErr(e instanceof Error ? e.message : "Could not save.");
                      } finally {
                        setBusy(null);
                      }
                    }}
                  />
                )}
              </span>
            </div>
          );
        })}
      </div>
    </>
  );
}

function ServiceFields({
  svc, connected, busy, error, onSave,
}: {
  svc: RecipeService;
  connected: boolean;
  busy: boolean;
  error: string | null;
  onSave: (p: { email?: string; secret?: string; userId?: string }) => Promise<void>;
}) {
  const [email, setEmail] = useState("");
  const [secret, setSecret] = useState("");
  const [userId, setUserId] = useState("");

  if (connected) {
    return <p className="lm-guide-lead" style={{ margin: "10px 0 0" }}>Already linked. You can change it later under Device → Channels.</p>;
  }

  return (
    <div className="lm-guide-extra">
      {svc.id === "google_calendar" && (
        <>
          <p className="lm-guide-lead" style={{ margin: "0 0 8px" }}>
            Google Calendar → Settings and sharing → Integrate calendar → Secret address in iCal format.
          </p>
          <label htmlFor="guide-cal-url" style={LABEL_STYLE}>Secret iCal address</label>
          <input id="guide-cal-url" value={secret} onChange={(e) => setSecret(e.target.value)}
            placeholder="https://calendar.google.com/calendar/ical/…" autoComplete="off" style={INPUT_STYLE} />
        </>
      )}
      {svc.id === "gmail" && (
        <>
          <label htmlFor="guide-gmail-email" style={LABEL_STYLE}>Gmail address</label>
          <input id="guide-gmail-email" value={email} onChange={(e) => setEmail(e.target.value)}
            placeholder="you@gmail.com" autoComplete="off" style={{ ...INPUT_STYLE, marginBottom: 8 }} />
          <label htmlFor="guide-gmail-key" style={LABEL_STYLE}>App password</label>
          <input id="guide-gmail-key" type="password" value={secret} onChange={(e) => setSecret(e.target.value)}
            placeholder="Google Account → Security → App passwords" autoComplete="off" style={INPUT_STYLE} />
        </>
      )}
      {svc.id === "telegram" && (
        <>
          <label htmlFor="guide-tg-token" style={LABEL_STYLE}>Bot token</label>
          <input id="guide-tg-token" value={secret} onChange={(e) => setSecret(e.target.value)}
            placeholder="from @BotFather" autoComplete="off" style={{ ...INPUT_STYLE, marginBottom: 8 }} />
          <label htmlFor="guide-tg-user" style={LABEL_STYLE}>Your user ID</label>
          <input id="guide-tg-user" value={userId} onChange={(e) => setUserId(e.target.value)}
            placeholder="from @userinfobot" autoComplete="off" style={INPUT_STYLE} />
        </>
      )}
      {error && <div className="lm-guide-err">{error}</div>}
      <button
        type="button"
        className="lm-guide-primary"
        style={{ marginTop: 10 }}
        disabled={busy || (svc.id === "gmail" ? !email.trim() || !secret.trim() : svc.id === "google_calendar" ? !secret.trim() : !secret.trim() || !userId.trim())}
        onClick={() => void onSave({ email, secret, userId })}
      >
        {busy ? "Saving…" : "Connect"}
      </button>
    </div>
  );
}
