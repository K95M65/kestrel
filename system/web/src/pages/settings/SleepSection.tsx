import { useEffect, useState, type CSSProperties } from "react";
import { Moon } from "lucide-react";
import { toast } from "sonner";
import { C, SectionCard, LABEL_STYLE, INPUT_STYLE } from "@/components/setup/shared";
import { getSleep, setSleepSchedule, sleepNow, wakeNow, type SleepStatus } from "@/lib/api";
import { isRobotQuiet, sleepToggleKind, sleepToggleLabel, withSleeping } from "@/lib/sleepToggle";
import { bodyCopy } from "@/lib/bodyProfile";
import { capsFromSet } from "@/lib/guideWalk";
import { useCapabilities } from "@/hooks/useCapabilities";

const DAYS = [
  { n: 0, label: "Sun" },
  { n: 1, label: "Mon" },
  { n: 2, label: "Tue" },
  { n: 3, label: "Wed" },
  { n: 4, label: "Thu" },
  { n: 5, label: "Fri" },
  { n: 6, label: "Sat" },
];

function formatNext(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString(undefined, { weekday: "short", hour: "2-digit", minute: "2-digit" });
}

export function SleepSection({ active }: { active: boolean }) {
  const { caps, deviceType, loaded } = useCapabilities();
  const copy = bodyCopy(deviceType, capsFromSet(caps), loaded);
  const [status, setStatus] = useState<SleepStatus | null>(null);
  const [enabled, setEnabled] = useState(false);
  const [sleepAt, setSleepAt] = useState("23:00");
  const [wakeAt, setWakeAt] = useState("07:00");
  const [days, setDays] = useState<number[]>([0, 1, 2, 3, 4, 5, 6]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<string | null>(null);

  function applyStatus(s: SleepStatus) {
    setStatus(s);
    setEnabled(!!s.schedule?.enabled);
    setSleepAt(s.schedule?.sleep_at || "23:00");
    setWakeAt(s.schedule?.wake_at || "07:00");
    setDays(s.schedule?.days?.length ? s.schedule.days : [0, 1, 2, 3, 4, 5, 6]);
  }

  useEffect(() => {
    if (!active) return;
    getSleep()
      .then(applyStatus)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [active]);

  async function save() {
    setBusy("save");
    try {
      applyStatus(await setSleepSchedule({
        enabled,
        sleep_at: sleepAt,
        wake_at: wakeAt,
        days: days.length === 7 ? [] : days,
      }));
      toast.success(enabled ? "Quiet hours saved." : "Quiet hours off.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not save quiet hours.");
    } finally {
      setBusy(null);
    }
  }

  async function now(kind: "sleep" | "wake") {
    setBusy(kind);
    setStatus((s) => withSleeping(s, kind === "sleep"));
    try {
      applyStatus(kind === "sleep" ? await sleepNow() : await wakeNow());
      toast.success(kind === "sleep" ? "The robot is quiet." : "The robot is awake.");
    } catch (err) {
      getSleep().then(applyStatus).catch(() => {});
      toast.error(err instanceof Error ? err.message : "Could not change sleep.");
    } finally {
      setBusy(null);
    }
  }

  function toggleDay(n: number) {
    setDays((prev) => {
      const has = prev.includes(n);
      const next = has ? prev.filter((d) => d !== n) : [...prev, n].sort((a, b) => a - b);
      return next.length ? next : prev;
    });
  }

  const quiet = isRobotQuiet(status?.sleeping, status?.emotion);
  const nextLabel = status?.next_transition
    ? `${status.next_transition_kind === "wake" ? "Wakes" : "Sleeps"} ${formatNext(status.next_transition)}`
    : "";

  return (
    <SectionCard
      id="sleep"
      title="Quiet hours"
      icon={<Moon size={17} />}
      active={active}
    >
      {loading ? (
        <div style={{ fontSize: 12, color: C.textMuted }}>Loading…</div>
      ) : (
        <>
          <div style={{ fontSize: 12.5, color: C.textDim, marginBottom: 14, lineHeight: 1.6 }}>
            At the sleep time this {copy.kind} goes still and silent — no motion, no speaker,
            no mic. Walking past will not wake it. Fire alerts still get through.
            {" "}{copy.sleep} Times use the device timezone.
          </div>

          <div style={{
            display: "flex", alignItems: "center", justifyContent: "space-between",
            gap: 10, marginBottom: 14, padding: "10px 12px",
            border: `1px solid ${C.border}`, borderRadius: 8, background: C.surface,
          }}>
            <div>
              <div style={{ fontSize: 13, fontWeight: 600, color: C.text }}>
                {quiet ? "Quiet now" : "Awake"}
              </div>
              {nextLabel && (
                <div style={{ fontSize: 11, color: C.textMuted, marginTop: 3 }}>{nextLabel}</div>
              )}
            </div>
            <button type="button" disabled={!!busy} onClick={() => void now(sleepToggleKind(quiet))}
              style={btn(true, !!busy)}>
              {busy === "sleep" || busy === "wake" ? "…" : sleepToggleLabel(quiet)}
            </button>
          </div>

          <label style={{ ...LABEL_STYLE, display: "flex", alignItems: "center", gap: 8, cursor: "pointer" }}>
            <input
              type="checkbox"
              checked={enabled}
              onChange={(e) => setEnabled(e.target.checked)}
            />
            Use a nightly schedule
          </label>

          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12, margin: "14px 0" }}>
            <div>
              <label htmlFor="sleep-at" style={LABEL_STYLE}>Sleep</label>
              <input id="sleep-at" type="time" value={sleepAt} onChange={(e) => setSleepAt(e.target.value)}
                style={INPUT_STYLE} />
            </div>
            <div>
              <label htmlFor="wake-at" style={LABEL_STYLE}>Wake</label>
              <input id="wake-at" type="time" value={wakeAt} onChange={(e) => setWakeAt(e.target.value)}
                style={INPUT_STYLE} />
            </div>
          </div>

          <div style={{ marginBottom: 14 }}>
            <div style={LABEL_STYLE}>Days</div>
            <div style={{ display: "flex", flexWrap: "wrap", gap: 6 }}>
              {DAYS.map((d) => {
                const on = days.includes(d.n);
                return (
                  <button
                    key={d.n}
                    type="button"
                    onClick={() => toggleDay(d.n)}
                    style={{
                      padding: "5px 9px", borderRadius: 7, fontSize: 11, fontWeight: 600,
                      border: `1px solid ${on ? C.amber : C.border}`,
                      background: on ? C.amber : "transparent",
                      color: on ? "var(--lm-on-amber)" : C.text,
                      cursor: "pointer",
                    }}
                  >
                    {d.label}
                  </button>
                );
              })}
            </div>
          </div>

          <button type="button" disabled={!!busy} onClick={() => void save()}
            style={{ ...btn(true, busy === "save"), padding: "8px 16px" }}>
            {busy === "save" ? "Saving…" : "Save schedule"}
          </button>
        </>
      )}
    </SectionCard>
  );
}

function btn(primary: boolean, wait: boolean): CSSProperties {
  return {
    padding: "6px 11px",
    borderRadius: 7,
    fontSize: 12,
    fontWeight: 600,
    border: primary ? "none" : `1px solid ${C.border}`,
    background: primary ? C.amber : "transparent",
    color: primary ? "var(--lm-on-amber)" : C.text,
    cursor: wait ? "wait" : "pointer",
    opacity: wait ? 0.7 : 1,
  };
}
