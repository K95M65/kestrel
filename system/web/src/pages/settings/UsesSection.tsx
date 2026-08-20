import { useCallback, useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { Library } from "lucide-react";
import { SectionCard } from "@/components/setup/shared";
import { getBehaviors, getDeviceConfig, getServices, listMCPTools } from "@/lib/api";
import { API } from "@/pages/monitor/types";
import { useCapabilities } from "@/hooks/useCapabilities";
import { capsFromSet } from "@/lib/guideWalk";
import {
  SCENARIOS,
  scenarioFor,
  scenarioStatus,
  scenarioStatusLabel,
  type Scenario,
} from "@/lib/scenarios";
import { ScenarioOnboarding } from "@/pages/settings/ScenarioOnboarding";

type BuddySnap = { paired: boolean };

export function UsesSection({ active }: { active: boolean }) {
  const loc = useLocation();
  const navigate = useNavigate();
  const { caps, loaded } = useCapabilities();
  const gcaps = capsFromSet(loaded ? caps : null);
  const [kids, setKids] = useState(false);
  const [telegram, setTelegram] = useState(false);
  const [buddy, setBuddy] = useState<BuddySnap>({ paired: false });
  const [toolsOn, setToolsOn] = useState(true);
  const [robotName, setRobotName] = useState("");
  const [open, setOpen] = useState<{ scenario: Scenario; start?: "try" } | null>(null);

  const reload = useCallback(async () => {
    const [beh, svcs, tools] = await Promise.all([
      getBehaviors().catch(() => null),
      getServices().catch(() => []),
      listMCPTools().catch(() => []),
    ]);
    if (beh) {
      setKids(!!beh.config.kids?.enabled);
      setToolsOn(
        !!(beh.config.tools?.weather || beh.config.tools?.search || beh.config.tools?.time)
        || tools.some((t) => t.name === "weather" || t.name === "search"),
      );
    } else {
      setToolsOn(tools.some((t) => t.name === "weather" || t.name === "search"));
    }
    setTelegram(svcs.some((s) => s.id === "telegram" && s.connected));
    try {
      const r = await fetch(`${API}/buddy/status`);
      const j = await r.json();
      setBuddy({ paired: Boolean(j.data?.paired) });
    } catch {
      /* leave */
    }
    try {
      const cfg = await getDeviceConfig();
      setRobotName(cfg.agent_name || "");
    } catch {
      /* leave */
    }
  }, []);

  useEffect(() => {
    if (!active) return;
    void reload();
  }, [active, reload]);

  useEffect(() => {
    if (!active || !loaded) return;
    const q = new URLSearchParams(loc.search);
    const id = q.get("use");
    if (!id) return;
    const s = scenarioFor(id);
    q.delete("use");
    const search = q.toString();
    navigate(
      { pathname: loc.pathname, search: search ? `?${search}` : "", hash: loc.hash },
      { replace: true },
    );
    if (s && s.need(capsFromSet(loaded ? caps : null))) setOpen({ scenario: s });
  }, [active, loaded, caps, loc.search, loc.pathname, loc.hash, navigate]);

  const ctx = useMemo(() => ({
    caps: gcaps,
    kids,
    telegram,
    buddyPaired: buddy.paired,
    toolsOn,
  }), [caps, loaded, kids, telegram, buddy.paired, toolsOn]);

  return (
    <SectionCard
      id="uses"
      title="What it can do"
      description="Jobs this body already knows. Set one up — Talk, Telegram, and the computer all use the same list."
      icon={<Library size={17} />}
      active={active}
    >
      <div className="lm-uses-grid">
        {SCENARIOS.map((s) => {
          const st = scenarioStatus(s, ctx);
          const blocked = st === "unsupported" || st === "kids";
          return (
            <article key={s.id} className={"lm-uses-card" + (blocked ? " lm-uses-card--na" : "") + (st === "ready" ? " lm-uses-card--on" : "")}>
              <div className="lm-uses-card-kicker">{s.category}</div>
              <h3 className="lm-uses-card-title">{s.title}</h3>
              <p className="lm-uses-card-line">{s.line}</p>
              <div className="lm-uses-card-foot">
                <span className={"lm-uses-status lm-uses-status--" + st}>{scenarioStatusLabel(st)}</span>
                <button
                  type="button"
                  className={blocked ? "lm-guide-ghost" : "lm-guide-primary"}
                  disabled={blocked}
                  onClick={() => setOpen({ scenario: s, start: st === "ready" ? "try" : undefined })}
                >
                  {st === "ready" ? "Try" : "Set up"}
                </button>
              </div>
            </article>
          );
        })}
      </div>
      {open && (
        <ScenarioOnboarding
          scenario={open.scenario}
          start={open.start}
          robotName={robotName}
          onClose={() => setOpen(null)}
          onDone={() => { void reload(); }}
        />
      )}
    </SectionCard>
  );
}
