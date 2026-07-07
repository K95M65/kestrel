#!/usr/bin/env python3
"""os-server-observer — PicoClaw process hook forwarding channel (Telegram) turns
to os-server so they appear in the device Flow Monitor and drive [HW:/…] markers.

Mirror of the Hermes os-server-observer hook (internal/hermes/hooks/…), adapted to
PicoClaw's process-hook wire protocol: a long-lived subprocess speaking NDJSON
JSON-RPC over stdio — one JSON object per line on stdin; responses on stdout.

Registered via config.json hooks.processes.os-server-observer (internal/picoclaw/
hooks.go), with observe=[turn_start,turn_end] + intercept=[after_llm].

Two message shapes arrive on stdin:
  * observe NOTIFICATION (no "id"): params is the events.Event envelope
    {kind, scope{channel,chat_id,sender_id,session_key,turn_id,…}, payload, …}.
    Fire-and-forget — no response expected.
  * intercept REQUEST (has "id"): after_llm — params is the LLMHookRequest
    {messages:[{role,content}], context{…}, …}. We are in the turn's CRITICAL
    PATH (interceptor_timeout_ms, default 5000), so we ALWAYS respond immediately
    with {"action":"continue"} — never block, never mutate — then use `messages`
    purely to capture the conversation text.

Only turns whose channel is in FORWARD_CHANNELS (default: telegram) are forwarded.
Device-local turns (channel "pico", already logged by os-server's own sendChat) are
ignored to avoid double-counting — the analogue of the Hermes hook's skipPlatform.

__OS_SERVER_TURN_URL__ is substituted with the real loopback URL at materialize
time by ensureObserverHook (internal/picoclaw/hooks.go).

Besides POSTing each forwarded turn to os-server, every forwarded payload is also
appended as one JSON line (JSONL) to OBSERVER_LOG (default
/root/.picoclaw/logs/messages_hooks.log) — a local, os-server-independent audit
trail of the Telegram↔PicoClaw turns this hook saw.

Set OBSERVER_DEBUG=1 to dump every raw stdin line to stderr — PicoClaw surfaces
subprocess stderr in its gateway log. Because each kind's `payload` is `any`, use
that dump to VERIFY the exact field names on-device, then tighten the extractors.
"""

import datetime
import json
import os
import sys
import threading
import urllib.request

OS_SERVER_TURN_URL = "__OS_SERVER_TURN_URL__"
DEBUG = os.environ.get("OBSERVER_DEBUG") == "1"
HOOK_LOG = os.environ.get("OBSERVER_LOG", "/root/.picoclaw/logs/messages_hooks.log")
FORWARD_CHANNELS = {
    c.strip().lower()
    for c in os.environ.get("OBSERVER_CHANNELS", "telegram").split(",")
    if c.strip()
}

# Per-turn buffer keyed by turn id: {"scope":{}, "user":str, "assistant":str, "started":bool}.
# after_llm (fires between turn_start and turn_end) fills user/assistant text.
_turns = {}


def _debug(*a):
    if DEBUG:
        print("[os-server-observer]", *a, file=sys.stderr, flush=True)


def _log_json(record):
    """Append one JSON line to HOOK_LOG. Best-effort + independent of the POST, so
    the local audit trail survives even when os-server is down."""
    try:
        os.makedirs(os.path.dirname(HOOK_LOG), exist_ok=True)
        with open(HOOK_LOG, "a", encoding="utf-8") as f:
            f.write(json.dumps(record, ensure_ascii=False) + "\n")
    except Exception as e:  # noqa: BLE001
        _debug("log write failed:", e)


