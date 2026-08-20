import { useRef, useState } from "react";
import { Mic, Loader2, X } from "lucide-react";
import { pickVoicePhrases, VOICE_DURATION_SEC } from "@/components/setup/voice-phrases";
import { hwUrl } from "@/lib/api";
import { voiceEnrollTarget } from "@/lib/voiceEnroll";

export function ContactVoiceBar({
  name, sttLanguage, onDone, onClose,
}: {
  name: string;
  sttLanguage: string;
  onDone: () => void;
  onClose: () => void;
}) {
  const phrases = pickVoicePhrases(sttLanguage);
  const [phase, setPhase] = useState<"idle" | "countdown" | "recording" | "processing">("idle");
  const [countdown, setCountdown] = useState(0);
  const [msg, setMsg] = useState<string | null>(null);
  const tickRef = useRef<number | null>(null);

  function start() {
    let target: string;
    try {
      target = voiceEnrollTarget(name);
    } catch (err) {
      setMsg(err instanceof Error ? err.message : "Pick a person first");
      return;
    }
    setMsg(null);
    setPhase("countdown");
    let pre = 3;
    setCountdown(pre);
    tickRef.current = window.setInterval(() => {
      pre -= 1;
      if (pre > 0) {
        setCountdown(pre);
        return;
      }
      if (tickRef.current) clearInterval(tickRef.current);
      setPhase("recording");
      let remaining = VOICE_DURATION_SEC;
      setCountdown(remaining);
      tickRef.current = window.setInterval(() => {
        remaining -= 1;
        if (remaining <= 0) {
          if (tickRef.current) clearInterval(tickRef.current);
          setPhase("processing");
          setCountdown(0);
        } else {
          setCountdown(remaining);
        }
      }, 1000);
      fetch(hwUrl("/speaker/record-enroll"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: target, duration_sec: VOICE_DURATION_SEC }),
      })
        .then((r) => r.json().then((data) => ({ ok: r.ok, data })))
        .then(({ ok, data }) => {
          if (tickRef.current) clearInterval(tickRef.current);
          setPhase("idle");
          setCountdown(0);
          if (ok && data.status === "ok") {
            setMsg(`Saved a voice sample for ${target}.`);
            onDone();
          } else {
            setMsg(data.detail ?? data.message ?? "Couldn't record.");
          }
        })
        .catch((e) => {
          if (tickRef.current) clearInterval(tickRef.current);
          setPhase("idle");
          setCountdown(0);
          setMsg(e instanceof Error ? e.message : "Couldn't record.");
        });
    }, 1000);
  }

  return (
    <div className="lm-mon-card" style={{ marginTop: 12 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 10 }}>
        <div style={{ fontSize: 13, fontWeight: 650, textTransform: "capitalize" }}>
          Record a voice for {name}
        </div>
        <button type="button" className="lm-u-btn" onClick={onClose} aria-label="Close" style={{ width: 28, height: 28, padding: 0 }}>
          <X size={14} />
        </button>
      </div>
      <p style={{ fontSize: 12.5, color: "var(--lm-text-dim)", margin: "0 0 10px" }}>
        Read these aloud when the robot is listening. Recording uses the robot mic.
      </p>
      <ol style={{ margin: "0 0 12px", paddingLeft: 18, fontSize: 13, lineHeight: 1.5 }}>
        {phrases.map((p) => <li key={p}>{p}</li>)}
      </ol>
      {msg && <div className={/couldn|error|pick/i.test(msg) ? "lm-guide-err" : "lm-guide-ok"}>{msg}</div>}
      <button
        type="button"
        className="lm-guide-primary"
        disabled={phase !== "idle" || !name.trim()}
        onClick={start}
        style={{ marginTop: 8 }}
      >
        {phase === "idle" && <><Mic size={15} /> Start recording ({VOICE_DURATION_SEC}s)</>}
        {phase === "countdown" && `Get ready… ${countdown}`}
        {phase === "recording" && `Recording — ${countdown}s`}
        {phase === "processing" && <><Loader2 size={15} className="lm-spin-ico" /> Processing…</>}
      </button>
    </div>
  );
}
