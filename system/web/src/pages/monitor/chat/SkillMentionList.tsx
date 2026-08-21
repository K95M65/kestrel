import type { SkillHint } from "@/lib/mentionSkills";

export function SkillMentionList({
  items, onPick,
}: {
  items: SkillHint[];
  onPick: (name: string) => void;
}) {
  if (items.length === 0) return null;
  return (
    <div
      role="listbox"
      className="lm-pop"
      style={{
        position: "absolute", bottom: "100%", left: 44, marginBottom: 6,
        minWidth: 220, maxWidth: 320, zIndex: 20,
        background: "var(--lm-card)", border: "1px solid var(--lm-border)",
        borderRadius: 10, padding: 6, boxShadow: "0 8px 24px -12px rgba(0,0,0,0.5)",
      }}
    >
      {items.map((s) => (
        <button
          key={s.name}
          type="button"
          role="option"
          onClick={() => onPick(s.name)}
          style={{
            display: "block", width: "100%", textAlign: "left",
            background: "transparent", border: "none", color: "var(--lm-text)",
            padding: "7px 8px", borderRadius: 7, cursor: "pointer", fontSize: 13,
          }}
        >
          <strong>@{s.name}</strong>
          {s.description && (
            <span style={{ display: "block", fontSize: 11, color: "var(--lm-text-muted)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
              {s.description}
            </span>
          )}
        </button>
      ))}
    </div>
  );
}
