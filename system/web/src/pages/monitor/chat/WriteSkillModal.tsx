import { useState } from "react";
import { PenLine } from "lucide-react";
import { ModalShell } from "./ModalShell";
import { fieldLabel, inputStyle, btnStyle } from "./styles";
import type { SkillDraft } from "./types";

// "Write skill" — three-field authoring form (name / description /
// instructions), the same shape as a SKILL.md: name + description become the
// front-matter, instructions become the body.
//
// UI ONLY for now: onSubmit is not wired to a backend yet. When the Go endpoint
// lands, replace the `onSubmit` prop's implementation in ChatSection — the form
// already validates the name shape and reports server errors through `error`.

// Skill dir names must be filesystem- and prompt-safe: the runtime addresses a
// skill as "/<name>", so the same lowercase/digit/dash/underscore rule the Go
// side enforces (roleNamePattern) applies here for instant feedback.
const NAME_PATTERN = /^[a-z0-9_-]+$/;

export function WriteSkillModal({
  onClose, onSubmit,
}: {
  onClose: () => void;
  /** Resolves when the skill is saved. Rejects with a message to show inline. */
  onSubmit?: (draft: SkillDraft) => Promise<void>;
}) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [instructions, setInstructions] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const nameOk = NAME_PATTERN.test(name.trim());
  const canSave = nameOk && description.trim() !== "" && instructions.trim() !== "" && !saving;

  const submit = async () => {
    if (!canSave) return;
    const draft: SkillDraft = {
      name: name.trim(),
      description: description.trim(),
      instructions: instructions.trim(),
    };
    if (!onSubmit) {
      // No backend wired yet — surface that instead of silently doing nothing.
      setError("Saving isn't wired up yet — the backend endpoint is still to come.");
      return;
    }
    setSaving(true);
    setError("");
    try {
      await onSubmit(draft);
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save skill");
    } finally {
      setSaving(false);
    }
  };

  return (
    <ModalShell
      icon={PenLine}
      title="Write skill"
      subtitle="Author a new skill for this agent"
      width={560}
      onClose={onClose}
      footer={
        <>
          <button type="button" className="lm-u-btn" style={btnStyle} onClick={onClose}>Cancel</button>
          <button
            type="button"
            className="lm-u-btn lm-u-btn-primary"
            style={{ ...btnStyle, opacity: canSave ? 1 : 0.5, cursor: canSave ? "pointer" : "not-allowed" }}
            disabled={!canSave}
            onClick={submit}
          >{saving ? "Saving…" : "Save skill"}</button>
        </>
      }
    >
      <div style={{ display: "flex", flexDirection: "column", gap: 15 }}>
        <div>
          <label htmlFor="skill-name" style={fieldLabel}>Skill name</label>
          <input
            id="skill-name"
            autoFocus
            value={name}
            onChange={(e) => { setName(e.target.value); if (error) setError(""); }}
            placeholder="weekly-status-report"
            className="lm-u-input"
            style={inputStyle}
          />
          <div style={{
            fontSize: 10.5, marginTop: 5,
            color: name && !nameOk ? "var(--lm-red)" : "var(--lm-text-muted)",
          }}>
            {name && !nameOk
              ? "Lowercase letters, digits, _ and - only."
              : "Becomes the skill directory — the agent invokes it as /" + (nameOk ? name.trim() : "name") + "."}
          </div>
        </div>

        <div>
          <label htmlFor="skill-description" style={fieldLabel}>Description</label>
          <input
            id="skill-description"
            value={description}
            onChange={(e) => { setDescription(e.target.value); if (error) setError(""); }}
            placeholder="Summarise the week's activity into a short status report."
            className="lm-u-input"
            style={inputStyle}
          />
          <div style={{ fontSize: 10.5, color: "var(--lm-text-muted)", marginTop: 5 }}>
            One line. This is what the agent reads when deciding whether to load the skill.
          </div>
        </div>

        <div>
          <label htmlFor="skill-instructions" style={fieldLabel}>Instructions</label>
          <textarea
            id="skill-instructions"
            value={instructions}
            onChange={(e) => { setInstructions(e.target.value); if (error) setError(""); }}
            placeholder={"When the user asks for a weekly status report:\n1. Collect what happened since last Monday.\n2. Group it into Done / In progress / Blocked.\n3. Keep it under 10 bullets, no filler."}
            rows={10}
            className="lm-u-input"
            style={{ ...inputStyle, resize: "vertical", minHeight: 160, lineHeight: 1.55 }}
          />
          <div style={{ fontSize: 10.5, color: "var(--lm-text-muted)", marginTop: 5 }}>
            The body of SKILL.md — markdown, written as instructions to the agent.
          </div>
        </div>

        {error && (
          <div style={{
            fontSize: 11.5, color: "var(--lm-red)",
            background: "var(--lm-red-dim)",
            border: "1px solid var(--lm-red-glow)",
            borderRadius: 9, padding: "8px 10px",
          }}>{error}</div>
        )}
      </div>
    </ModalShell>
  );
}
