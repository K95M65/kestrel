// Shared types for the chat composer's "+" menu → Skills flows.
//
// SkillDraft (Write skill) and StoreSkill (Browse skills) live in lib/api.ts
// next to their requests, mirroring how HFSpace sits beside searchHFPlugins.
// What's left here is the Manage-skills shape, which has no endpoint yet.

/** A node in an installed skill's file tree. `children` is only set on dirs;
 *  a leaf with no children is a file. */
export interface SkillNode {
  name: string;
  /** Path relative to the skills root, e.g. "music/reference/tempo.md". */
  path: string;
  dir?: boolean;
  size?: number;
  children?: SkillNode[];
}

/** One skill installed in the active agentic runtime. `name` is the directory
 *  name (rendered as "/music"), `files` is its tree. */
export interface InstalledSkill {
  name: string;
  description?: string;
  files: SkillNode[];
}
