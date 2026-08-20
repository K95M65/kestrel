import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { Camera, X, Loader2, Check } from "lucide-react";
import { fileToBase64, hwUrl } from "@/lib/api";
import { LiveFaceEnroll } from "@/pages/settings/guide/LiveFaceEnroll";
import { mainFacePhoto } from "@/lib/facePhoto";
import type { FaceOwnerDetail } from "../types";

/** Retake a friend's face photo, or pick one they already have. */
export function FriendPhotoModal({
  themeClass, person, robotName, onClose, onChanged,
}: {
  themeClass: string;
  person: FaceOwnerDetail;
  robotName?: string;
  onClose: () => void;
  onChanged: () => void;
}) {
  const [using, setUsing] = useState<string | null>(null);
  const [err, setErr] = useState("");
  const main = mainFacePhoto(person.photos);
  const named = person.label
    ? person.label[0].toUpperCase() + person.label.slice(1)
    : "them";

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === "Escape" && !using) onClose(); };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose, using]);

  async function useExisting(filename: string) {
    if (filename === main) {
      onClose();
      return;
    }
    setUsing(filename);
    setErr("");
    try {
      const pic = await fetch(hwUrl(`/face/photo/${encodeURIComponent(person.label)}/${encodeURIComponent(filename)}`));
      if (!pic.ok) throw new Error("Couldn't open that photo.");
      const blob = await pic.blob();
      const b64 = await fileToBase64(new File([blob], filename, { type: blob.type || "image/jpeg" }));
      const resp = await fetch(hwUrl("/face/enroll"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ label: person.label, image_base64: b64 }),
      });
      const raw = await resp.text();
      let data: { detail?: string; message?: string } = {};
      try { data = raw ? JSON.parse(raw) : {}; } catch { /* html */ }
      if (!resp.ok) throw new Error(data.detail || data.message || "Couldn't save that photo.");
      onChanged();
      onClose();
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Couldn't use that photo.");
    } finally {
      setUsing(null);
    }
  }

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
        aria-label={`Photo for ${named}`}
        className="lm-pop"
        style={{
          width: "min(520px, 100%)", maxHeight: "90vh",
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
            <Camera size={16} style={{ color: "var(--lm-amber)" }} />
            <span style={{ fontSize: 14.5, fontWeight: 600, color: "var(--lm-text)" }}>
              Photo for {named}
            </span>
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
        <div style={{ padding: 18, overflowY: "auto" }}>
          {person.photos.length > 0 && (
            <div style={{ marginBottom: 18 }}>
              <div style={{
                fontSize: 10, fontWeight: 700, letterSpacing: "0.06em",
                textTransform: "uppercase", color: "var(--lm-text-dim)", marginBottom: 8,
              }}>
                Their pictures
              </div>
              <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
                {person.photos.map((photo) => {
                  const isMain = photo === main;
                  const wait = using === photo;
                  return (
                    <button
                      key={photo}
                      type="button"
                      disabled={!!using}
                      onClick={() => void useExisting(photo)}
                      title={isMain ? "This is their photo" : "Use this photo"}
                      style={{
                        position: "relative", width: 84, height: 84, padding: 0,
                        borderRadius: 10, overflow: "hidden", cursor: wait ? "wait" : "pointer",
                        border: `2px solid ${isMain ? "var(--lm-amber)" : "var(--lm-border)"}`,
                        background: "var(--lm-surface)",
                      }}
                    >
                      <img
                        src={hwUrl(`/face/photo/${encodeURIComponent(person.label)}/${encodeURIComponent(photo)}`)}
                        alt=""
                        style={{ width: "100%", height: "100%", objectFit: "cover", display: "block" }}
                      />
                      {isMain && (
                        <span style={{
                          position: "absolute", left: 4, bottom: 4,
                          fontSize: 9, fontWeight: 700, letterSpacing: 0.3,
                          padding: "1px 5px", borderRadius: 4,
                          background: "var(--lm-amber)", color: "var(--lm-on-amber)",
                          display: "inline-flex", alignItems: "center", gap: 3,
                        }}>
                          <Check size={10} /> Now
                        </span>
                      )}
                      {wait && (
                        <span style={{
                          position: "absolute", inset: 0,
                          display: "flex", alignItems: "center", justifyContent: "center",
                          background: "rgba(0,0,0,0.45)",
                        }}>
                          <Loader2 size={18} className="lm-spin" color="#fff" />
                        </span>
                      )}
                    </button>
                  );
                })}
              </div>
              <div style={{ fontSize: 11, color: "var(--lm-text-muted)", marginTop: 8, lineHeight: 1.45 }}>
                Tap a picture to make it the one on their card.
              </div>
            </div>
          )}
          <div style={{
            fontSize: 10, fontWeight: 700, letterSpacing: "0.06em",
            textTransform: "uppercase", color: "var(--lm-text-dim)", marginBottom: 8,
          }}>
            Retake or choose a new one
          </div>
          <LiveFaceEnroll
            robotName={robotName}
            fixedLabel={person.label}
            lead=""
            saveLabel="Save this photo"
            onEnrolled={() => { onChanged(); onClose(); }}
          />
          {err && (
            <div style={{
              marginTop: 10, fontSize: 11.5, color: "var(--lm-red)", padding: "7px 10px",
              borderRadius: 7, background: "var(--lm-red-dim)", border: "1px solid var(--lm-red-glow)",
            }}>{err}</div>
          )}
        </div>
      </div>
    </div>,
    document.body,
  );
}
