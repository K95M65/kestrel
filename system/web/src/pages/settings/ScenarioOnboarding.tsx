import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { ArrowLeft, ArrowRight, Sparkles, X } from "lucide-react";
import { toast } from "sonner";
import {
  addMCPTool, getBehaviors, listMCPTools,
  setBehaviors, type BehaviorsConfig,
} from "@/lib/api";
import { mergeBehaviors } from "@/lib/behaviorsModel";
import { useTheme } from "@/lib/useTheme";
import { GuideConnectStep } from "@/pages/settings/guide/GuideConnectStep";
import { GuideTalkStep } from "@/pages/settings/guide/GuideTalkStep";
import { GuideBuddyStep } from "@/pages/settings/guide/GuideBuddyStep";
import {
  scenarioSteps, type Scenario, type ScenarioStepId,
} from "@/lib/scenarios";

async function mergeBehaviorsSafe(): Promise<BehaviorsConfig> {
  try {
    const s = await getBehaviors();
    return mergeBehaviors(s.config);
  } catch {
    return mergeBehaviors(null);
  }
}

function withScenarioFlags(cfg: BehaviorsConfig, scenario: Scenario): BehaviorsConfig {
  let next: BehaviorsConfig = { ...cfg };
  if (scenario.tools.weather || scenario.tools.time || scenario.tools.search) {
    next = {
      ...next,
      tools: {
        weather: scenario.tools.weather ?? cfg.tools.weather,
        time: scenario.tools.time ?? cfg.tools.time,
        search: scenario.tools.search ?? cfg.tools.search,
      },
    };
  }
  if (scenario.briefWeather) {
    next = { ...next, morning_brief: { ...cfg.morning_brief, ...next.morning_brief, weather: true } };
  }
  if (scenario.flip?.stories) next = { ...next, stories: { ...cfg.stories, enabled: true } };
  if (scenario.flip?.look) next = { ...next, look: { enabled: true } };
  if (scenario.flip?.dance) next = { ...next, dance: { ...cfg.dance, enabled: true } };
  return next;
}

