import { useEffect, useState } from "react";
import {
  FolderTree, ChevronRight, ChevronDown, Folder, FolderOpen, FileText, Loader2,
} from "lucide-react";
import { ModalShell } from "./ModalShell";
import type { InstalledSkill, SkillNode } from "./types";

// "Manage skills" — the skills currently installed in the active agentic
// runtime. Click a skill to expand its file tree (SKILL.md, reference/, …).
//
// UI ONLY for now: `loadInstalledSkills` returns a fixed sample so the layout
// can be reviewed. Swap its body for an apiRequest call once the Go endpoint
// exists — nothing else in this file changes.

async function loadInstalledSkills(): Promise<InstalledSkill[]> {
  return SAMPLE_SKILLS;
}

export function ManageSkillsModal({ onClose }: { onClose: () => void }) {
  const [skills, setSkills] = useState<InstalledSkill[]>([]);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<string | null>(null);

  useEffect(() => {
    let alive = true;
    void loadInstalledSkills().then((list) => {
      if (!alive) return;
      setSkills(list);
      setLoading(false);
    });
    return () => { alive = false; };
  }, []);

  return (
    <ModalShell
      icon={FolderTree}
      title="Manage skills"
      subtitle={loading ? "Loading…" : `${skills.length} installed on this runtime`}
      width={560}
      onClose={onClose}
    >
      {loading ? (
        <div style={{
          display: "flex", alignItems: "center", justifyContent: "center", gap: 8,
          padding: "44px 12px", fontSize: 12.5, color: "var(--lm-text-muted)",
        }}><Loader2 size={18} className="lm-spin-ico" /> Loading skills…</div>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
          {skills.map((s) => {
            const open = expanded === s.name;
            return (
              <div
                key={s.name}
                style={{
                  borderRadius: 10, overflow: "hidden",
                  background: "var(--lm-card)", border: "1px solid var(--lm-border)",
                }}
              >
                <button
                  type="button"
                  onClick={() => setExpanded(open ? null : s.name)}
                  aria-expanded={open}
                  style={{
                    display: "flex", alignItems: "center", gap: 8, width: "100%",
                    padding: "10px 12px", background: "transparent", border: "none",
                    cursor: "pointer", textAlign: "left", color: "var(--lm-text)",
                  }}
                >
                  {open ? <ChevronDown size={14} style={{ color: "var(--lm-text-muted)", flexShrink: 0 }} />
                        : <ChevronRight size={14} style={{ color: "var(--lm-text-muted)", flexShrink: 0 }} />}
                  {open ? <FolderOpen size={15} style={{ color: "var(--lm-amber)", flexShrink: 0 }} />
                        : <Folder size={15} style={{ color: "var(--lm-amber)", flexShrink: 0 }} />}
                  <span style={{ flex: 1, minWidth: 0 }}>
                    <span style={{ display: "block", fontSize: 13, fontWeight: 600 }}>/{s.name}</span>
                    {s.description && (
                      <span style={{
                        display: "block", fontSize: 11, color: "var(--lm-text-muted)",
                        marginTop: 2, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
                      }}>{s.description}</span>
                    )}
                  </span>
                </button>

                {open && (
                  <div style={{
                    padding: "4px 12px 10px 12px",
                    borderTop: "1px solid var(--lm-border)",
                  }}>
                    <Tree nodes={s.files} depth={0} />
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </ModalShell>
  );
}

// Recursive file tree. Directories start expanded — a skill tree is small
// enough that showing it whole beats making the user click through it.
function Tree({ nodes, depth }: { nodes: SkillNode[]; depth: number }) {
  return (
    <div style={{ display: "flex", flexDirection: "column" }}>
      {nodes.map((n) => <TreeNode key={n.path} node={n} depth={depth} />)}
    </div>
  );
}

function TreeNode({ node, depth }: { node: SkillNode; depth: number }) {
  const [open, setOpen] = useState(true);
  const isDir = Boolean(node.dir);

  return (
    <>
      <div
        onClick={isDir ? () => setOpen((v) => !v) : undefined}
        style={{
          display: "flex", alignItems: "center", gap: 6,
          padding: "4px 6px", paddingLeft: 6 + depth * 16,
          borderRadius: 6, fontSize: 11.5,
          color: isDir ? "var(--lm-text)" : "var(--lm-text-dim)",
          cursor: isDir ? "pointer" : "default",
          fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
        }}
        onMouseEnter={(e) => { e.currentTarget.style.background = "color-mix(in srgb, var(--lm-text) 6%, transparent)"; }}
        onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
      >
        {isDir
          ? (open ? <ChevronDown size={12} style={{ color: "var(--lm-text-muted)", flexShrink: 0 }} />
                  : <ChevronRight size={12} style={{ color: "var(--lm-text-muted)", flexShrink: 0 }} />)
          : <span style={{ width: 12, flexShrink: 0 }} />}
        {isDir
          ? <Folder size={13} style={{ color: "var(--lm-text-muted)", flexShrink: 0 }} />
          : <FileText size={13} style={{ color: "var(--lm-text-muted)", flexShrink: 0 }} />}
        <span style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
          {node.name}{isDir ? "/" : ""}
        </span>
      </div>
      {isDir && open && node.children && <Tree nodes={node.children} depth={depth + 1} />}
    </>
  );
}

// ─── Sample data (UI preview only — replace with the real listing) ───────────

const SAMPLE_SKILLS: InstalledSkill[] = [
  {
    name: "music",
    description: "Play, queue and control music on the device speaker.",
    files: [
      { name: "SKILL.md", path: "music/SKILL.md" },
      {
        name: "reference", path: "music/reference", dir: true,
        children: [
          { name: "providers.md", path: "music/reference/providers.md" },
          { name: "playlists.md", path: "music/reference/playlists.md" },
        ],
      },
      {
        name: "scripts", path: "music/scripts", dir: true,
        children: [{ name: "resolve_track.py", path: "music/scripts/resolve_track.py" }],
      },
    ],
  },
  {
    name: "voice",
    description: "Speak, change voice and tune TTS behaviour.",
    files: [
      { name: "SKILL.md", path: "voice/SKILL.md" },
      {
        name: "reference", path: "voice/reference", dir: true,
        children: [{ name: "voices.md", path: "voice/reference/voices.md" }],
      },
    ],
  },
];
