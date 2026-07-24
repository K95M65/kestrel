import { C } from "@/components/setup/shared";
import { useTheme } from "@/lib/useTheme";

// Placeholder shown while the Setup page still can't know which step to open.
//
// Two callers, one shape:
//  - SetupGate (App.tsx), while checkInternet()/getSetupStatus() decide whether
//    this is `initial` or `continue` mode.
//  - Setup itself, while the URL hash names a step (#voice / #face) that isn't
//    in visibleSections yet because the mode hasn't resolved.
//
// Both are the same question — "which tab does the operator get?" — so both
// render this instead of guessing Wi-Fi and correcting a beat later. It is
// deliberately NOT time-based: it appears only while the answer is genuinely
// unknown and disappears the instant it is, so a fast network barely sees it
// and a slow one never flashes the wrong tab.
//
// Mirrors the real chrome (sidebar + topbar + card) at the same dimensions so
// resolving swaps content in without the layout jumping.
export function SetupSkeleton() {
  // Read the persisted theme so the skeleton lands in the operator's mode
  // rather than flashing dark-on-light before the real page mounts.
  const [, , themeClass] = useTheme();
  const bar = (w: number | string, h: number, mb = 0) => (
    <div style={{ width: w, height: h, borderRadius: 6, background: C.surface, marginBottom: mb }} />
  );

  return (
    <div
      className={`lm-root lm-setup ${themeClass} lm-fade-in`}
      aria-busy="true"
      aria-label="Loading setup"
      style={{
        display: "flex", height: "100vh",
        background: C.bg, color: C.text,
        fontFamily: "'Inter', 'Segoe UI', sans-serif", fontSize: 14,
      }}
    >
      {/* Sidebar — same 192px as Setup.tsx so nothing shifts on resolve. */}
      <aside
        className="lm-sidebar"
        style={{
          width: 192, flexShrink: 0,
          background: C.sidebar, borderRight: `1px solid ${C.border}`,
          display: "flex", flexDirection: "column",
        }}
      >
        <div style={{ padding: "16px 16px 12px" }}>
          {bar(96, 12, 8)}
          {bar(56, 8, 10)}
          <div className="lm-progress-track" />
        </div>
        <nav style={{ padding: "4px 12px 10px", flex: 1, display: "flex", flexDirection: "column", gap: 10 }}>
          {/* Three rows: the common continue-mode shape (Wi-Fi / Voice / Face).
              Count is cosmetic — the real nav replaces this wholesale. */}
          {[0, 1, 2].map((i) => (
            <div key={i} style={{ display: "flex", alignItems: "center", gap: 8, padding: "6px 4px" }}>
              {bar(15, 15)}
              {bar(i === 0 ? 52 : 72, 9)}
            </div>
          ))}
        </nav>
      </aside>

      <main style={{ flex: 1, minWidth: 0, display: "flex", flexDirection: "column", overflow: "hidden" }}>
        <div style={{ borderBottom: `1px solid ${C.border}`, flexShrink: 0 }}>
          <div style={{
            padding: "12px 24px 10px",
            display: "flex", alignItems: "center", justifyContent: "space-between",
          }}>
            {bar(88, 13)}
            {bar(52, 9)}
          </div>
          <div className="lm-progress-track" style={{ borderRadius: 0 }} />
        </div>

        <div style={{ flex: 1, minHeight: 0, overflowY: "auto", padding: "24px 32px" }}>
          <div style={{ maxWidth: 560, margin: "0 auto" }}>
            <div className="lm-card" style={{ padding: "20px 22px", marginBottom: 16 }}>
              <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 18 }}>
                {bar(34, 34)}
                <div style={{ flex: 1 }}>
                  {bar(132, 11, 8)}
                  {bar("70%", 9)}
                </div>
              </div>
              {bar(64, 9, 10)}
              {bar("100%", 38, 16)}
              {bar(84, 9, 10)}
              {bar("100%", 38)}
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