export function ScenarioOnboarding({
  scenario, robotName, onClose, onDone, start,
}: {
  scenario: Scenario;
  robotName: string;
  onClose: () => void;
  onDone: () => void;
  start?: ScenarioStepId;
}) {
  const [, , themeClass] = useTheme();
  const steps = useMemo(() => scenarioSteps(scenario), [scenario]);
  const [idx, setIdx] = useState(() => {
    const i = start ? steps.indexOf(start) : 0;
    return i >= 0 ? i : 0;
  });
  const [busy, setBusy] = useState(false);
  const [tried, setTried] = useState(false);
  const [buddyPaired, setBuddyPaired] = useState(false);
  const [toolsNote, setToolsNote] = useState<string | null>(null);
  const [toolsErr, setToolsErr] = useState<string | null>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const step: ScenarioStepId = steps[Math.min(idx, steps.length - 1)] ?? "intro";
  const lastIdx = steps.length - 1;

  useEffect(() => {
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    panelRef.current?.focus();
    return () => { document.body.style.overflow = prev; };
  }, []);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") { e.preventDefault(); onClose(); }
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  async function applyTools() {
    setBusy(true);
    setToolsErr(null);
    try {
      await setBehaviors(withScenarioFlags(await mergeBehaviorsSafe(), scenario));
      const have = await listMCPTools().catch(() => []);
      const names = new Set(have.map((t) => t.name));
      const added: string[] = [];
      const skipped: string[] = [];
      for (const tool of scenario.mcp) {
        if (names.has(tool.name)) { skipped.push(tool.title); continue; }
        try {
          await addMCPTool({ name: tool.name, url: tool.url });
          added.push(tool.title);
          names.add(tool.name);
        } catch (e) {
          const msg = e instanceof Error ? e.message : "";
          if (/already exists/i.test(msg)) skipped.push(tool.title);
          else skipped.push(`${tool.title} (not reached)`);
        }
      }
      const bits = [];
      if (added.length) bits.push(`Added ${added.join(", ")}.`);
      if (skipped.length) bits.push(`${skipped.join(", ")} already on or skipped.`);
      setToolsNote(bits.join(" ") || "Weather, time, and search are on.");
    } catch (e) {
      setToolsErr(e instanceof Error ? e.message : "Could not turn those on.");
    } finally {
      setBusy(false);
    }
  }

  async function finish() {
    setBusy(true);
    try {
      const needsFlags = !!(
        scenario.tools.weather || scenario.tools.search || scenario.tools.time
        || scenario.briefWeather || scenario.flip
      );
      if (needsFlags) {
        await setBehaviors(withScenarioFlags(await mergeBehaviorsSafe(), scenario));
      }
      toast.success(`${scenario.title} is on.`);
      onDone();
      onClose();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Could not save.");
    } finally {
      setBusy(false);
    }
  }

  const nextLocked = busy || (step === "buddy" && scenario.buddy === "required" && !buddyPaired);
  const pct = ((idx + 1) / steps.length) * 100;

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
        aria-labelledby="use-guide-title"
        tabIndex={-1}
        className={"lm-guide-panel" + (step === "connect" || step === "buddy" || step === "try" ? " lm-guide-panel--wide" : "")}
      >
        <div className="lm-guide-progress" aria-hidden>
          <div className="lm-guide-progress-bar" style={{ width: `${pct}%` }} />
        </div>
        <div className="lm-guide-head">
          <span className="lm-guide-kicker">
            <Sparkles size={14} /> {scenario.title} · {idx + 1} of {steps.length}
          </span>
          <button type="button" className="lm-guide-x" onClick={onClose} aria-label="Close">
            <X size={16} />
          </button>
        </div>
        <div className="lm-guide-body" key={step}>
          {step === "intro" && (
            <StepShell title={scenario.title} id="use-guide-title">
              <p className="lm-guide-lead">{scenario.why}</p>
              <p className="lm-guide-lead">{scenario.honest}</p>
            </StepShell>
          )}
          {step === "connect" && (
            <StepShell title="Connect" id="use-guide-title">
              <GuideConnectStep
                services={scenario.services}
                lead="Skip any you do not want. You can finish this later under Device → Channels."
              />
            </StepShell>
          )}
          {step === "buddy" && (
            <StepShell title="Pair a computer" id="use-guide-title">
              <GuideBuddyStep onStatus={(s) => setBuddyPaired(s.paired)} />
            </StepShell>
          )}
          {step === "tools" && (
            <StepShell title="Turn on live info" id="use-guide-title">
              <p className="lm-guide-lead">
                Weather, time, and search for this use. Weather still works from a public forecast if a live tool does not load. After you turn them on, wait a few seconds before the try — the brain restarts.
              </p>
              <button type="button" className="lm-guide-primary" disabled={busy} onClick={() => void applyTools()}>
                {busy ? "Turning on…" : toolsNote ? "Turned on" : "Turn on weather and search"}
              </button>
              {toolsNote && <div className="lm-guide-ok">{toolsNote}</div>}
              {toolsErr && <div className="lm-guide-err">{toolsErr}</div>}
            </StepShell>
          )}
          {step === "try" && (
            <StepShell title="Try it" id="use-guide-title">
              <GuideTalkStep
                robotName={robotName}
                lead={scenario.tryHint}
                prompt={scenario.tryPrompt}
                greet={false}
                timeoutMs={90_000}
                onTried={() => setTried(true)}
              />
            </StepShell>
          )}
          {step === "done" && (
            <StepShell title="This use is on" id="use-guide-title">
              <p className="lm-guide-lead">Change it later under House → Uses, or the matching toggle in Behaviors.</p>
              <ul className="lm-guide-summary">
                <li>{scenario.title}</li>
                {tried && <li>Tried a line from this guide</li>}
                {buddyPaired && scenario.buddy === "required" && <li>Mac paired</li>}
                <li>{scenario.honest}</li>
              </ul>
            </StepShell>
          )}
        </div>
        <div className="lm-guide-foot">
          <button type="button" className="lm-guide-ghost" onClick={() => idx === 0 ? onClose() : setIdx((i) => Math.max(0, i - 1))}>
            <ArrowLeft size={14} /> {idx === 0 ? "Close" : "Back"}
          </button>
          {idx < lastIdx ? (
            <button
              type="button"
              className="lm-guide-primary"
              disabled={nextLocked}
              onClick={() => setIdx((i) => Math.min(i + 1, lastIdx))}
            >
              Next <ArrowRight size={14} />
            </button>
          ) : (
            <button type="button" className="lm-guide-primary" disabled={busy} onClick={() => void finish()}>
              {busy ? "Saving…" : "Done"}
            </button>
          )}
        </div>
      </div>
    </div>
  );

  return createPortal(body, document.body);
}

function StepShell({ title, id, children }: { title: string; id: string; children: ReactNode }) {
  return (
    <>
      <h2 id={id} className="lm-guide-title">{title}</h2>
      {children}
    </>
  );
}
