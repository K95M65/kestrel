// One-turn chat for guided setup. Same path Talk uses (POST /api/sensing/event
// + /api/agent/events), trimmed to a promise so the guide isn't coupled to
// ChatSection's conversation store.

import { stripChatMarkers } from "./stripChatMarkers";

type Loose = Record<string, unknown>;

function runIdOf(ev: Loose): string | undefined {
  const d = (ev.detail ?? null) as Loose | null;
  const nested = (d?.data ?? null) as Loose | null;
  for (const v of [ev.runId, d?.run_id, d?.runId, nested?.run_id]) {
    if (typeof v === "string" && v) return v;
  }
  return undefined;
}

export async function sendGuideChat(
  message: string,
  opts?: { timeoutMs?: number; onDelta?: (text: string) => void },
): Promise<string> {
  const text = message.trim();
  if (!text) throw new Error("Type something first.");

  const res = await fetch("/api/sensing/event", {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ type: "web_chat", message: text }),
  });
  const json = (await res.json().catch(() => ({}))) as {
    status?: number;
    message?: string;
    data?: { runId?: string; handler?: string; response?: string };
  };

  if (json.data?.handler === "local") {
    return String(json.data.response || "✓");
  }
  if (json.data?.handler === "dropped" || json.data?.handler === "queued") {
    throw new Error("The robot is busy. Try again in a moment.");
  }
  const runId = json.data?.runId;
  if (json.status !== 1 || !runId) {
    throw new Error(json.message || "The robot didn't take that.");
  }
  return waitForRun(runId, opts?.timeoutMs ?? 45_000, opts?.onDelta);
}

function waitForRun(
  runId: string,
  timeoutMs: number,
  onDelta?: (text: string) => void,
): Promise<string> {
  return new Promise((resolve, reject) => {
    const es = new EventSource("/api/agent/events", { withCredentials: true });
    let buf = "";
    let settled = false;

    const finish = (text: string, err?: Error) => {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      es.close();
      if (err) reject(err);
      else resolve(stripChatMarkers(text || "…"));
    };

    const timer = window.setTimeout(() => {
      finish(buf, buf ? undefined : new Error("No reply yet. You can skip this try."));
    }, timeoutMs);

    es.onmessage = (msg) => {
      let ev: Loose;
      try {
        ev = JSON.parse(msg.data) as Loose;
      } catch {
        return;
      }
      if (runIdOf(ev) !== runId) return;
      const type = String(ev.type ?? "");
      const state = String(ev.state ?? "");
      const d = (ev.detail ?? null) as Loose | null;
      const nested = (d?.data ?? null) as Loose | null;

      if (type === "assistant_delta") {
        const delta = String(ev.summary ?? "");
        if (delta) {
          buf += delta;
          onDelta?.(stripChatMarkers(buf));
        }
        return;
      }

      if (type === "chat_response") {
        const chatMsg = String(d?.message ?? ev.summary ?? "");
        if (chatMsg === "[no reply]") {
          finish(buf || "…");
          return;
        }
        if (state === "error") {
          finish("", new Error(String((d as Loose | null)?.error ?? ev.summary ?? "error")));
          return;
        }
        if (state === "complete" || state === "final") {
          finish(chatMsg || buf);
          return;
        }
        if (chatMsg) {
          buf = chatMsg;
          onDelta?.(stripChatMarkers(buf));
        }
        return;
      }

      if (type === "flow_event") {
        const node = String(d?.node ?? "");
        if (node === "tts_send" || node === "tts_suppressed") {
          const full = String(nested?.full_text ?? d?.full_text ?? nested?.text ?? d?.text ?? "");
          if (full) finish(full);
          return;
        }
        if (node === "no_reply") finish(buf || "…");
      }
    };

    es.onerror = () => {
      // EventSource reconnects; don't fail the turn on a blip.
    };
  });
}
