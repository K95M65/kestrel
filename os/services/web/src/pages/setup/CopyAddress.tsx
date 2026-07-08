import { useState } from "react";
import { C } from "@/components/setup/shared";

// CopyAddress — a device URL with a one-tap Copy button. Shown on the
// post-submit screen so the operator can capture the address BEFORE they
// switch Wi-Fi networks (at which point this page loses its connection and any
// un-copied address is gone). Pass the full URL via `url` — callers prefer the
// raw-IP address (works on every LAN) over the .local name (fails when the
// router blocks mDNS).
//
// Clipboard: the Setup page is served over plain HTTP (http://192.168.100.1),
// where `navigator.clipboard` is undefined (it requires a secure context), so
// the modern API silently no-ops. Fall back to a hidden-textarea +
// document.execCommand("copy"), which works on http:// origins, so the button
// actually copies instead of doing nothing.
export function CopyAddress({ url }: { url: string }) {
  const [copied, setCopied] = useState(false);
  const text = url;
  const flashCopied = () => {
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1800);
  };
  const legacyCopy = () => {
    try {
      const ta = document.createElement("textarea");
      ta.value = text;
      // Keep it off-screen and non-disruptive to scroll/focus.
      ta.style.position = "fixed";
      ta.style.top = "-9999px";
      ta.setAttribute("readonly", "");
      document.body.appendChild(ta);
      ta.select();
      const ok = document.execCommand("copy");
      document.body.removeChild(ta);
      if (ok) flashCopied();
    } catch {
      /* copy unsupported — user can still select the text manually */
    }
  };
  const copy = () => {
    if (navigator.clipboard?.writeText) {
      navigator.clipboard.writeText(text).then(flashCopied, legacyCopy);
    } else {
      legacyCopy();
    }
  };
  return (
    <div style={{
      display: "flex", alignItems: "center", gap: 8,
      background: C.surface, border: `1px solid ${C.border}`,
      borderRadius: 8, padding: "8px 10px",
    }}>
      <span style={{
        flex: 1, textAlign: "left", fontSize: 13.5, color: C.text,
        fontFamily: "ui-monospace, monospace", overflow: "hidden", textOverflow: "ellipsis",
      }}>
        {text}
      </span>
      <button
        type="button"
        onClick={copy}
        className={`lm-btn lm-btn-ghost${copied ? " lm-copied" : ""}`}
        style={{ padding: "5px 10px", fontSize: 11, flexShrink: 0, transition: "background 0.15s, color 0.15s, border-color 0.15s", minWidth: 64 }}
      >
        {copied ? "Copied ✓" : "Copy"}
      </button>
    </div>
  );
}
