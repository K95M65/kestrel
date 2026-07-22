import { useEffect, useState } from "react";
import { toast } from "sonner";
import { C, SectionCard, LABEL_STYLE, INPUT_STYLE } from "@/components/setup/shared";
import { listMCPTools, addMCPTool, removeMCPTool } from "@/lib/api";
import type { MCPTool } from "@/lib/api";

// MCP Tools section — manages public remote MCP tool endpoints (HF Spaces,
// community tools). Not part of the main form Save flow; each add/remove
// hits its own API endpoint and takes effect immediately (gateway restart).
// OAuth-authenticated connectors (Notion, GitHub, …) are managed separately
// via the MQTT connector.set flow.

export function MCPToolsSection({ active }: { active: boolean }) {
  const [tools, setTools] = useState<MCPTool[]>([]);
  const [loading, setLoading] = useState(true);
  const [adding, setAdding] = useState(false);
  const [name, setName] = useState("");
  const [url, setUrl] = useState("");
  const [apiKey, setApiKey] = useState("");

  useEffect(() => {
    listMCPTools()
      .then(setTools)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  async function handleAdd() {
    const n = name.trim();
    const u = url.trim();
    if (!n || !u) { toast.error("Name and URL are required."); return; }
    setAdding(true);
    const key = apiKey.trim() || undefined;
    try {
      await addMCPTool({ name: n, url: u, api_key: key });
      setTools((prev) => [...prev, { name: n, url: u, api_key: key }]);
      setName("");
      setUrl("");
      setApiKey("");
      toast.success(`Added MCP tool "${n}".`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to add tool.");
    } finally {
      setAdding(false);
    }
  }

  async function handleRemove(toolName: string) {
    try {
      await removeMCPTool(toolName);
      setTools((prev) => prev.filter((t) => t.name !== toolName));
      toast.success(`Removed "${toolName}".`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to remove tool.");
    }
  }

  const BTN: React.CSSProperties = {
    padding: "6px 14px", borderRadius: 8, fontSize: 13, fontWeight: 500,
    cursor: "pointer", border: `1px solid ${C.border}`, background: C.surface,
    color: C.text,
  };

  return (
    <SectionCard id="mcp" title="MCP Tools" active={active}>
      <div style={{ fontSize: 12.5, color: C.textDim, marginBottom: 14, lineHeight: 1.6 }}>
        Remote MCP tool endpoints the agent can call over HTTPS. Add public
        tools (HF Spaces, community servers) or authenticated tools (with API key).
        Each tool is synced to the active runtime and takes effect immediately.
      </div>

      {loading ? (
        <div style={{ fontSize: 12, color: C.textMuted }}>Loading…</div>
      ) : (
        <>
          {/* Configured tools list */}
          {tools.length > 0 && (
            <div style={{ marginBottom: 16 }}>
              {tools.map((t) => (
                <div
                  key={t.name}
                  style={{
                    display: "flex", alignItems: "center", justifyContent: "space-between",
                    gap: 8, padding: "10px 12px", marginBottom: 6,
                    background: C.surface, border: `1px solid ${C.border}`, borderRadius: 8,
                  }}
                >
                  <div style={{ overflow: "hidden", minWidth: 0 }}>
                    <div style={{ fontSize: 13, fontWeight: 600, color: C.text, display: "flex", alignItems: "center", gap: 6 }}>
                      {t.name}
                      {t.api_key && (
                        <span style={{ fontSize: 10, color: C.amber, fontWeight: 400 }}>key</span>
                      )}
                    </div>
                    <div style={{
                      fontSize: 11, color: C.textMuted,
                      overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
                    }}>
                      {t.url}
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={() => handleRemove(t.name)}
                    style={{ ...BTN, color: "#ef4444", flexShrink: 0 }}
                  >
                    Remove
                  </button>
                </div>
              ))}
            </div>
          )}

          {tools.length === 0 && (
            <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 14 }}>
              No MCP tools configured yet.
            </div>
          )}

          {/* Add form */}
          <div style={{ marginBottom: 6 }}>
            <label htmlFor="mcp-name" style={LABEL_STYLE}>Name</label>
            <input
              id="mcp-name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="search"
              style={INPUT_STYLE}
            />
          </div>
          <div style={{ marginBottom: 6 }}>
            <label htmlFor="mcp-url" style={LABEL_STYLE}>URL</label>
            <input
              id="mcp-url"
              type="text"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://owner-space.hf.space/gradio_api/mcp/"
              style={INPUT_STYLE}
            />
          </div>
          <div style={{ marginBottom: 10 }}>
            <label htmlFor="mcp-apikey" style={LABEL_STYLE}>API Key <span style={{ color: C.textMuted, fontWeight: 400 }}>(optional)</span></label>
            <input
              id="mcp-apikey"
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="sk-..."
              style={INPUT_STYLE}
            />
          </div>
          <button
            type="button"
            onClick={handleAdd}
            disabled={adding || !name.trim() || !url.trim()}
            style={{
              ...BTN,
              background: C.amber, color: "#000", borderColor: C.amber,
              opacity: adding || !name.trim() || !url.trim() ? 0.5 : 1,
              cursor: adding ? "not-allowed" : "pointer",
            }}
          >
            {adding ? "Adding…" : "Add Tool"}
          </button>
        </>
      )}
    </SectionCard>
  );
}