def _send(payload):
    try:
        req = urllib.request.Request(
            OS_SERVER_TURN_URL,
            data=json.dumps(payload).encode("utf-8"),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        urllib.request.urlopen(req, timeout=3).close()
    except Exception as e:  # noqa: BLE001 — best-effort; os-server down must not affect the turn
        _debug("post failed:", e)


def _post(event, ctx):
    payload = {"event": event, "context": ctx}
    # Local audit copy first (synchronous, fast, keeps event order) so it is written
    # regardless of the POST outcome.
    _log_json({"ts": datetime.datetime.now(datetime.timezone.utc).isoformat(), **payload})
    # Fire the network POST OFF the stdin loop: a slow/hung os-server must never delay
    # our reply to an after_llm intercept (interceptor_timeout_ms=5000) nor stall the
    # next event (observer_timeout_ms is only 500ms). Best-effort daemon thread.
    threading.Thread(target=_send, args=(payload,), daemon=True).start()


def _text(v):
    """Best-effort text out of str | {text|content|message|final} | list-of-blocks."""
    if v is None:
        return ""
    if isinstance(v, str):
        return v
    if isinstance(v, dict):
        for k in ("text", "content", "message", "final", "output"):
            if k in v:
                t = _text(v[k])
                if t:
                    return t
        return ""
    if isinstance(v, list):
        return " ".join(t for t in (_text(it) for it in v) if t).strip()
    return ""


def _last_role(messages, role):
    if not isinstance(messages, list):
        return ""
    for m in reversed(messages):
        if isinstance(m, dict) and m.get("role") == role:
            return _text(m.get("content"))
    return ""


def _scope_of(params):
    """The observe envelope carries `scope`; the intercept request carries `context`
    (TurnContext). Return whichever is present, else params itself as a last resort."""
    if not isinstance(params, dict):
        return {}
    for k in ("scope", "context"):
        s = params.get(k)
        if isinstance(s, dict):
            return s
    return params


def _channel(scope):
    return str(scope.get("channel") or scope.get("platform") or "").lower()


def _turn_key(scope):
    return (
        scope.get("turn_id")
        or scope.get("session_key")
        or scope.get("session_id")
        or "default"
    )


def _ctx(scope, message="", response=""):
    """Map PicoClaw scope → the ChannelTurn payload context (handler_channel_turn.go)."""
    return {
        "platform": _channel(scope) or "telegram",
        "user_id": str(scope.get("sender_id") or scope.get("user_id") or ""),
        "chat_id": str(scope.get("chat_id") or ""),
        "session_id": str(scope.get("session_key") or scope.get("session_id") or ""),
        "message": message,
        "response": response,
    }


def _kind(msg, params):
    k = ""
    if isinstance(params, dict):
        k = params.get("kind") or params.get("event") or ""
    return str(k or msg.get("method") or "").lower()


def handle(msg):
    params = msg.get("params") if isinstance(msg, dict) else None
    is_request = isinstance(msg, dict) and msg.get("id") is not None

    # CRITICAL PATH: answer intercept requests immediately, before any other work,
    # so a slow/unreachable os-server can never stall the turn.
    if is_request:
        sys.stdout.write(
            json.dumps({"jsonrpc": "2.0", "id": msg["id"], "result": {"action": "continue"}})
            + "\n"
        )
        sys.stdout.flush()

    scope = _scope_of(params)
    channel = _channel(scope)
    kind = _kind(msg, params)

    # Only forward configured channels; skip device-local ("pico") turns entirely.
    if channel and channel not in FORWARD_CHANNELS:
        return
    if not channel and not is_request:
        return

    key = _turn_key(scope)
    buf = _turns.setdefault(key, {"scope": {}, "user": "", "assistant": "", "started": False})
    buf["scope"].update({k: v for k, v in scope.items() if v})

    # after_llm (intercept): the full message list carries user + assistant text.
    if is_request or "llm" in kind:
        messages = params.get("messages") if isinstance(params, dict) else None
        u = _last_role(messages, "user")
        a = _last_role(messages, "assistant")
        if u:
            buf["user"] = u
        if a:
            buf["assistant"] = a
        if not buf["assistant"] and isinstance(params, dict):
            buf["assistant"] = _text(params.get("response") or params.get("result"))
        return

    if kind.endswith("turn.start") or kind == "turn_start":
        payload = params.get("payload") if isinstance(params, dict) else None
        _post("agent:start", _ctx(buf["scope"], message=buf["user"] or _text(payload)))
        buf["started"] = True
        return

    if kind.endswith("turn.end") or kind == "turn_end":
        if not buf["started"]:
            # start missed (e.g. only after_llm seen) — synthesise so start/end pair up.
            _post("agent:start", _ctx(buf["scope"], message=buf["user"]))
        payload = params.get("payload") if isinstance(params, dict) else None
        _post("agent:end", _ctx(buf["scope"], response=buf["assistant"] or _text(payload)))
        _turns.pop(key, None)
        return


def main():
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        _debug("recv:", line[:2000])
        try:
            msg = json.loads(line)
        except Exception as e:  # noqa: BLE001
            _debug("json error:", e)
            continue
        try:
            handle(msg)
        except Exception as e:  # noqa: BLE001 — one bad event must not kill the hook
            _debug("handle error:", e)


if __name__ == "__main__":
    main()
