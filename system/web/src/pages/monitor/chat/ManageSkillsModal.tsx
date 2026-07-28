import { useCallback, useEffect, useState } from "react";
import {
  FolderTree, ChevronRight, ChevronDown, Folder, FolderOpen, FileText, Loader2,
  RefreshCw, AlertCircle,
} from "lucide-react";
import { listInstalledSkills } from "@/lib/api";
import type { InstalledSkill, SkillNode } from "@/lib/api";
import { ModalShell } from "./ModalShell";

// "Manage skills" — the skills currently present in the ACTIVE agentic
// runtime's skills dir (GET /api/agent/skills → AgentGateway.ListSkills).
// Click a skill to expand its file tree (SKILL.md, reference/, …).
//
// Everything the runtime has shows up here regardless of how it got there:
// authored, store-installed, role-bundled and OTA-pushed skills all land in the
// same tree. A runtime that can't list skills answers 501 and the message is
// shown inline — an empty list means "provisioned but empty", not "unsupported".

export function ManageSkillsModal({ onClose }: { onClose: () => void }) {
  const [skills, setSkills] = useState<InstalledSkill[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [expanded, setExpanded] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setSkills(await listInstalledSkills());
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to list skills");
      setSkills([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  const subtitle = loading
    ? "Loading…"
    : error
      ? "Could not read the runtime's skills"
      : `${skills.length} installed on this runtime`;

  return (
    <ModalShell
      icon={FolderTree}
      title="Manage skills"
      subtitle={subtitle}
      width={560}
      onClose={onClose}
    >
      <div style={{ display: "flex", justifyContent: "flex-end", marginBottom: 10 }}>
        <button
          type="button"
          className="lm-u-btn"
          onClick={() => void load()}
          disabled={loading}
          title="Reload"
          aria-label="Reload"
          style={{
            width: 32, height: 30, borderRadius: 8, display: "flex",
            alignItems: "center", justifyContent: "center",
          }}
        ><RefreshCw size={13} className={loading ? "lm-spin-ico" : undefined} /></button>
      </div>

      {loading ? (
        <div style={{
          display: "flex", alignItems: "center", justifyContent: "center", gap: 8,
          padding: "44px 12px", fontSize: 12.5, color: "var(--lm-text-muted)",
        }}><Loader2 size={18} className="lm-spin-ico" /> Loading skills…</div>
      ) : error ? (
        <div style={{
          display: "flex", alignItems: "center", justifyContent: "center", gap: 8,
          padding: "44px 12px", textAlign: "center", fontSize: 12.5, color: "var(--lm-red)",
        }}><AlertCircle size={16} /> {error}</div>
      ) : skills.length === 0 ? (
        <div style={{
          display: "flex", alignItems: "center", justifyContent: "center",
          padding: "44px 12px", textAlign: "center", fontSize: 12.5, color: "var(--lm-text-muted)",
        }}>No skills installed on this runtime yet.</div>
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
