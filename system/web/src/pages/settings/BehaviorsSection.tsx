import { useEffect, useMemo, useState, type CSSProperties, type ReactNode } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { ChevronDown, Sparkles } from "lucide-react";
import { toast } from "sonner";
import { C, SectionCard, LABEL_STYLE, INPUT_STYLE } from "@/components/setup/shared";
import {
  addMemory, defaultBehaviors, deleteMemory, fireBriefNow, getBehaviors,
  listMemories, setBehaviors, setMeeting, startPomodoro, stopPomodoro,
  type BehaviorsConfig, type BehaviorsStatus, type MemoryItem,
} from "@/lib/api";
import {
  applyPreset, countOn, DAYS, FEATURE_GROUPS, formatNext, isFeatureOn, mergeBehaviors,
  num, PRESETS, type FeatureKey, type PresetId,
} from "@/lib/behaviorsModel";
import { BehaviorsOnboarding } from "@/pages/settings/BehaviorsOnboarding";
import { useCapabilities } from "@/hooks/useCapabilities";
import { ASK_LEVELS, draftFromAsk, normalizeAsk, type AskLevel } from "@/lib/askLevels";
import { bodyCopy } from "@/lib/bodyProfile";
import { capsFromSet, clampBehaviors, featureSupported, UNSUPPORTED } from "@/lib/guideWalk";

const FEATURE_META: Record<FeatureKey, { title: string; hint: string; badge?: string }> = {
  morning_brief: { title: "Morning briefing", hint: "Weather, calendar, overnight mail — spoken, read-only." },
  remember: { title: "Remember this", hint: "Voice or chat inbox. Not a dump of pose logs." },
  kitchen: { title: "Kitchen / meals", hint: "Dinner ideas and the lunch/dinner windows wellbeing uses." },
  pomodoro: { title: "Pomodoro", hint: "OS-owned work/break timer. The robot announces the flip." },
  wearables: { title: "Wearables in the briefing", hint: "Oura / Whoop / Garmin, only if a connector exists." },
  dance: { title: "Dance to any song", hint: "Named track → music + motion together." },
  presence: { title: "Idle presence", hint: "Stay alive when nobody is talking. Breathing is HAL's job." },
  doa: { title: "Turn toward voice", hint: "Look at the person speaking.", badge: "needs HAL" },
  layered_motion: { title: "Layered motion", hint: "Speech wobble + emotion + face track.", badge: "needs HAL" },
  hand_track: { title: "Hand tracking / mime", hint: "Follow a hand. Visitor demo.", badge: "needs HAL" },
  marionette: { title: "Marionette", hint: "Grab the head, record, replay.", badge: "needs HAL" },
  radio: { title: "Radio", hint: "A stream while it moves.", badge: "partial" },
  greeter: { title: "Office greeter", hint: "Hello when someone walks in." },
  look: { title: "Look at this", hint: "One JPEG on demand — not a live stream." },
  privacy: { title: "Camera snapshots", hint: "“Look at this” does not leave the stream running." },
  connectors: { title: "Ask before sending", hint: "Mail and calendar. Default is important actions — drafts only." },
  kids: { title: "Kids profile", hint: "No mail, calendar, HA, or computer-use. Gentle stories." },
  stories: { title: "Stories", hint: "Bedtime / tell-me-a-story, time-capped." },
  focus: { title: "Phone / focus coach", hint: "Nag when a phone is in frame." },
  home_assistant: { title: "Home Assistant", hint: "Lights and climate. Token is write-only." },
  tools: { title: "Weather / time / search", hint: "Permission flags — add MCP servers under MCP Tools." },
  telepresence: { title: "Telepresence", hint: "Points at the camera page. No public tunnel.", badge: "needs HAL" },
};

