import { useEffect, useState, type CSSProperties } from "react";
import { Laptop } from "lucide-react";
import { SectionCard, C } from "./shared";
import { getCompanionApps, type CompanionApp } from "@/lib/api";

export function AppsSection({ active }: { active: boolean }) {
  const [apps, setApps] = useState<CompanionApp[]>([]);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    getCompanionApps().then(setApps).catch((e: Error) => setErr(e.message));
  }, []);

  return (
    <SectionCard
      id="apps"
      title="Companion apps"
      active={active}
      icon={<Laptop size={17} />}
      description="Optional. Install these on your computer so the robot can use it. You can skip and pair later from the dash."
    >
      {err && <div style={{ fontSize: 12, color: C.red, marginBottom: 10 }}>{err}</div>}
      {apps.length === 0 && !err && (
        <div style={{ fontSize: 12, color: C.textMuted }}>Loading…</div>
      )}
      {apps.map((app) => (
        <div
          key={app.id}
          style={{
            border: `1px solid ${C.border}`,
            borderRadius: 10,
            padding: "14px 16px",
            marginBottom: 12,
            background: C.surface,
          }}
        >
          <div style={{ display: "flex", justifyContent: "space-between", gap: 8, marginBottom: 6 }}>
            <strong style={{ fontSize: 14 }}>{app.name}</strong>
            <span style={{ fontSize: 11, color: C.textMuted }}>{app.platform}{app.version ? ` · ${app.version}` : ""}</span>
          </div>
          <p style={{ margin: "0 0 8px", fontSize: 13, color: C.textDim, lineHeight: 1.45 }}>{app.summary}</p>
          {app.hint && <p style={{ margin: "0 0 12px", fontSize: 12, color: C.textMuted, lineHeight: 1.45 }}>{app.hint}</p>}
          <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
            <a href={app.direct_url || app.download_url} target="_blank" rel="noreferrer"
              style={btnStyle(true)}>
              Download
            </a>
            <a href={app.download_url} target="_blank" rel="noreferrer" style={btnStyle(false)}>
              Releases
            </a>
            <a href={app.source_url} target="_blank" rel="noreferrer" style={btnStyle(false)}>
              Source
            </a>
          </div>
        </div>
      ))}
    </SectionCard>
  );
}

function btnStyle(primary: boolean): CSSProperties {
  return {
    display: "inline-block",
    padding: "7px 12px",
    borderRadius: 7,
    fontSize: 12,
    fontWeight: 600,
    textDecoration: "none",
    border: primary ? "none" : `1px solid ${C.border}`,
    background: primary ? C.amber : "transparent",
    color: primary ? C.bg : C.text,
  };
}
