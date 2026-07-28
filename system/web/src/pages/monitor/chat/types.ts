// Shared types for the chat composer's "+" menu → Skills flows.
//
// Three surfaces, three data shapes:
//   • WriteSkillModal   — SkillDraft (local form state, POSTed later)
//   • BrowseSkillsModal — StoreSkill[] from the Autonomous skill store, fetched
//                         through the Go proxy (GET /api/agent/skills/browse)
//   • ManageSkillsModal — InstalledSkill[] from the agentic runtime's skills dir

/** The three-field form of "Write skill". Mirrors a SKILL.md front-matter +
 *  body: name + description become front-matter, instructions become the body. */
export interface SkillDraft {
  name: string;
  description: string;
  instructions: string;
}

// StoreSkill (the skill-store listing entry) lives in lib/api.ts next to its
// fetch, mirroring how HFSpace sits beside searchHFPlugins.

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
