import { useState } from "react";
import { Volume2 } from "lucide-react";
import { toast } from "sonner";
import { C, SectionCard, LABEL_STYLE, INPUT_STYLE, FIELD_GAP } from "./shared";
import { testTTSVoice } from "@/lib/api";
import { ttsProviderLabel } from "@/lib/ttsLabels";
import { HW } from "@/pages/monitor/types";

export function TTSSection({
  active, isContinue,
  ttsProvider, setTtsProvider, ttsProviders,
  ttsVoice, setTtsVoice, ttsVoices,
  sttLanguage,
}: {
  active: boolean;
  isContinue: boolean;
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
    <SectionCard id="tts" title="Voice" active={active} icon={<Volume2 size={17} />}
      description="Choose how your device sounds when it speaks back to you.">
      {/* tts_api_key + tts_base_url are not exposed in Setup —
          they're auto-mirrored from AI Brain via useEffect and
          submitted silently. */}
      <div style={{ marginBottom: FIELD_GAP }}>
        <label htmlFor="tts_provider" style={LABEL_STYLE}>
          Provider
        </label>
        <select
          id="tts_provider"
          value={ttsProvider}
          onChange={(e) => setTtsProvider(e.target.value)}
          style={{ ...INPUT_STYLE, cursor: "pointer" }}
        >
          {(ttsProviders.length > 0 ? ttsProviders : ["elevenlabs"]).map((p) => (
            <option key={p} value={p}>{ttsProviderLabel(p)}</option>
          ))}
        </select>
      </div>
      <div style={{ marginBottom: FIELD_GAP }}>
        <label htmlFor="tts_voice" style={LABEL_STYLE}>
          Voice
        </label>
        <select
          id="tts_voice"
          value={ttsVoice}
          onChange={(e) => setTtsVoice(e.target.value)}
          style={{ ...INPUT_STYLE, cursor: "pointer" }}
        >
          {(ttsVoices.length > 0 ? ttsVoices : ["Rachel"]).map((v) => (
            <option key={v} value={v}>{v}</option>
          ))}
        </select>
        {isContinue ? (
          <button
            type="button"
            disabled={testing}
            onClick={() => void preview()}
            style={{
              marginTop: 10, width: "100%", padding: "10px 0",
              background: C.amber, color: "var(--lm-on-amber)", border: "none",
              borderRadius: 8, fontSize: 13, cursor: testing ? "wait" : "pointer", fontWeight: 600,
              opacity: testing ? 0.7 : 1,
            }}
          >
            {testing ? "Playing…" : "Test Voice"}
          </button>
        ) : (
          <div style={{ marginTop: 8, fontSize: 12.5, color: C.textDim }}>
            You can preview voices after your device is online (next step).
          </div>
        )}
      </div>
    </SectionCard>
  );
}
