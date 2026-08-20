import { createPortal } from "react-dom";
import { UserPlus, X } from "lucide-react";
import { LiveFaceEnroll } from "@/pages/settings/guide/LiveFaceEnroll";

export function EnrollModal({
  themeClass, onClose, onEnrolled, robotName,
}: {
  themeClass: string;
  onClose: () => void;
  onEnrolled: (name: string) => void;
  robotName?: string;
}) {
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
        aria-label="Add a friend"
        className="lm-pop"
        style={{
          width: "min(480px, 100%)", maxHeight: "90vh",
          display: "flex", flexDirection: "column",
          background: "linear-gradient(180deg, color-mix(in srgb, var(--lm-amber) 4%, transparent), transparent 130px), var(--lm-card)",
          border: "1px solid var(--lm-border-hi)",
          borderRadius: 14, boxShadow: "0 24px 64px -20px rgba(0,0,0,0.7), 0 2px 8px rgba(0,0,0,0.4)",
          overflow: "hidden",
        }}
      >
        <div style={{
          flexShrink: 0, display: "flex", alignItems: "center", justifyContent: "space-between",
          gap: 12, padding: "16px 18px", borderBottom: "1px solid var(--lm-border)",
        }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <UserPlus size={16} style={{ color: "var(--lm-amber)" }} />
            <span style={{ fontSize: 14.5, fontWeight: 600, color: "var(--lm-text)" }}>Add a friend</span>
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
        <div style={{ padding: "18px", overflowY: "auto" }}>
          <LiveFaceEnroll
            robotName={robotName}
            onEnrolled={onEnrolled}
            lead="Look at the camera. We'll take a photo, then you name them."
            saveLabel="Add this person"
          />
        </div>
      </div>
    </div>,
    document.body,
  );
}
