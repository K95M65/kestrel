import { useEffect, useState } from "react";
import { Brain } from "lucide-react";
import { ConfiguredHint, LockedField, LockedPasswordField, SectionCard, C } from "./shared";
import type { LlmLoadedState } from "@/hooks/setup/types";
import {
  getLLMProviders,
  startLLMOAuth,
  pollLLMOAuth,
  type LLMProvider,
} from "@/lib/api";

function expandURL(tmpl: string, extras: Record<string, string>): string {
  let out = tmpl || "";
  for (const [k, v] of Object.entries(extras)) {
    out = out.replaceAll(`{${k}}`, v.trim());
  }
  return out;
}

const selectStyle = {
  width: "100%",
  boxSizing: "border-box" as const,
  background: C.surface,
  border: `1px solid ${C.border}`,
  borderRadius: 7,
  padding: "8px 11px",
  fontSize: 13,
  color: C.text,
  outline: "none",
  cursor: "pointer",
  marginBottom: 12,
};

export function LLMSection({
  active, llmLoaded,
  llmApiKey, setLlmApiKey,
  llmUrl, setLlmUrl,
  llmModel, setLlmModel,
}: {
  active: boolean;
  llmLoaded: LlmLoadedState;
  llmApiKey: string; setLlmApiKey: (v: string) => void;
  llmUrl: string; setLlmUrl: (v: string) => void;
  llmModel: string; setLlmModel: (v: string) => void;
}) {
  const [providers, setProviders] = useState<LLMProvider[]>([]);
  const [providerKey, setProviderKey] = useState("");
  const [extras, setExtras] = useState<Record<string, string>>({});
  const [oauthBusy, setOauthBusy] = useState(false);
  const [oauthErr, setOauthErr] = useState<string | null>(null);
  const [oauth, setOauth] = useState<{
    userCode: string;
    uri: string;
    deviceCode: string;
    interval: number;
  } | null>(null);

  useEffect(() => {
    getLLMProviders().then(setProviders).catch(() => {});
  }, []);

  const selected = providers.find((p) => p.key === providerKey);

  const applyProvider = (key: string) => {
    setProviderKey(key);
    setOauth(null);
    setOauthErr(null);
    setExtras({});
    const p = providers.find((x) => x.key === key);
    if (!p) return;
    if (p.base_url && !p.base_url.includes("{")) setLlmUrl(p.base_url);
    if (p.default_model) setLlmModel(p.default_model);
    if (p.auth === "byo" && p.key === "ollama" && !llmApiKey) setLlmApiKey("ollama");
    if (p.auth === "byo" && p.key === "lmstudio" && !llmApiKey) setLlmApiKey("lmstudio");
  };

  const setExtra = (k: string, v: string) => {
    const next = { ...extras, [k]: v };
    setExtras(next);
    if (selected?.base_url) setLlmUrl(expandURL(selected.base_url, next));
  };

  const startGrok = async () => {
    setOauthBusy(true);
    setOauthErr(null);
    try {
      const started = await startLLMOAuth("xai");
      setOauth({
        userCode: started.user_code,
        uri: started.verification_uri_complete || started.verification_uri,
        deviceCode: started.device_code,
        interval: Math.max(3, started.interval || 5),
      });
      if (started.base_url) setLlmUrl(started.base_url);
      if (started.default_model) setLlmModel(started.default_model);
    } catch (e) {
      setOauthErr(e instanceof Error ? e.message : "Could not start Grok login");
    } finally {
      setOauthBusy(false);
    }
  };

  useEffect(() => {
    if (!oauth) return;
    let stop = false;
    const tick = async () => {
      try {
        const r = await pollLLMOAuth("xai", oauth.deviceCode);
        if (stop) return;
        if (r.pending) {
          const next = Math.max(3, r.interval || oauth.interval);
          if (next !== oauth.interval) {
            setOauth({ ...oauth, interval: next });
          }
          return;
        }
        if (r.access_token) {
          setLlmApiKey(r.access_token);
          if (r.base_url) setLlmUrl(r.base_url);
          if (r.default_model) setLlmModel(r.default_model);
        }
        setOauth(null);
      } catch (e) {
        if (!stop) {
          setOauthErr(e instanceof Error ? e.message : "Grok login failed");
          setOauth(null);
        }
      }
    };
    const id = window.setInterval(tick, oauth.interval * 1000);
    void tick();
    return () => {
      stop = true;
      window.clearInterval(id);
    };
  }, [oauth, setLlmApiKey, setLlmUrl, setLlmModel]);

  return (
    <SectionCard id="llm" title="AI Brain" active={active} icon={<Brain size={17} />}
      description="Pick a provider the way OpenCode /connect does. Grok can sign in with your subscription. Everyone else uses an API key or a local URL.">
      <label htmlFor="llm_provider" style={{ display: "block", fontSize: 12, color: C.textDim, marginBottom: 6 }}>
        Provider
      </label>
      <select
        id="llm_provider"
        value={providerKey}
        onChange={(e) => applyProvider(e.target.value)}
        style={selectStyle}
      >
        <option value="">Select a provider…</option>
        {providers.map((p) => (
          <option key={p.key} value={p.key}>{p.name}</option>
        ))}
      </select>

      {selected?.hint && (
        <div style={{ fontSize: 12, color: C.textDim, lineHeight: 1.5, marginBottom: 12 }}>{selected.hint}</div>
      )}
      {selected?.docs_url && (
        <a href={selected.docs_url} target="_blank" rel="noreferrer"
          style={{ display: "inline-block", fontSize: 12, color: C.amber, marginBottom: 12 }}>
          Provider docs
        </a>
      )}

      {selected?.auth === "device_code" && (
        <div style={{ marginBottom: 14 }}>
          <button
            type="button"
            onClick={() => void startGrok()}
            disabled={oauthBusy || !!oauth}
            style={{
              padding: "8px 14px",
              borderRadius: 7,
              fontSize: 13,
              fontWeight: 600,
              border: "none",
              cursor: oauthBusy || oauth ? "not-allowed" : "pointer",
              background: C.amber,
              color: C.bg,
            }}
          >
            {oauthBusy ? "Starting…" : "Sign in with Grok"}
          </button>
          {oauth && (
            <div style={{ marginTop: 12, fontSize: 13, color: C.text, lineHeight: 1.5 }}>
              Open <a href={oauth.uri} target="_blank" rel="noreferrer" style={{ color: C.amber }}>{oauth.uri}</a>
              <div style={{
                marginTop: 8, fontFamily: "ui-monospace, monospace",
                fontSize: 22, letterSpacing: "0.18em", fontWeight: 700,
              }}>
                {oauth.userCode}
              </div>
              <div style={{ fontSize: 12, color: C.textMuted, marginTop: 6 }}>Waiting for authorization…</div>
            </div>
          )}
          {oauthErr && <div style={{ marginTop: 8, fontSize: 12, color: C.red }}>{oauthErr}</div>}
        </div>
      )}

      {(selected?.fields ?? []).map((f) => (
        f.secret ? (
          <LockedPasswordField
            key={f.key}
            lockedInitially={false}
            label={f.label}
            id={`llm_extra_${f.key}`}
            value={extras[f.key] || ""}
            onChange={(v) => setExtra(f.key, v)}
            placeholder={f.placeholder}
          />
        ) : (
          <LockedField
            key={f.key}
            lockedInitially={false}
            label={f.label}
            id={`llm_extra_${f.key}`}
            value={extras[f.key] || ""}
            onChange={(v) => setExtra(f.key, v)}
            placeholder={f.placeholder}
          />
        )
      ))}

      {llmLoaded.apiKey && !llmApiKey ? (
        <ConfiguredHint label="API Key" />
      ) : (
        <LockedPasswordField lockedInitially={false} label="API Key" id="llm_api_key" value={llmApiKey} onChange={setLlmApiKey} placeholder={selected?.auth === "byo" ? "ollama" : "sk-..."} />
      )}
      <LockedField lockedInitially={llmLoaded.baseUrl && !selected} label="Base URL" id="llm_url" value={llmUrl} onChange={setLlmUrl} placeholder="https://api.openai.com/v1" />
      <LockedField lockedInitially={llmLoaded.model && !selected} label="Model" id="llm_model" value={llmModel} onChange={setLlmModel} placeholder="gpt-4o-mini" />
    </SectionCard>
  );
}
