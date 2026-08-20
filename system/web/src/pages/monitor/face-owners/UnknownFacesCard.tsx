import type { CSSProperties } from "react";
import { ScanFace, UserX, UserPlus, Trash2 } from "lucide-react";
import { hwUrl } from "@/lib/api";
import { CardLabel } from "../components";
import { EmptyState } from "./EmptyState";
import { fmtIsoAgo } from "./format";
import { FAMILIAR_VISIT_THRESHOLD } from "./types";
import type { FaceStrangerStat } from "./types";

export function UnknownFacesCard({
  faceStrangers, faceStrangersError, claimingId, forgettingId,
  onClaim, onForget, monCard, cardHeader,
}: {
  faceStrangers: FaceStrangerStat[] | null;
  faceStrangersError: boolean;
  claimingId?: string | null;
  forgettingId?: string | null;
  onClaim: (strangerId: string) => void;
  onForget: (strangerId: string) => void;
  monCard: CSSProperties;
  cardHeader: CSSProperties;
}) {
  return (
    <div className="lm-mon-card" style={monCard}>
      <div style={cardHeader}>
        <CardLabel icon={<ScanFace size={13} />} text="Unknown faces" />
        <span style={{ fontSize: 10, color: "var(--lm-text-muted)" }}>
          {faceStrangers ? `${faceStrangers.length} ${faceStrangers.length === 1 ? "face" : "faces"}` : ""}
        </span>
      </div>

      {faceStrangersError && (
        <EmptyState icon={<UserX size={18} />} text="Can't load unknown faces right now." />
      )}

      {!faceStrangersError && faceStrangers && faceStrangers.length === 0 && (
        <EmptyState icon={<ScanFace size={18} />} text="Nobody unrecognized yet." />
      )}

      {!faceStrangersError && faceStrangers && faceStrangers.length > 0 && (
        <div style={{ display: "flex", flexDirection: "column", gap: 8, maxHeight: 420, overflowY: "auto" }} className="lm-hide-scroll lm-scroll-fade">
          {faceStrangers.map((s) => {
            const familiar = s.count >= FAMILIAR_VISIT_THRESHOLD;
            const busy = claimingId === s.stranger_id || forgettingId === s.stranger_id;
            return (
              <div key={s.stranger_id} className="lm-u-interactive" style={{
                padding: "10px 12px",
                borderRadius: 8,
                background: "var(--lm-surface)",
              }}>
                <div style={{ display: "flex", gap: 12, alignItems: "flex-start" }}>
                  <img
                    src={hwUrl(`/face/stranger-photo/${encodeURIComponent(s.stranger_id)}`)}
                    alt=""
                    style={{
                      width: 56, height: 56, borderRadius: 8, objectFit: "cover", flexShrink: 0,
                      background: "var(--lm-bg)", border: "1px solid var(--lm-border)",
                    }}
                    onError={(e) => { (e.currentTarget as HTMLImageElement).style.visibility = "hidden"; }}
                  />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
                      <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
                        <span style={{ fontSize: 13, fontWeight: 700, color: "var(--lm-text)" }}>
                          Unrecognized
                        </span>
                        <span style={{
                          fontSize: 10, padding: "1px 6px", borderRadius: 4, fontWeight: 600,
                          background: familiar ? "var(--lm-amber-dim)" : "var(--lm-surface)",
                          color: familiar ? "var(--lm-amber)" : "var(--lm-text-muted)",
                          border: "1px solid var(--lm-border)",
                        }}>
                          {s.count} visit{s.count !== 1 ? "s" : ""}
                        </span>
                      </div>
                      <span style={{ fontSize: 10, color: "var(--lm-text-muted)" }}>
                        last {s.last_seen ? fmtIsoAgo(s.last_seen) : "?"}
                      </span>
                    </div>
                    <div style={{ display: "flex", gap: 8, marginTop: 10, flexWrap: "wrap" }}>
                      <button
                        type="button"
                        className="lm-u-btn lm-u-btn-primary"
                        disabled={busy}
                        onClick={() => onClaim(s.stranger_id)}
                        style={{ fontSize: 11, padding: "5px 10px", display: "inline-flex", alignItems: "center", gap: 5 }}
                      >
                        <UserPlus size={12} />
                        {claimingId === s.stranger_id ? "Saving…" : "This is someone I know"}
                      </button>
                      <button
                        type="button"
                        className="lm-u-btn"
                        disabled={busy}
                        onClick={() => onForget(s.stranger_id)}
                        title="Forget this face"
                        style={{ fontSize: 11, padding: "5px 10px", display: "inline-flex", alignItems: "center", gap: 5, color: "var(--lm-text-dim)" }}
                      >
                        <Trash2 size={12} />
                        {forgettingId === s.stranger_id ? "…" : "Forget"}
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
