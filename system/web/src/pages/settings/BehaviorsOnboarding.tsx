import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { ArrowLeft, ArrowRight, Sparkles, X } from "lucide-react";
import { toast } from "sonner";
import { INPUT_STYLE, LABEL_STYLE } from "@/components/setup/shared";
import { setBehaviors, setIdentity, type BehaviorsConfig } from "@/lib/api";
import { bodyCopy } from "@/lib/bodyProfile";
import {
  applyPreset, PRESETS, PRESET_COMPARE, summaryLines, type PresetId,
} from "@/lib/behaviorsModel";
import {
  capsFromSet, extrasFor, guideSteps, ownsEnter, presetRowsFor, wideStep,
  clampBehaviors, type ExtraChip,
} from "@/lib/guideWalk";
import { useCapabilities } from "@/hooks/useCapabilities";
import { defaultWakePhrase, EXAMPLE_ROBOT_NAME } from "@/lib/robotName";
import { useTheme } from "@/lib/useTheme";
import { GuideTalkStep } from "@/pages/settings/guide/GuideTalkStep";
import { GuideSeeStep } from "@/pages/settings/guide/GuideSeeStep";
import { GuideProveStep } from "@/pages/settings/guide/GuideProveStep";
import { GuideConnectStep } from "@/pages/settings/guide/GuideConnectStep";
import { lifeHasConnect, recipeFor } from "@/lib/lifeRecipes";
import type { ServiceStatus } from "@/lib/api";

