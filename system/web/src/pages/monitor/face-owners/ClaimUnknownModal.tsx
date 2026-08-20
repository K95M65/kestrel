import { useEffect, type CSSProperties } from "react";
import { createPortal } from "react-dom";
import { UserPlus, X, Loader2 } from "lucide-react";

/** Name an unknown face or voice as a known person (new or existing). */
export function ClaimUnknownModal({
  themeClass, title, lead, photoUrl, household, name, setName,
  asMe, setAsMe,
  error, saving, saveLabel, onClose, onSubmit,
  inputStyle, fieldLabel, btnStyle,
}: {
  themeClass: string;
  title: string;
  lead?: string;
  photoUrl?: string | null;
  household: string[];
  name: string;
  setName: (v: string) => void;
  asMe?: boolean;
  setAsMe?: (v: boolean) => void;
  error: string;
  saving: boolean;
  saveLabel?: string;
  onClose: () => void;
  onSubmit: () => void;
  inputStyle: CSSProperties;
  fieldLabel: CSSProperties;
  btnStyle: CSSProperties;
}) {
  const others = household.filter((n) => n && n !== "unknown");
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === "Escape" && !saving) onClose(); };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose, saving]);
  return createPortal(
    <div
      className={`lm-root ${themeClass}`}
      onClick={onClose}
      style={{
        position: "fixed", inset: 0, background: "rgba(0,0,0,0.72)", backdropFilter: "blur(4px)",
        display: "flex", justifyContent: "center", alignItems: "center",
        zIndex: 1000, padding: 20,
      }}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className="lm-pop"
        style={{
          width: "min(400px, 100%)",
          background: "linear-gradient(180deg, color-mix(in srgb, var(--lm-amber) 4%, transparent), transparent 130px), var(--lm-surface)",
          border: "1px solid var(--lm-border-hi)",
          borderRadius: 14, boxShadow: "0 24px 64px -20px rgba(0,0,0,0.7), 0 2px 8px rgba(0,0,0,0.4)",
          overflow: "hidden",
        }}
      >
        <div style={{
          display: "flex", alignItems: "center", justifyContent: "space-between",
          gap: 12, padding: "16px 18px", borderBottom: "1px solid var(--lm-border)",
        }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <UserPlus size={16} style={{ color: "var(--lm-amber)" }} />
            <span style={{ fontSize: 14.5, fontWeight: 600, color: "var(--lm-text)" }}>{title}</span>
          </div>
          <button
            type="button" onClick={onClose} aria-label="Close"
            className="lm-u-btn"
            style={{
              width: 30, height: 30, borderRadius: 8, background: "var(--lm-bg)",
              border: "1px solid var(--lm-border)", color: "var(--lm-text-dim)", cursor: "pointer",
              display: "flex", alignItems: "center", justifyContent: "center",
            }}
          >
            <X size={15} />
          </button>
        </div>
        <div style={{ padding: 18, display: "flex", flexDirection: "column", gap: 14 }}>
          {photoUrl && (
            <img
              src={photoUrl}
              alt=""
              style={{
                width: "100%", maxHeight: 220, objectFit: "cover",
                borderRadius: 10, background: "var(--lm-bg)",
                border: "1px solid var(--lm-border)",
              }}
              onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = "none"; }}
            />
          )}
          {lead && (
            <div style={{ fontSize: 12.5, color: "var(--lm-text-dim)", lineHeight: 1.5 }}>{lead}</div>
          )}
          <div>
            <label htmlFor="claim-name" style={fieldLabel}>Name</label>
            <input
              id="claim-name"
              type="text"
              autoFocus
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter" && !saving && name.trim()) onSubmit(); }}
              placeholder="Alex"
              className="lm-u-input"
              style={inputStyle}
            />
            <div style={{ fontSize: 10.5, color: "var(--lm-text-muted)", marginTop: 5 }}>
              Letters, numbers, _ and - .
            </div>
          </div>
          {setAsMe && (
            <label style={{ display: "flex", alignItems: "center", gap: 8, fontSize: 13, color: "var(--lm-text)", cursor: "pointer" }}>
              <input type="checkbox" checked={!!asMe} onChange={(e) => setAsMe(e.target.checked)} />
              This is me
            </label>
          )}
          {others.length > 0 && (
            <div>
              <div style={fieldLabel}>Or someone already here</div>
              <div style={{ display: "flex", flexWrap: "wrap", gap: 6 }}>
                {others.map((n) => (
                  <button
                    key={n}
                    type="button"
                    onClick={() => setName(n)}
                    className="lm-u-btn"
                    style={{
                      ...btnStyle, padding: "5px 10px", fontSize: 12, textTransform: "capitalize",
                      background: name.trim().toLowerCase() === n ? "var(--lm-amber)" : "var(--lm-surface)",
                      color: name.trim().toLowerCase() === n ? "var(--lm-on-amber)" : "var(--lm-text)",
                      border: `1px solid ${name.trim().toLowerCase() === n ? "var(--lm-amber)" : "var(--lm-border)"}`,
                    }}
                  >
                    {n}
                  </button>
                ))}
              </div>
            </div>
          )}
          {error && (
            <div style={{
              fontSize: 11.5, color: "var(--lm-red)", padding: "7px 10px", borderRadius: 7,
              background: "var(--lm-red-dim)", border: "1px solid var(--lm-red-glow)",
            }}>{error}</div>
          )}
          <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
            <button
              type="button" onClick={onClose}
              className="lm-u-btn"
              style={{ ...btnStyle, padding: "8px 14px", fontSize: 12 }}
            >
              Cancel
            </button>
            <button
              type="button" onClick={onSubmit}
              disabled={saving || !name.trim()}
              className={"lm-u-btn" + (saving || !name.trim() ? "" : " lm-u-btn-primary")}
              style={{
                ...btnStyle, padding: "8px 14px", fontSize: 12,
                display: "inline-flex", alignItems: "center", gap: 6,
              }}
            >
              {saving
                ? <><Loader2 size={13} className="lm-spin" /> Saving…</>
                : (saveLabel ?? "Save as a friend")}
            </button>
          </div>
        </div>
      </div>
    </div>,
    document.body,
  );
}
