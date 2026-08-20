import { useEffect, useState } from "react";
import { sendGuideChat } from "@/lib/guideChat";
import { speakGuide } from "@/lib/guideSpeak";
import { talkName } from "@/lib/robotName";
import { INPUT_STYLE } from "@/components/setup/shared";

export function GuideTalkStep({
  robotName, onTried, lead, prompt, greet = true, timeoutMs,
}: {
  robotName: string;
  onTried: () => void;
  lead?: string;
  /** Prefill the compose box. */
  prompt?: string;
  /** Speak a hello on enter. Off for later try-steps that already have a prompt. */
  greet?: boolean;
  timeoutMs?: number;
}) {
  const who = talkName(robotName);
  const greeting = who === "the robot" ? "Hi, I'm here." : `Hi, I'm ${who}.`;
  const [input, setInput] = useState(prompt ?? `Hi, what's your name?`);
  const [userLine, setUserLine] = useState("");
  const [reply, setReply] = useState(greet ? greeting : "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [muted, setMuted] = useState(false);

  useEffect(() => {
    if (!greet) {
      setReply("");
      return;
    }
    let cancelled = false;
    setReply(greeting);
    void speakGuide(greeting).then((r) => {
      if (!cancelled && r === "muted") setMuted(true);
    });
    return () => { cancelled = true; };
  }, [greeting, greet]);

  async function send() {
    const text = input.trim();
    if (!text || busy) return;
    setBusy(true);
    setError(null);
    setUserLine(text);
    setReply("");
    try {
      const out = await sendGuideChat(text, { onDelta: setReply, timeoutMs });
      setReply(out);
      onTried();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Couldn't reach the robot.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <p className="lm-guide-lead">
        {lead || (who === "the robot"
          ? "The robot will say hello. Reply here if you don't hear it."
          : `Say hi to ${who}. If the speaker is muted, type here instead.`)}
      </p>
      {muted && (
        <div className="lm-guide-err" style={{ marginTop: 0, marginBottom: 10 }}>
          Speaker is muted — the reply will show here, not out loud. Unmute on Home to hear it.
        </div>
      )}
      <div className="lm-guide-chat">
        {!userLine && reply && (
          <div className="lm-guide-bubble lm-guide-bubble--bot">{reply}</div>
        )}
        {userLine && (
          <div className="lm-guide-bubble lm-guide-bubble--you">{userLine}</div>
        )}
        {userLine && (busy || reply) && (
          <div className={"lm-guide-bubble lm-guide-bubble--bot" + (busy && !reply ? " lm-guide-bubble--wait" : "")}>
            {reply || "Listening…"}
          </div>
        )}
        {error && <div className="lm-guide-err">{error}</div>}
      </div>
      <textarea
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter" && !e.shiftKey) {
            e.preventDefault();
            void send();
          }
        }}
        rows={2}
        placeholder={`Message ${who}…`}
        disabled={busy}
        style={{ ...INPUT_STYLE, resize: "none", minHeight: 56 }}
      />
      <button
        type="button"
        className="lm-guide-primary"
        style={{ marginTop: 8 }}
        disabled={busy || !input.trim()}
        onClick={() => void send()}
      >
        {busy ? "Sending…" : "Send"}
      </button>
    </>
  );
}