export function BehaviorsSection({ active }: { active: boolean }) {
  const loc = useLocation();
  const navigate = useNavigate();
  const { caps, deviceType, loaded } = useCapabilities();
  const gcaps = capsFromSet(caps);
  const copy = bodyCopy(deviceType, gcaps, loaded);
  const [cfg, setCfg] = useState<BehaviorsConfig>(defaultBehaviors());
  const [status, setStatus] = useState<BehaviorsStatus | null>(null);
  const [haToken, setHaToken] = useState("");
  const [haSet, setHaSet] = useState(false);
  const [memories, setMemories] = useState<MemoryItem[]>([]);
  const [note, setNote] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<string | null>(null);
  const [openGroup, setOpenGroup] = useState<string>("daily");
  const [openFeat, setOpenFeat] = useState<FeatureKey | null>("morning_brief");
  const [guide, setGuide] = useState(false);

  function applyStatus(s: BehaviorsStatus) {
    setStatus(s);
    setCfg(mergeBehaviors(s.config));
    setHaSet(!!s.ha_token_set);
  }

  async function reload() {
    const [s, mem] = await Promise.all([getBehaviors(), listMemories().catch(() => [] as MemoryItem[])]);
    applyStatus(s);
    setMemories(mem || []);
    return s;
  }

  useEffect(() => {
    reload().catch(() => {}).finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    const q = new URLSearchParams(loc.search);
    if (q.get("guide") !== "1") return;
    setGuide(true);
    q.delete("guide");
    const search = q.toString();
    navigate(
      { pathname: loc.pathname, search: search ? `?${search}` : "", hash: loc.hash },
      { replace: true },
    );
  }, [loc.hash, loc.pathname, loc.search, navigate]);

  const counts = useMemo(
    () => countOn(cfg, (k) => featureSupported(k, gcaps)),
    [cfg, caps],
  );

  async function save(next = cfg, msg = "Behaviors saved.") {
    setBusy("save");
    try {
      const payload: BehaviorsConfig = clampBehaviors(
        { ...next, home_assistant: { ...next.home_assistant } },
        gcaps,
      );
      if (haToken.trim()) payload.home_assistant.token = haToken.trim();
      else delete payload.home_assistant.token;
      applyStatus(await setBehaviors(payload));
      setHaToken("");
      toast.success(msg);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not save behaviors.");
    } finally {
      setBusy(null);
    }
  }

  async function act(kind: string, fn: () => Promise<BehaviorsStatus>) {
    setBusy(kind);
    try {
      applyStatus(await fn());
      toast.success("Done.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "That did not work.");
    } finally {
      setBusy(null);
    }
  }

  async function saveNote() {
    const text = note.trim();
    if (!text) return;
    setBusy("note");
    try {
      const item = await addMemory(text);
      setMemories((prev) => [...prev, item]);
      setNote("");
      toast.success("Saved.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not save note.");
    } finally {
      setBusy(null);
    }
  }

  async function dropNote(id: string) {
    setBusy("del-" + id);
    try {
      await deleteMemory(id);
      setMemories((prev) => prev.filter((m) => m.id !== id));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not delete.");
    } finally {
      setBusy(null);
    }
  }

  function setOn(key: FeatureKey, on: boolean) {
    switch (key) {
      case "morning_brief": setCfg({ ...cfg, morning_brief: { ...cfg.morning_brief, enabled: on } }); break;
      case "remember": setCfg({ ...cfg, remember: { ...cfg.remember, enabled: on } }); break;
      case "kitchen": setCfg({ ...cfg, kitchen: { ...cfg.kitchen, enabled: on } }); break;
      case "pomodoro": setCfg({ ...cfg, pomodoro: { ...cfg.pomodoro, enabled: on } }); break;
      case "wearables": setCfg({ ...cfg, wearables: { ...cfg.wearables, enabled: on } }); break;
      case "dance": setCfg({ ...cfg, dance: { ...cfg.dance, enabled: on } }); break;
      case "presence": setCfg({ ...cfg, presence: { idle_motion: on } }); break;
      case "doa": setCfg({ ...cfg, doa: { enabled: on } }); break;
      case "layered_motion": setCfg({ ...cfg, layered_motion: { enabled: on } }); break;
      case "hand_track": setCfg({ ...cfg, hand_track: { enabled: on } }); break;
      case "marionette": setCfg({ ...cfg, marionette: { enabled: on } }); break;
      case "radio": setCfg({ ...cfg, radio: { enabled: on } }); break;
      case "greeter": setCfg({ ...cfg, greeter: { ...cfg.greeter, enabled: on } }); break;
      case "look": setCfg({ ...cfg, look: { enabled: on } }); break;
      case "privacy": setCfg({ ...cfg, privacy: { ...cfg.privacy, camera_on_demand: on } }); break;
      case "connectors": setCfg({ ...cfg, connectors: { draft_not_send: on, ask: on ? "important_actions" : "never_ask" } }); break;
      case "kids": setCfg({ ...cfg, kids: { ...cfg.kids, enabled: on } }); break;
      case "stories": setCfg({ ...cfg, stories: { ...cfg.stories, enabled: on } }); break;
      case "focus": setCfg({ ...cfg, focus: { ...cfg.focus, enabled: on } }); break;
      case "home_assistant": setCfg({ ...cfg, home_assistant: { ...cfg.home_assistant, enabled: on } }); break;
      case "tools": setCfg({ ...cfg, tools: { weather: on, time: on, search: on } }); break;
      case "telepresence": setCfg({ ...cfg, telepresence: { enabled: on } }); break;
    }
  }

  const days = cfg.morning_brief.days?.length ? cfg.morning_brief.days : [0, 1, 2, 3, 4, 5, 6];

  function knobs(key: FeatureKey): ReactNode {
    if (key === "morning_brief") {
      return (
        <>
          <div style={{ fontSize: 11, color: C.textMuted, marginBottom: 8 }}>
            {status?.last_brief ? `Last ran ${status.last_brief}. ` : ""}
            {status?.next_brief ? `Next ${formatNext(status.next_brief)}.` : ""}
          </div>
          <Row>
            <Field label="Time" type="time" value={cfg.morning_brief.at}
              onChange={(v) => setCfg({ ...cfg, morning_brief: { ...cfg.morning_brief, at: v } })} />
            <Field label="Spoken seconds" type="number" value={String(cfg.morning_brief.max_seconds)}
              onChange={(v) => setCfg({ ...cfg, morning_brief: { ...cfg.morning_brief, max_seconds: num(v, 40) } })} />
          </Row>
          <DayRow days={days} onToggle={(n) => {
            const has = days.includes(n);
            const next = has ? days.filter((d) => d !== n) : [...days, n].sort((a, b) => a - b);
            setCfg({ ...cfg, morning_brief: { ...cfg.morning_brief, days: next.length === 7 ? [] : next } });
          }} />
          <Checks items={[
            ["speak", "Speak it", cfg.morning_brief.speak],
            ["telegram", "Telegram copy", cfg.morning_brief.telegram],
            ["weather", "Weather", cfg.morning_brief.weather],
            ["calendar", "Calendar", cfg.morning_brief.calendar],
            ["email", "Overnight mail", cfg.morning_brief.email],
            ["habits", "Habit beat", cfg.morning_brief.habits],
          ]} onChange={(k, v) => setCfg({ ...cfg, morning_brief: { ...cfg.morning_brief, [k]: v } })} />
          <button type="button" disabled={!!busy} onClick={() => void act("brief", fireBriefNow)} style={btn(false, busy === "brief")}>
            {busy === "brief" ? "…" : "Brief now"}
          </button>
        </>
      );
    }
    if (key === "connectors") {
      const ask = normalizeAsk(cfg.connectors.ask, cfg.connectors.draft_not_send);
      return (
        <>
          {ASK_LEVELS.map((lv) => (
            <label key={lv.id} style={{ display: "flex", gap: 8, alignItems: "flex-start", margin: "8px 0", fontSize: 13, cursor: "pointer" }}>
              <input type="radio" name="ask" checked={ask === lv.id}
                onChange={() => setCfg({
                  ...cfg,
                  connectors: { ask: lv.id, draft_not_send: draftFromAsk(lv.id as AskLevel) },
                })} />
              <span>
                <strong>{lv.title}</strong>
                <span style={{ display: "block", fontSize: 12, color: C.textMuted }}>{lv.hint}</span>
              </span>
            </label>
          ))}
        </>
      );
    }
    if (key === "remember") {
      return (
        <>
          <Field label="Max notes" type="number" value={String(cfg.remember.max_items)}
            onChange={(v) => setCfg({ ...cfg, remember: { ...cfg.remember, max_items: num(v, 200) } })} />
          <div style={{ display: "flex", gap: 8, marginTop: 8 }}>
            <input value={note} onChange={(e) => setNote(e.target.value)} placeholder="Add a note"
              style={{ ...INPUT_STYLE, flex: 1 }} />
            <button type="button" disabled={!!busy} onClick={() => void saveNote()} style={btn(true, busy === "note")}>Add</button>
          </div>
          <div style={{ marginTop: 8, maxHeight: 140, overflow: "auto" }}>
            {memories.length === 0 && <div style={{ fontSize: 11, color: C.textMuted }}>No notes yet.</div>}
            {memories.slice().reverse().map((m) => (
              <div key={m.id} style={{
                display: "flex", justifyContent: "space-between", gap: 8,
                fontSize: 12, padding: "6px 0", borderBottom: `1px solid ${C.border}`,
              }}>
                <span>{m.text}</span>
                <button type="button" disabled={!!busy} onClick={() => void dropNote(m.id)}
                  style={{ ...btn(false, busy === "del-" + m.id), padding: "2px 8px" }}>×</button>
              </div>
            ))}
          </div>
        </>
      );
    }
    if (key === "kitchen") {
      return (
        <>
          <Row>
            <Field label="Lunch from" type="time" value={cfg.kitchen.lunch_start}
              onChange={(v) => setCfg({ ...cfg, kitchen: { ...cfg.kitchen, lunch_start: v } })} />
            <Field label="Lunch to" type="time" value={cfg.kitchen.lunch_end}
              onChange={(v) => setCfg({ ...cfg, kitchen: { ...cfg.kitchen, lunch_end: v } })} />
          </Row>
          <Row>
            <Field label="Dinner from" type="time" value={cfg.kitchen.dinner_start}
              onChange={(v) => setCfg({ ...cfg, kitchen: { ...cfg.kitchen, dinner_start: v } })} />
            <Field label="Dinner to" type="time" value={cfg.kitchen.dinner_end}
              onChange={(v) => setCfg({ ...cfg, kitchen: { ...cfg.kitchen, dinner_end: v } })} />
          </Row>
        </>
      );
    }
    if (key === "pomodoro") {
      return (
        <>
          <Row>
            <Field label="Work min" type="number" value={String(cfg.pomodoro.work_min)}
              onChange={(v) => setCfg({ ...cfg, pomodoro: { ...cfg.pomodoro, work_min: num(v, 25) } })} />
            <Field label="Break min" type="number" value={String(cfg.pomodoro.break_min)}
              onChange={(v) => setCfg({ ...cfg, pomodoro: { ...cfg.pomodoro, break_min: num(v, 5) } })} />
          </Row>
          <div style={{ fontSize: 11, color: C.textMuted, margin: "6px 0" }}>
            {status?.pomodoro?.running ? `${status.pomodoro.phase} · ${status.pomodoro.remain_sec}s left` : "Idle"}
          </div>
          <div style={{ display: "flex", gap: 8 }}>
            <button type="button" disabled={!!busy} onClick={() => void act("pomo", startPomodoro)} style={btn(true, busy === "pomo")}>Start</button>
            <button type="button" disabled={!!busy} onClick={() => void act("pomo-stop", stopPomodoro)} style={btn(false, busy === "pomo-stop")}>Stop</button>
          </div>
        </>
      );
    }
    if (key === "wearables") {
      return (
        <>
          <label style={LABEL_STYLE}>Provider</label>
          <select value={cfg.wearables.provider || "none"}
            onChange={(e) => setCfg({ ...cfg, wearables: { ...cfg.wearables, provider: e.target.value } })}
            style={INPUT_STYLE}>
            <option value="none">None</option>
            <option value="oura">Oura</option>
            <option value="whoop">Whoop</option>
            <option value="garmin">Garmin</option>
          </select>
        </>
      );
    }
    if (key === "dance") {
      return (
        <Field label="Default when they just say dance" value={cfg.dance.default_query}
          onChange={(v) => setCfg({ ...cfg, dance: { ...cfg.dance, default_query: v } })} />
      );
    }
    if (key === "greeter") {
      return (
        <Check label="Named friends only" checked={cfg.greeter.named_only}
          onChange={(v) => setCfg({ ...cfg, greeter: { ...cfg.greeter, named_only: v } })} />
      );
    }
    if (key === "privacy") {
      return (
        <>
          <div style={{ fontSize: 12, marginBottom: 8, color: status?.meeting ? C.amber : C.textMuted }}>
            {status?.meeting ? "In a meeting — mic, speaker, camera off" : "Not in a meeting"}
          </div>
          <div style={{ display: "flex", gap: 8, marginBottom: 10 }}>
            <button type="button" disabled={!!busy} onClick={() => void act("meet-on", () => setMeeting(true))}
              style={btn(true, busy === "meet-on")}>Meeting now</button>
            <button type="button" disabled={!!busy} onClick={() => void act("meet-off", () => setMeeting(false))}
              style={btn(false, busy === "meet-off")}>End meeting</button>
          </div>
          <Check label="Face-follow only after wake / named friend" checked={cfg.privacy.face_follow_after_wake}
            onChange={(v) => setCfg({ ...cfg, privacy: { ...cfg.privacy, face_follow_after_wake: v } })} />
        </>
      );
    }
    if (key === "kids") {
      return (
        <Field label="Session minutes" type="number" value={String(cfg.kids.session_min)}
          onChange={(v) => setCfg({ ...cfg, kids: { ...cfg.kids, session_min: num(v, 30) } })} />
      );
    }
    if (key === "stories") {
      return (
        <Field label="Max minutes" type="number" value={String(cfg.stories.max_min)}
          onChange={(v) => setCfg({ ...cfg, stories: { ...cfg.stories, max_min: num(v, 10) } })} />
      );
    }
    if (key === "focus") {
      return (
        <>
          <Check label="Nag when a phone is in frame" checked={cfg.focus.phone_nag}
            onChange={(v) => setCfg({ ...cfg, focus: { ...cfg.focus, phone_nag: v } })} />
          <Field label="Cooldown minutes" type="number" value={String(cfg.focus.cooldown_min)}
            onChange={(v) => setCfg({ ...cfg, focus: { ...cfg.focus, cooldown_min: num(v, 15) } })} />
        </>
      );
    }
    if (key === "home_assistant") {
      return (
        <>
          <Field label="URL" value={cfg.home_assistant.url} placeholder="http://homeassistant.local:8123"
            onChange={(v) => setCfg({ ...cfg, home_assistant: { ...cfg.home_assistant, url: v } })} />
          <Field label={haSet ? "Token (saved — leave blank to keep)" : "Long-lived token"} type="password"
            value={haToken} onChange={setHaToken} />
        </>
      );
    }
    if (key === "tools") {
      return (
        <Checks items={[
          ["weather", "Weather", cfg.tools.weather],
          ["time", "Time", cfg.tools.time],
          ["search", "Search", cfg.tools.search],
        ]} onChange={(k, v) => setCfg({ ...cfg, tools: { ...cfg.tools, [k]: v } })} />
      );
    }
    return null;
  }

  return (
    <SectionCard
      id="behaviors"
      title="How it lives here"
      description="Guided setup for first run. Chat, websites, news, music, and stories live under House → Uses."
      icon={<Sparkles size={17} />}
      active={active}
    >
      {loading ? (
        <div style={{ fontSize: 12, color: C.textMuted }}>Loading…</div>
      ) : (
        <>
          {!cfg.onboarded && (
            <div className="lm-beh-welcome">
              <div>
                <div className="lm-beh-welcome-title">Set up how this {copy.kind} lives here</div>
                <div className="lm-beh-welcome-line">{copy.welcomeLine}</div>
              </div>
              <button type="button" className="lm-guide-primary" onClick={() => setGuide(true)}>
                Start guided setup
              </button>
            </div>
          )}

          <div className="lm-beh-hero">
            <div>
              <div className="lm-beh-count">{counts.on}<span> / {counts.total} on</span></div>
              <div className="lm-beh-hero-line">Quiet hours still win at bedtime. Off means the skill stays quiet.</div>
            </div>
            <div className="lm-beh-hero-actions">
              <button type="button" className="lm-guide-ghost" onClick={() => setGuide(true)}>
                Guided setup
              </button>
              <button type="button" disabled={!!busy} onClick={() => void save()}
                className="lm-guide-primary" style={{ opacity: busy === "save" ? 0.7 : 1 }}>
                {busy === "save" ? "Saving…" : "Save"}
              </button>
            </div>
          </div>

          <div className="lm-beh-presets" role="group" aria-label="Starting personality">
            {PRESETS.map((p) => (
              <button key={p.id} type="button" className="lm-preset"
                title={p.line}
                onClick={() => { setCfg(clampBehaviors(applyPreset(cfg, p.id as PresetId), gcaps)); toast.message(`${p.title} applied — save to keep it.`); }}>
                {p.title}
              </button>
            ))}
          </div>

          <div className="lm-beh-status">
            <span className={"lm-pill" + (status?.meeting ? " lm-pill--hot" : "")}>
              {status?.meeting ? "Meeting" : "Awake"}
            </span>
            {cfg.morning_brief.enabled && (
              <span className="lm-pill">Brief {cfg.morning_brief.at}{status?.next_brief ? ` · ${formatNext(status.next_brief)}` : ""}</span>
            )}
            {status?.pomodoro?.running && (
              <span className="lm-pill">{status.pomodoro.phase} · {status.pomodoro.remain_sec}s</span>
            )}
          </div>

          {FEATURE_GROUPS.map((g) => {
            const onN = g.keys.filter((k) => featureSupported(k, gcaps) && isFeatureOn(cfg, k)).length;
            const open = openGroup === g.id;
            return (
              <div key={g.id} className="lm-acc">
                <button type="button" className="lm-acc-head" aria-expanded={open}
                  onClick={() => setOpenGroup(open ? "" : g.id)}>
                  <span>{g.title}</span>
                  <span className="lm-acc-meta">{onN} on <ChevronDown size={14} className={open ? "lm-chev-open" : ""} /></span>
                </button>
                {open && (
                  <div className="lm-acc-body">
                    {g.keys.map((key) => {
                      const meta = FEATURE_META[key];
                      const on = isFeatureOn(cfg, key);
                      const expanded = openFeat === key;
                      const extra = knobs(key);
                      const ok = featureSupported(key, gcaps);
                      return (
                        <div key={key} className={"lm-feat" + (on ? " lm-feat--on" : "") + (ok ? "" : " lm-feat--na")}>
                          <div className="lm-feat-row">
                            <label className="lm-switch">
                              <input type="checkbox" checked={ok && on} disabled={!ok}
                                onChange={(e) => setOn(key, e.target.checked)} />
                              <span />
                            </label>
                            <button type="button" className="lm-feat-main"
                              onClick={() => setOpenFeat(expanded ? null : key)}>
                              <span className="lm-feat-title">
                                {meta.title}
                                {meta.badge && <span className="lm-feat-badge">{meta.badge}</span>}
                              </span>
                              <span className="lm-feat-hint">{ok ? meta.hint : UNSUPPORTED}</span>
                            </button>
                            {extra && ok && (
                              <ChevronDown size={14} className={expanded ? "lm-chev-open" : ""} style={{ color: C.textMuted, flexShrink: 0 }} />
                            )}
                          </div>
                          {expanded && extra && ok && <div className="lm-feat-knobs">{extra}</div>}
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          })}
        </>
      )}

      <BehaviorsOnboarding
        open={guide && active && !loading}
        initial={cfg}
        onClose={() => setGuide(false)}
        onSaved={(next) => {
          setCfg(mergeBehaviors(next));
          void reload();
        }}
      />
    </SectionCard>
  );
}

function Check({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", marginBottom: 6 }}>
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
      <span style={{ fontSize: 12.5, color: C.text }}>{label}</span>
    </label>
  );
}

function Checks({ items, onChange }: { items: [string, string, boolean][]; onChange: (key: string, v: boolean) => void }) {
  return (
    <div style={{ display: "flex", flexWrap: "wrap", gap: "4px 16px", marginBottom: 10 }}>
      {items.map(([k, label, on]) => (
        <Check key={k} label={label} checked={on} onChange={(v) => onChange(k, v)} />
      ))}
    </div>
  );
}

function Field({ label, value, onChange, type = "text", placeholder }: {
  label: string; value: string; onChange: (v: string) => void; type?: string; placeholder?: string;
}) {
  const id = "beh-" + label.replace(/\s+/g, "-").toLowerCase();
  return (
    <div style={{ flex: 1, minWidth: 0 }}>
      <label htmlFor={id} style={LABEL_STYLE}>{label}</label>
      <input id={id} type={type} value={value} placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)} style={INPUT_STYLE} />
    </div>
  );
}

function Row({ children }: { children: ReactNode }) {
  return <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12, marginBottom: 10 }}>{children}</div>;
}

function DayRow({ days, onToggle }: { days: number[]; onToggle: (n: number) => void }) {
  return (
    <div style={{ marginBottom: 10 }}>
      <div style={LABEL_STYLE}>Days</div>
      <div style={{ display: "flex", flexWrap: "wrap", gap: 6 }}>
        {DAYS.map((d) => {
          const on = days.includes(d.n);
          return (
            <button key={d.n} type="button" onClick={() => onToggle(d.n)} style={{
              padding: "5px 9px", borderRadius: 7, fontSize: 11, fontWeight: 600,
              border: `1px solid ${on ? C.amber : C.border}`,
              background: on ? C.amber : "transparent",
              color: on ? "var(--lm-on-amber)" : C.text, cursor: "pointer",
            }}>{d.label}</button>
          );
        })}
      </div>
    </div>
  );
}

function btn(primary: boolean, wait: boolean): CSSProperties {
  return {
    padding: "6px 11px", borderRadius: 7, fontSize: 12, fontWeight: 600,
    border: primary ? "none" : `1px solid ${C.border}`,
    background: primary ? C.amber : "transparent",
    color: primary ? "var(--lm-on-amber)" : C.text,
    cursor: wait ? "wait" : "pointer", opacity: wait ? 0.7 : 1,
  };
}
