import { useState } from "react";
import { toast } from "sonner";
import { C, LockedField, LockedPasswordField, SectionCard } from "@/components/setup/shared";
import { testTTSVoice } from "@/lib/api";
import { ttsProviderLabel } from "@/lib/ttsLabels";
import { HW } from "@/pages/monitor/types";
import type { LlmLoadedState } from "@/hooks/setup/types";

export interface TtsLoadedState {
  apiKey: boolean;
  baseUrl: boolean;
}

// Edit-mode TTS exposes the api key + base URL fields so operators can override
// them per-section. Setup hides those because they auto-mirror from AI Brain.
export function TTSSection({
  active,
  ttsLoaded, llmLoaded,
  ttsApiKey, setTtsApiKey,
  ttsBaseUrl, setTtsBaseUrl,
  ttsProvider, setTtsProvider, ttsProviders,
  ttsVoice, setTtsVoice, ttsVoices,
  sttLanguage,
}: {
  active: boolean;
  ttsLoaded: TtsLoadedState;
  llmLoaded: LlmLoadedState;
  ttsApiKey: string; setTtsApiKey: (v: string) => void;
  ttsBaseUrl: string; setTtsBaseUrl: (v: string) => void;
  ttsProvider: string; setTtsProvider: (v: string) => void;
  ttsProviders: string[];
  ttsVoice: string; setTtsVoice: (v: string) => void;
  ttsVoices: string[];
  sttLanguage: string;
}) {
  const [testing, setTesting] = useState(false);

  async function preview() {
    if (testing) return;
    setTesting(true);
    try {
      await testTTSVoice(ttsVoice, { lang: sttLanguage, provider: ttsProvider });
      toast.success("Playing on the robot.");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "";
      if (/muted/i.test(msg)) {
        toast.error("Speaker is muted — unmute, then try again.", {
          action: {
            label: "Unmute",
            onClick: () => {
              void fetch(`${HW}/speaker/unmute`, { method: "POST" })
                .then(() => toast.message("Speaker unmuted. Test Voice again."))
                .catch(() => toast.error("Couldn't unmute."));
            },
          },
        });
      } else {
        toast.error(msg || "Couldn't play a preview.");
      }
    } finally {
      setTesting(false);
    }
  }

  return (
    <SectionCard id="tts" title="Voice" active={active} description="How the robot sounds when it talks.">
      <details style={{ marginBottom: 14 }}>
        <summary style={{ cursor: "pointer", fontSize: 12.5, fontWeight: 600, color: C.textDim }}>Advanced keys</summary>
        <div style={{ marginTop: 10 }}>
      <LockedPasswordField lockedInitially={ttsLoaded.apiKey || llmLoaded.apiKey} label="API Key (optional — leave blank to reuse the brain key)" id="tts_api_key" value={ttsApiKey} onChange={setTtsApiKey} placeholder="sk-..." />
      <LockedField lockedInitially={ttsLoaded.baseUrl || llmLoaded.baseUrl} label="Base URL (optional — leave blank to reuse the brain URL)" id="tts_base_url" value={ttsBaseUrl} onChange={setTtsBaseUrl} placeholder="https://api.openai.com/v1" />
        </div>
      </details>
      <div style={{ marginBottom: 12 }}>
        <label htmlFor="tts_provider" style={{ display: "block", fontSize: 11, color: C.textDim, marginBottom: 5 }}>
          Provider
        </label>
        <select
          id="tts_provider"
          value={ttsProvider}
          onChange={(e) => setTtsProvider(e.target.value)}
          style={{
            width: "100%", boxSizing: "border-box",
            background: C.card, border: `1px solid ${C.border}`,
            borderRadius: 7, padding: "8px 11px",
            fontSize: 12.5, color: C.text, outline: "none", cursor: "pointer",
          }}
        >
          {(ttsProviders.length > 0 ? ttsProviders : ["elevenlabs"]).map((p) => (
            <option key={p} value={p}>{ttsProviderLabel(p)}</option>
          ))}
        </select>
      </div>
      <div style={{ marginBottom: 12 }}>
        <label htmlFor="tts_voice" style={{ display: "block", fontSize: 11, color: C.textDim, marginBottom: 5 }}>
          Voice
        </label>
        <select
          id="tts_voice"
          value={ttsVoice}
          onChange={(e) => setTtsVoice(e.target.value)}
          style={{
            width: "100%", boxSizing: "border-box",
            background: C.card, border: `1px solid ${C.border}`,
            borderRadius: 7, padding: "8px 11px",
            fontSize: 12.5, color: C.text, outline: "none", cursor: "pointer",
          }}
        >
          {(ttsVoices.length > 0 ? ttsVoices : ["Rachel"]).map((v) => (
            <option key={v} value={v}>{v}</option>
          ))}
        </select>
        <button
          type="button"
          disabled={testing}
          onClick={() => void preview()}
          style={{
            marginTop: 8, width: "100%", padding: "8px 0",
            background: C.amber, color: "#fff", border: "none",
            borderRadius: 7, fontSize: 12, cursor: testing ? "wait" : "pointer", fontWeight: 600,
            opacity: testing ? 0.7 : 1,
          }}
        >
          {testing ? "Playing…" : "Test Voice"}
        </button>
        <p style={{ margin: "6px 0 0", fontSize: 11, color: C.textDim, lineHeight: 1.4 }}>
          Plays through the robot speaker. If Home shows Speaker MUTED, unmute first.
        </p>
      </div>
    </SectionCard>
  );
}