export function BehaviorsOnboarding({
  open, initial, onClose, onSaved,
}: {
  open: boolean;
  initial: BehaviorsConfig;
  onClose: () => void;
  onSaved: (cfg: BehaviorsConfig) => void;
}) {
  const [, , themeClass] = useTheme();
  const { caps, deviceType, loaded } = useCapabilities();
  const gcaps = capsFromSet(loaded ? caps : null);
  const copy = useMemo(() => bodyCopy(deviceType, gcaps, loaded), [deviceType, caps, loaded]);

  const [idx, setIdx] = useState(0);
  const [cfg, setCfg] = useState<BehaviorsConfig>(initial);
  const [preset, setPreset] = useState<PresetId | null>(null);
  const steps = useMemo(() => guideSteps(gcaps, lifeHasConnect(preset)), [caps, loaded, preset]);
  const [robotName, setRobotName] = useState("");
  const [wakePhrase, setWakePhrase] = useState("");
  const [identitySaved, setIdentitySaved] = useState(false);
  const [triedTalk, setTriedTalk] = useState(false);
  const [triedSee, setTriedSee] = useState(false);
  const [triedProve, setTriedProve] = useState(false);
  const [enrolledName, setEnrolledName] = useState("");
  const [linked, setLinked] = useState<ServiceStatus[]>([]);
  const [busy, setBusy] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);

  const step = steps[Math.min(idx, steps.length - 1)] ?? "intro";
  const lastIdx = steps.length - 1;
  const extras = extrasFor(gcaps);
  const recipe = recipeFor(preset);

  useEffect(() => {
    if (!steps.includes(step)) {
      const i = steps.indexOf("preset");
      setIdx(i >= 0 ? i : 0);
    }
  }, [steps, step]);

  useEffect(() => {
    if (!open) return;
    setIdx(0);
    setCfg(initial);
    setPreset(null);
    setRobotName("");
    setWakePhrase("");
    setIdentitySaved(false);
    setTriedTalk(false);
    setTriedSee(false);
    setTriedProve(false);
    setEnrolledName("");
    setLinked([]);
  }, [open]); // eslint-disable-line react-hooks/exhaustive-deps -- initial captured at open

  useEffect(() => {
    if (!open) return;
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    panelRef.current?.focus();
    return () => { document.body.style.overflow = prev; };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") { e.preventDefault(); onClose(); }
      if (e.key === "Enter" && !ownsEnter(step) && idx < lastIdx) {
        if (step === "name" && !robotName.trim()) return;
        e.preventDefault();
        void goNext();
      }
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onClose, step, idx, lastIdx, robotName]); // eslint-disable-line react-hooks/exhaustive-deps

  if (!open) return null;

  function pickPreset(id: PresetId) {
    setPreset(id);
    setCfg(clampBehaviors(applyPreset(cfg, id), gcaps));
  }

  function setExtra(chip: ExtraChip, on: boolean) {
    if (chip.key === "greeter") setCfg({ ...cfg, greeter: { ...cfg.greeter, enabled: on } });
    else if (chip.key === "dance") setCfg({ ...cfg, dance: { ...cfg.dance, enabled: on } });
    else if (chip.key === "remember") setCfg({ ...cfg, remember: { ...cfg.remember, enabled: on } });
    else if (chip.key === "look") setCfg({ ...cfg, look: { enabled: on } });
    else if (chip.key === "kitchen") setCfg({ ...cfg, kitchen: { ...cfg.kitchen, enabled: on } });
    else if (chip.key === "focus") setCfg({ ...cfg, focus: { ...cfg.focus, enabled: on } });
    else if (chip.key === "pomodoro") setCfg({ ...cfg, pomodoro: { ...cfg.pomodoro, enabled: on } });
    else if (chip.key === "stories") setCfg({ ...cfg, stories: { ...cfg.stories, enabled: on } });
    else if (chip.key === "kids") setCfg({ ...cfg, kids: { ...cfg.kids, enabled: on } });
  }

  function extraOn(chip: ExtraChip): boolean {
    if (chip.key === "greeter") return cfg.greeter.enabled;
    if (chip.key === "dance") return cfg.dance.enabled;
    if (chip.key === "remember") return cfg.remember.enabled;
    if (chip.key === "look") return cfg.look.enabled;
    if (chip.key === "kitchen") return cfg.kitchen.enabled;
    if (chip.key === "focus") return cfg.focus.enabled;
    if (chip.key === "pomodoro") return cfg.pomodoro.enabled;
    if (chip.key === "stories") return cfg.stories.enabled;
    if (chip.key === "kids") return cfg.kids.enabled;
    return false;
  }

  async function applyName(): Promise<boolean> {
    if (identitySaved || !robotName.trim()) return true;
    const n = robotName.trim();
    const def = defaultWakePhrase(n);
    const custom = wakePhrase.trim().toLowerCase();
    try {
      await setIdentity({ name: n, wake_phrase: custom && custom !== def ? custom : "" });
      setIdentitySaved(true);
      return true;
    } catch (err) {
      const raw = err instanceof Error ? err.message : "";
      toast.error(/JSON|Unexpected|not found|404/i.test(raw)
        ? "This device build can't keep the name yet. Talk and camera still work."
        : (raw || "Couldn't save the name on the device."));
      return true;
    }
  }

  async function goNext() {
    if (step === "name") {
      setBusy(true);
      const ok = await applyName();
      setBusy(false);
      if (!ok) return;
    }
    setIdx((i) => Math.min(i + 1, lastIdx));
  }

  async function finish(skip: boolean) {
    setBusy(true);
    try {
      if (!skip) await applyName();
      const pack = skip ? null : recipeFor(preset);
      const base = skip ? initial : cfg;
      const payload: BehaviorsConfig = clampBehaviors({
        ...base,
        onboarded: true,
        privacy: { camera_on_demand: true, face_follow_after_wake: true },
        connectors: { draft_not_send: pack?.policy.draft_not_send ?? true },
        kids: pack?.policy.kids ? { ...base.kids, enabled: true } : base.kids,
      }, gcaps);
      const saved = await setBehaviors(payload);
      onSaved(saved.config);
      toast.success(skip ? "You can run this guide any time." : "You're set.");
      onClose();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not save.");
    } finally {
      setBusy(false);
    }
  }

  const pct = ((idx + 1) / steps.length) * 100;
  const nextLocked = (step === "name" && !robotName.trim()) || busy || !loaded;
  const who = robotName.trim() || copy.kind;
  const triedThis = step === "talk" ? triedTalk : step === "see" ? triedSee : step === "prove" ? triedProve : false;

  const body = (
    <div
      className={`lm-root ${themeClass} lm-guide-overlay`}
      role="presentation"
      onMouseDown={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="beh-guide-title"
        tabIndex={-1}
        className={"lm-guide-panel" + (wideStep(step) ? " lm-guide-panel--wide" : "")}
      >
        <div className="lm-guide-progress" aria-hidden>
          <div className="lm-guide-progress-bar" style={{ width: `${pct}%` }} />
        </div>
        <div className="lm-guide-head">
          <span className="lm-guide-kicker">
            <Sparkles size={14} /> {loaded ? `Guided setup · ${idx + 1} of ${steps.length}` : "Guided setup"}
          </span>
          <button type="button" className="lm-guide-x" onClick={onClose} aria-label="Close">
            <X size={16} />
          </button>
        </div>

        <div className="lm-guide-body" key={loaded ? step : "wait"}>
          {!loaded && (
            <StepShell title="Guided setup" id="beh-guide-title">
              <p className="lm-guide-lead">Looking up what this body can do…</p>
            </StepShell>
          )}

          {loaded && step === "intro" && (
            <StepShell title={copy.introTitle} id="beh-guide-title">
              <p className="lm-guide-lead">{copy.introLead}</p>
              <p className="lm-guide-lead">{copy.sleep} {copy.expression}</p>
            </StepShell>
          )}

          {loaded && step === "name" && (
            <StepShell title={`What should we call this ${copy.kind}?`} id="beh-guide-title">
              <p className="lm-guide-lead">
                Next it will say hello using this name. You can change it later in Device → General.
              </p>
              <label htmlFor="guide-robot-name" style={LABEL_STYLE}>Name</label>
              <input
                id="guide-robot-name"
                value={robotName}
                onChange={(e) => {
                  const v = e.target.value;
                  const prev = defaultWakePhrase(robotName);
                  setRobotName(v);
                  setIdentitySaved(false);
                  if (!wakePhrase.trim() || wakePhrase.trim().toLowerCase() === prev) {
                    setWakePhrase(defaultWakePhrase(v));
                  }
                }}
                placeholder={`e.g. ${EXAMPLE_ROBOT_NAME}`}
                autoComplete="off"
                style={{ ...INPUT_STYLE, marginBottom: 12 }}
              />
              <label htmlFor="guide-wake" style={LABEL_STYLE}>Wake phrase</label>
              <input
                id="guide-wake"
                value={wakePhrase}
                onChange={(e) => { setWakePhrase(e.target.value); setIdentitySaved(false); }}
                placeholder={`e.g. ${defaultWakePhrase(EXAMPLE_ROBOT_NAME)}`}
                autoComplete="off"
                style={INPUT_STYLE}
              />
              <p className="lm-guide-lead" style={{ marginTop: 10 }}>
                Leave as “hey {robotName.trim() || "name"}” for the usual phrases, or type one custom phrase.
              </p>
            </StepShell>
          )}

          {loaded && step === "talk" && (
            <StepShell title={`Talk to ${who}`} id="beh-guide-title">
              <GuideTalkStep robotName={robotName} onTried={() => setTriedTalk(true)} lead={copy.talkLead(who)} />
            </StepShell>
          )}

          {loaded && step === "see" && (
            <StepShell title="Who is this?" id="beh-guide-title">
              <GuideSeeStep
                robotName={robotName}
                onTried={() => setTriedSee(true)}
                onEnrolled={(n) => setEnrolledName(n)}
                lead={copy.seeLead(who)}
              />
            </StepShell>
          )}

          {loaded && step === "prove" && (
            <StepShell title="Meet the body" id="beh-guide-title">
              <GuideProveStep copy={copy} caps={gcaps} onTried={() => setTriedProve(true)} />
            </StepShell>
          )}

          {loaded && step === "preset" && (
            <StepShell title="Who is this for?" id="beh-guide-title">
              <p className="lm-guide-lead">Pick a starting personality. The list on the right is what actually turns on for this body.</p>
              <div className="lm-guide-split">
                <div className="lm-guide-choices">
                  {PRESETS.map((p) => (
                    <Choice key={p.id} selected={preset === p.id} title={p.title} line={p.line}
                      onPick={() => pickPreset(p.id)} />
                  ))}
                </div>
                {preset ? (
                  <PresetPane id={preset} caps={gcaps} />
                ) : (
                  <p className="lm-guide-lead" style={{ margin: 0 }}>Choose one to see the differences.</p>
                )}
              </div>
            </StepShell>
          )}

          {loaded && step === "connect" && recipe && (
            <StepShell title="Connect what this life uses" id="beh-guide-title">
              <GuideConnectStep services={recipe.services} onStatus={setLinked} />
            </StepShell>
          )}

          {loaded && step === "mornings" && (
            <StepShell title="Mornings" id="beh-guide-title">
              <p className="lm-guide-lead">
                Do you want a morning brief — weather, calendar, overnight mail, news — spoken at a time you pick?
              </p>
              <div className="lm-guide-choices">
                <Choice selected={cfg.morning_brief.enabled} title="Yes, brief me"
                  line="Spoken once a day. Read-only — it never sends mail."
                  onPick={() => setCfg({ ...cfg, morning_brief: { ...cfg.morning_brief, enabled: true } })} />
                <Choice selected={!cfg.morning_brief.enabled} title="Stay quiet until I talk"
                  line="No scheduled speech. You can still ask “what's today?”"
                  onPick={() => setCfg({ ...cfg, morning_brief: { ...cfg.morning_brief, enabled: false } })} />
              </div>
              {cfg.morning_brief.enabled && (
                <div className="lm-guide-extra">
                  <label htmlFor="guide-brief-at" style={LABEL_STYLE}>Around</label>
                  <input id="guide-brief-at" type="time" value={cfg.morning_brief.at}
                    onChange={(e) => setCfg({ ...cfg, morning_brief: { ...cfg.morning_brief, at: e.target.value } })}
                    style={{ ...INPUT_STYLE, maxWidth: 160 }} />
                  <div className="lm-guide-chips">
                    <Chip on={cfg.morning_brief.weather} label="Weather"
                      onClick={() => setCfg({ ...cfg, morning_brief: { ...cfg.morning_brief, weather: !cfg.morning_brief.weather } })} />
                    <Chip on={cfg.morning_brief.calendar} label="Calendar"
                      onClick={() => setCfg({ ...cfg, morning_brief: { ...cfg.morning_brief, calendar: !cfg.morning_brief.calendar } })} />
                    <Chip on={cfg.morning_brief.email} label="Overnight mail"
                      onClick={() => setCfg({ ...cfg, morning_brief: { ...cfg.morning_brief, email: !cfg.morning_brief.email } })} />
                    <Chip on={cfg.morning_brief.telegram} label="Telegram copy"
                      onClick={() => setCfg({ ...cfg, morning_brief: { ...cfg.morning_brief, telegram: !cfg.morning_brief.telegram } })} />
                  </div>
                </div>
              )}
            </StepShell>
          )}

          {loaded && step === "extras" && (
            <StepShell title="When you're around" id="beh-guide-title">
              <p className="lm-guide-lead">Optional extras this body can do. Off stays quiet. Chat, websites, news, music, and stories have their own setup under House → Uses after this.</p>
              <div className="lm-guide-chips" style={{ marginTop: 4 }}>
                {extras.map((chip) => (
                  <Chip key={chip.key} on={extraOn(chip)} label={chip.label}
                    onClick={() => setExtra(chip, !extraOn(chip))} />
                ))}
              </div>
            </StepShell>
          )}

          {loaded && step === "done" && (
            <StepShell title="You're set" id="beh-guide-title">
              <p className="lm-guide-lead">Quick setup complete. Change any of this later under House, including Uses for chat, websites, news, music, and stories.</p>
              <ul className="lm-guide-summary">
                {robotName.trim() && (
                  <li>Name {robotName.trim()} · wake “{wakePhrase.trim() || defaultWakePhrase(robotName)}”</li>
                )}
                {triedTalk && <li>Talked to it from this guide</li>}
                {triedProve && <li>Met the body</li>}
                {enrolledName
                  ? <li>First person: {enrolledName}</li>
                  : triedSee && <li>Tried the camera</li>}
                {linked.filter((s) => s.connected).map((s) => (
                  <li key={s.id}>{s.id === "gmail" ? "Gmail" : s.id === "google_calendar" ? "Calendar" : "Telegram"} connected</li>
                ))}
                {summaryFor(cfg, gcaps).map((line) => <li key={line}>{line}</li>)}
                {summaryFor(cfg, gcaps).length === 0 && !robotName.trim() && <li>Quiet defaults — talk to it when you want.</li>}
              </ul>
            </StepShell>
          )}
        </div>

        <div className="lm-guide-foot">
          {step === "intro" ? (
            <>
              <button type="button" className="lm-guide-ghost" disabled={busy} onClick={() => void finish(true)}>
                Skip for now
              </button>
              <button type="button" className="lm-guide-primary" disabled={!loaded} onClick={() => setIdx(1)}>
                Start <ArrowRight size={14} />
              </button>
            </>
          ) : (
            <>
              <button type="button" className="lm-guide-ghost" onClick={() => setIdx((i) => Math.max(0, i - 1))}>
                <ArrowLeft size={14} /> Back
              </button>
              {idx < lastIdx ? (
                <button
                  type="button"
                  className="lm-guide-primary"
                  disabled={nextLocked}
                  onClick={() => void goNext()}
                >
                  {step === "name" && busy ? "Saving…" : ownsEnter(step) && triedThis ? "Continue" : "Next"} <ArrowRight size={14} />
                </button>
              ) : (
                <button type="button" className="lm-guide-primary" disabled={busy} onClick={() => void finish(false)}>
                  {busy ? "Saving…" : "Save this setup"}
                </button>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );

  return createPortal(body, document.body);
}

function summaryFor(cfg: BehaviorsConfig, caps: ReturnType<typeof capsFromSet>): string[] {
  return summaryLines(cfg).filter((line) => {
    if (line.startsWith("Dances") && !caps.motion) return false;
    if (line.startsWith("Greets") && !(caps.presence || caps.vision)) return false;
    if (line.includes("Camera") && !caps.vision) return false;
    return true;
  });
}

function PresetPane({ id, caps }: { id: PresetId; caps: ReturnType<typeof capsFromSet> }) {
  const title = PRESETS.find((p) => p.id === id)?.title ?? id;
  const rows = presetRowsFor(caps, PRESET_COMPARE);
  return (
    <div className="lm-guide-pane">
      <div className="lm-guide-pane-title">{title}</div>
      <table className="lm-guide-compare">
        <thead>
          <tr>
            <th> </th>
            {PRESETS.map((p) => (
              <th key={p.id} className={p.id === id ? "lm-guide-compare--on" : undefined}>{p.title}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.label}>
              <td>{row.label}</td>
              {PRESETS.map((p) => (
                <td key={p.id} className={p.id === id ? "lm-guide-compare--on" : undefined}>
                  {row.values[p.id]}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function StepShell({ title, id, children }: { title: string; id: string; children: ReactNode }) {
  return (
    <>
      <h2 id={id} className="lm-guide-title">{title}</h2>
      {children}
    </>
  );
}

function Choice({ selected, title, line, onPick }: {
  selected: boolean; title: string; line: string; onPick: () => void;
}) {
  return (
    <button type="button" className={"lm-choice" + (selected ? " lm-choice--on" : "")} onClick={onPick}
      aria-pressed={selected}>
      <span className="lm-choice-dot" aria-hidden />
      <span>
        <span className="lm-choice-title">{title}</span>
        <span className="lm-choice-line">{line}</span>
      </span>
    </button>
  );
}

function Chip({ on, label, onClick }: { on: boolean; label: string; onClick: () => void }) {
  return (
    <button type="button" className={"lm-chip" + (on ? " lm-chip--on" : "")} onClick={onClick}
      aria-pressed={on}>{label}</button>
  );
}
