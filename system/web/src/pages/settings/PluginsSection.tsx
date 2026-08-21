import { useEffect, useState } from "react";
import { Blocks } from "lucide-react";
// Download, Heart — PARKED with the browse block (#213)
import { toast } from "sonner";
import { C, SectionCard, LABEL_STYLE, INPUT_STYLE } from "@/components/setup/shared";
import { listPlugins, installPlugin, installTrustedPlugin, startPlugin, stopPlugin, uninstallPlugin, getCompanionApps } from "@/lib/api";
import type { Plugin, CompanionApp } from "@/lib/api";
// PARKED (#213): plugin discovery moves off Hugging Face Spaces to our own
// catalog. Restore alongside the api.ts pair, the Go handler, and its route.
// import { searchHFPlugins } from "@/lib/api";
// import type { HFSpace } from "@/lib/api";

export function PluginsSection({ active }: { active: boolean }) {
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [loading, setLoading] = useState(true);
  const [url, setUrl] = useState("");
  const [installing, setInstalling] = useState(false);
  const [acting, setActing] = useState<string | null>(null);
  const [trusted, setTrusted] = useState<CompanionApp[]>([]);
  const isDebug = typeof window !== "undefined" && new URLSearchParams(window.location.search).get("debug") === "true";

  // HF browse state — PARKED (#213)
  // const [hfSpaces, setHfSpaces] = useState<HFSpace[]>([]);
  // const [hfLoading, setHfLoading] = useState(true);
  // const [hfInstalling, setHfInstalling] = useState<string | null>(null);

  function refresh() {
    listPlugins()
      .then(setPlugins)
      .catch(() => {})
      .finally(() => setLoading(false));
  }

  // PARKED (#213) — the browse half. Installing from a URL below is unaffected.
  // function refreshHF() {
  //   setHfLoading(true);
  //   searchHFPlugins()
  //     .then(setHfSpaces)
  //     .catch(() => {})
  //     .finally(() => setHfLoading(false));
  // }
  //
  // function hfUrlForSpace(id: string) {
  //   return `https://huggingface.co/spaces/${id}`;
  // }
  //
  // function isInstalled(spaceId: string) {
  //   const spaceUrl = hfUrlForSpace(spaceId);
  //   return plugins.some((p) => p.url === spaceUrl);
  // }
  //
  // async function handleHFInstall(spaceId: string) {
  //   setHfInstalling(spaceId);
  //   try {
  //     await installPlugin(hfUrlForSpace(spaceId));
  //     toast.success("Plugin install started.");
  //     setTimeout(refresh, 5000);
  //   } catch (err) {
  //     toast.error(err instanceof Error ? err.message : "Failed to install.");
  //   } finally {
  //     setHfInstalling(null);
  //   }
  // }

  useEffect(() => { refresh(); }, []);
  useEffect(() => {
    getCompanionApps()
      .then((list) => setTrusted(list.filter((a) => a.kind === "robot-app")))
      .catch(() => {});
  }, []);

  async function handleInstall() {
    const u = url.trim();
    if (!u) { toast.error("URL is required."); return; }
    setInstalling(true);
    try {
      await installPlugin(u);
      setUrl("");
      toast.success("Plugin install started. Refresh to check status.");
      setTimeout(refresh, 5000);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to install.");
    } finally {
      setInstalling(false);
    }
  }

  async function handleStart(name: string) {
    setActing(name);
    try {
      await startPlugin(name);
      setPlugins((prev) => prev.map((p) => p.name === name ? { ...p, status: "running" } : p));
      toast.success(`Started "${name}".`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to start.");
    } finally {
      setActing(null);
    }
  }

  async function handleStop(name: string) {
    setActing(name);
    try {
      await stopPlugin(name);
      setPlugins((prev) => prev.map((p) => p.name === name ? { ...p, status: "stopped" } : p));
      toast.success(`Stopped "${name}".`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to stop.");
    } finally {
      setActing(null);
    }
  }

  async function handleUninstall(name: string) {
    setActing(name);
    try {
      await uninstallPlugin(name);
      setPlugins((prev) => prev.filter((p) => p.name !== name));
      toast.success(`Uninstalled "${name}".`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to uninstall.");
    } finally {
      setActing(null);
    }
  }

  const BTN: React.CSSProperties = {
    padding: "6px 14px", borderRadius: 8, fontSize: 13, fontWeight: 500,
    cursor: "pointer", border: `1px solid ${C.border}`, background: C.surface,
    color: C.text,
  };

  const STATUS_COLOR: Record<string, string> = {
    running: C.green,
    stopped: C.textMuted,
    failed: C.red,
    installing: C.amber,
  };

  return (
    <SectionCard id="plugins" title="Plugins" icon={<Blocks size={17} />} active={active}>
      <div style={{ fontSize: 12.5, color: C.textDim, marginBottom: 14, lineHeight: 1.6 }}>
        Trusted plugins from this repo. They stay on the device. A raw git URL stays behind Advanced.
      </div>

      {trusted.length > 0 && (
        <div style={{ marginBottom: 16 }}>
          <div style={{ fontSize: 11, fontWeight: 600, color: C.textMuted, textTransform: "uppercase", letterSpacing: 0.5, marginBottom: 8 }}>
            Trusted
          </div>
          {trusted.map((app) => {
            const installed = plugins.some((p) => p.name === app.id);
            return (
              <div key={app.id} style={{
                display: "flex", alignItems: "center", justifyContent: "space-between",
                gap: 8, padding: "10px 12px", marginBottom: 6,
                background: C.surface, border: `1px solid ${C.border}`, borderRadius: 8,
              }}>
                <div style={{ minWidth: 0, flex: 1 }}>
                  <div style={{ fontSize: 13, fontWeight: 600 }}>{app.name}</div>
                  <div style={{ fontSize: 11, color: C.textDim }}>{app.summary}</div>
                </div>
                {installed ? (
                  <span style={{ fontSize: 11, color: C.green }}>Installed</span>
                ) : (
                  <button type="button" disabled={installing} style={BTN}
                    onClick={() => {
                      setInstalling(true);
                      installTrustedPlugin(app.id)
                        .then(() => { toast.success("Install started."); setTimeout(refresh, 4000); })
                        .catch((e: Error) => toast.error(e.message))
                        .finally(() => setInstalling(false));
                    }}
                  >Install</button>
                )}
              </div>
            );
          })}
        </div>
      )}

      {loading ? (
        <div style={{ fontSize: 12, color: C.textMuted }}>Loading...</div>
      ) : (
        <>
          {/* Installed plugins */}
          {plugins.length > 0 && (
            <div style={{ marginBottom: 16 }}>
              <div style={{ fontSize: 11, fontWeight: 600, color: C.textMuted, textTransform: "uppercase", letterSpacing: 0.5, marginBottom: 8 }}>
                Installed
              </div>
              {plugins.map((p) => (
                <div
                  key={p.name}
                  style={{
                    display: "flex", alignItems: "center", justifyContent: "space-between",
                    gap: 8, padding: "10px 12px", marginBottom: 6,
                    background: C.surface, border: `1px solid ${C.border}`, borderRadius: 8,
                  }}
                >
                  <div style={{ overflow: "hidden", minWidth: 0, flex: 1 }}>
                    <div style={{ fontSize: 13, fontWeight: 600, color: C.text, display: "flex", alignItems: "center", gap: 6 }}>
                      {p.name}
                      {p.version && (
                        <span style={{ fontSize: 10, color: C.textMuted, fontWeight: 400 }}>v{p.version}</span>
                      )}
                      <span style={{
                        fontSize: 10, fontWeight: 500,
                        color: STATUS_COLOR[p.status] || C.textMuted,
                      }}>
                        {p.status}
                      </span>
                    </div>
                    {p.description && (
                      <div style={{
                        fontSize: 11, color: C.textDim, marginTop: 2,
                        overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
                      }}>
                        {p.description}
                      </div>
                    )}
                  </div>
                  <div style={{ display: "flex", gap: 6, flexShrink: 0 }}>
                    {p.status === "running" ? (
                      <button
                        type="button"
                        onClick={() => handleStop(p.name)}
                        disabled={acting === p.name}
                        style={{ ...BTN, opacity: acting === p.name ? 0.5 : 1 }}
                      >
                        Stop
                      </button>
                    ) : (
                      <button
                        type="button"
                        onClick={() => handleStart(p.name)}
                        disabled={acting === p.name}
                        style={{ ...BTN, opacity: acting === p.name ? 0.5 : 1 }}
                      >
                        Start
                      </button>
                    )}
                    <button
                      type="button"
                      onClick={() => handleUninstall(p.name)}
                      disabled={acting === p.name}
                      style={{ ...BTN, color: C.red, opacity: acting === p.name ? 0.5 : 1 }}
                    >
                      Uninstall
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* PARKED (#213) — Browse listed plugins from Hugging Face Spaces.
              Restore with the api.ts pair + Go handler + route, pointing at
              our own catalog instead. Install-from-URL below is unaffected.
          <div style={{ marginBottom: 16 }}>
            <div style={{ fontSize: 11, fontWeight: 600, color: C.textMuted, textTransform: "uppercase", letterSpacing: 0.5, marginBottom: 8 }}>
              Browse
            </div>
            {hfLoading ? (
              <div style={{ fontSize: 12, color: C.textMuted }}>Loading plugins...</div>
            ) : hfSpaces.length === 0 ? (
              <div style={{ fontSize: 12, color: C.textMuted }}>No community plugins found.</div>
            ) : (
              hfSpaces.map((s) => {
                const installed = isInstalled(s.id);
                const busy = hfInstalling === s.id;
                const title = s.cardData?.title || s.id.split("/").pop() || s.id;
                const emoji = s.cardData?.emoji || "";
                const desc = s.cardData?.description || "";
                return (
                  <div
                    key={s.id}
                    style={{
                      display: "flex", alignItems: "center", justifyContent: "space-between",
                      gap: 8, padding: "10px 12px", marginBottom: 6,
                      background: C.surface, border: `1px solid ${C.border}`, borderRadius: 8,
                    }}
                  >
                    <div style={{ overflow: "hidden", minWidth: 0, flex: 1 }}>
                      <div style={{ fontSize: 13, fontWeight: 600, color: C.text, display: "flex", alignItems: "center", gap: 6 }}>
                        {emoji && <span>{emoji}</span>}
                        {title}
                        {s.likes > 0 && (
                          <span style={{ fontSize: 10, color: C.textMuted, fontWeight: 400, display: "inline-flex", alignItems: "center", gap: 2 }}>
                            <Heart size={9} /> {s.likes}
                          </span>
                        )}
                      </div>
                      {desc && (
                        <div style={{
                          fontSize: 11, color: C.textDim, marginTop: 2,
                          overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
                        }}>
                          {desc}
                        </div>
                      )}
                    </div>
                    <div style={{ flexShrink: 0 }}>
                      {installed ? (
                        <span style={{ fontSize: 11, color: C.green, fontWeight: 500 }}>Installed</span>
                      ) : (
                        <button
                          type="button"
                          onClick={() => handleHFInstall(s.id)}
                          disabled={busy}
                          style={{
                            ...BTN,
                            background: C.amber, color: "#000", borderColor: C.amber,
                            opacity: busy ? 0.5 : 1,
                            cursor: busy ? "not-allowed" : "pointer",
                            display: "inline-flex", alignItems: "center", gap: 4,
                          }}
                        >
                          <Download size={12} />
                          {busy ? "Installing..." : "Install"}
                        </button>
                      )}
                    </div>
                  </div>
                );
              })
            )}
          </div>

          */}

          {isDebug && (
            <>
              <div style={{ marginBottom: 6 }}>
                <label htmlFor="plugin-url" style={LABEL_STYLE}>Install from URL</label>
                <input
                  id="plugin-url"
                  type="text"
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                  placeholder="https://github.com/user/my-plugin"
                  style={INPUT_STYLE}
                />
              </div>
              <div style={{ display: "flex", gap: 8 }}>
                <button
                  type="button"
                  onClick={handleInstall}
                  disabled={installing || !url.trim()}
                  style={{
                    ...BTN,
                    background: C.amber, color: "#000", borderColor: C.amber,
                    opacity: installing || !url.trim() ? 0.5 : 1,
                    cursor: installing ? "not-allowed" : "pointer",
                  }}
                >
                  {installing ? "Installing..." : "Install"}
                </button>
                <button type="button" onClick={refresh} style={BTN}>Refresh</button>
              </div>
            </>
          )}
          {!isDebug && (
            <button type="button" onClick={refresh} style={BTN}>Refresh</button>
          )}
        </>
      )}
    </SectionCard>
  );
}
