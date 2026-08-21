/** Talk @skill — pin a skill for this turn. */

export type SkillHint = { name: string; description?: string };

export function mentionQuery(input: string, caret = input.length): { at: number; q: string } | null {
  const left = input.slice(0, caret);
  const m = left.match(/(^|\s)@([a-zA-Z0-9_-]*)$/);
  if (!m) return null;
  return { at: left.length - m[2].length - 1, q: m[2].toLowerCase() };
}

export function filterSkills(skills: SkillHint[], q: string): SkillHint[] {
  const n = q.trim().toLowerCase();
  const list = skills.filter((s) => s.name && s.name !== "skill-creator");
  if (!n) return list.slice(0, 8);
  return list.filter((s) => s.name.toLowerCase().includes(n) || (s.description || "").toLowerCase().includes(n)).slice(0, 8);
}

export function insertMention(input: string, at: number, name: string, caret = input.length): string {
  const after = input.slice(caret);
  return `${input.slice(0, at)}@${name} ${after}`.replace(/\s+$/, " ").trimEnd() + (after.startsWith(" ") ? "" : " ");
}

/** Turn a leading @skill into a marker the brain already honors. */
export function applySkillMention(text: string): string {
  const m = text.match(/^@([a-z0-9_-]{1,64})\b\s*/i);
  if (!m) return text;
  const rest = text.slice(m[0].length);
  return `[use-skill: ${m[1].toLowerCase()}] ${rest}`.trim();
}
