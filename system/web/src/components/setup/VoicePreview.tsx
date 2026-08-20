import { useState, type CSSProperties } from "react";
import { toast } from "sonner";
import { C } from "./shared";
import { previewTTSInBrowser, testTTSVoice } from "@/lib/api";
import { HW } from "@/pages/monitor/types";

type Dest = "robot" | "browser";

export function VoicePreview({
  ttsVoice,
  ttsProvider,
  sttLanguage,
  fullWidth = true,
}: {
  ttsVoice: string;
  ttsProvider: string;
  sttLanguage: string;
  fullWidth?: boolean;
}) {
  const [testing, setTesting] = useState<Dest | null>(null);

  async function preview(dest: Dest) {
    if (testing) return;
    setTesting(dest);
    try {
      if (dest === "robot") {
        await testTTSVoice(ttsVoice, { lang: sttLanguage, provider: ttsProvider });
        toast.success("Playing on the robot.");
      } else {
        await previewTTSInBrowser(ttsVoice, { lang: sttLanguage, provider: ttsProvider });
        toast.success("Playing in this browser.");
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : "";
      if (dest === "robot" && /muted/i.test(msg)) {
        toast.error("Speaker is muted — unmute, then try again.", {
          action: {
            label: "Unmute",
            onClick: () => {
              void fetch(`${HW}/speaker/unmute`, { method: "POST" })
                .then(() => toast.message("Speaker unmuted. Test on the robot again."))
                .catch(() => toast.error("Couldn't unmute."));
            },
          },
        });
      } else {
        toast.error(msg || "Couldn't play a preview.");
      }
    } finally {
      setTesting(null);
    }
  }

  const btn = (dest: Dest, primary: boolean): CSSProperties => ({
    flex: 1,
    padding: "8px 0",
    background: primary ? C.amber : "transparent",
    color: primary ? "var(--lm-on-amber)" : C.text,
    border: primary ? "none" : `1px solid ${C.border}`,
    borderRadius: 7,
    fontSize: 12,
    cursor: testing ? "wait" : "pointer",
    fontWeight: 600,
    fontFamily: "inherit",
    opacity: testing && testing !== dest ? 0.55 : testing === dest ? 0.7 : 1,
  });

  return (
    <div style={{ marginTop: 8, width: fullWidth ? "100%" : undefined }}>
      <div style={{ display: "flex", gap: 8 }}>
        <button
          type="button"
          disabled={!!testing}
          onClick={() => void preview("robot")}
          style={btn("robot", true)}
        >
          {testing === "robot" ? "Playing…" : "On robot"}
        </button>
        <button
          type="button"
          disabled={!!testing}
          onClick={() => void preview("browser")}
          style={btn("browser", false)}
        >
          {testing === "browser" ? "Playing…" : "In this browser"}
        </button>
      </div>
      <p style={{ margin: "6px 0 0", fontSize: 11, color: C.textDim, lineHeight: 1.4 }}>
        On robot uses the speaker. In this browser plays the same voice here — even if the robot is muted.
      </p>
    </div>
  );
}
