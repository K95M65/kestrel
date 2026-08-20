import { useEffect, useRef, useState } from "react";
import { hwUrl, fileToBase64 } from "@/lib/api";
import { speakGuide } from "@/lib/guideSpeak";
import { wakeAndPoseForPhoto, releasePhotoPose } from "@/lib/preparePhoto";
import { HW } from "@/pages/monitor/types";
import { INPUT_STYLE, LABEL_STYLE } from "@/components/setup/shared";

/** Live MJPEG + countdown snap + enroll. Used by guided setup and People. */
export function LiveFaceEnroll({
  robotName, speak = false, fixedLabel, onTried, onEnrolled, lead, saveLabel,
}: {
  robotName?: string;
  speak?: boolean;
  /** When set, skip the name field and enroll under this contact. */
  fixedLabel?: string;
  onTried?: () => void;
  onEnrolled?: (name: string) => void;
  lead?: string;
  saveLabel?: string;
}) {
  const [streamEpoch, setStreamEpoch] = useState(() => Date.now());
  const [camOff, setCamOff] = useState(false);
  const [streamError, setStreamError] = useState(false);
  const [count, setCount] = useState<number | null>(null);
  const [stillUrl, setStillUrl] = useState<string | null>(null);
  const stillBlob = useRef<Blob | null>(null);
  const [label, setLabel] = useState(fixedLabel ?? "");
  const [busy, setBusy] = useState<string | null>(null);
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const cued = useRef(false);
  const fileRef = useRef<HTMLInputElement | null>(null);
  const posedRef = useRef(false);

  useEffect(() => {
    return () => { if (stillUrl) URL.revokeObjectURL(stillUrl); };
  }, [stillUrl]);

  useEffect(() => {
    return () => { if (posedRef.current) void releasePhotoPose(); };
  }, []);

  async function refreshCam() {
    try {
      const r = await fetch(`${HW}/camera`).then((x) => x.json());
      setCamOff(!!r.disabled);
      return !!r.disabled;
    } catch {
      setCamOff(true);
      return true;
    }
  }

  useEffect(() => {
    void (async () => {
      const off = await refreshCam();
      if (cued.current || !speak) return;
      cued.current = true;
      void speakGuide(off ? "Turn my camera on so I can see you." : "Look at my camera so I can see you.");
    })();
  }, [speak]);

  async function turnOn() {
    setBusy("cam");
    try {
      await fetch(`${HW}/camera/enable`, { method: "POST" });
      setCamOff(false);
      setStreamError(false);
      setStreamEpoch(Date.now());
      onTried?.();
      if (speak) void speakGuide("Look at my camera so I can see you.");
    } catch {
      setMsg({ ok: false, text: "Couldn't turn the camera on." });
    } finally {
      setBusy(null);
    }
  }

  function clearStill() {
    if (stillUrl) URL.revokeObjectURL(stillUrl);
    setStillUrl(null);
    stillBlob.current = null;
  }

  async function captureStill(): Promise<boolean> {
    try {
      const res = await fetch(hwUrl(`/camera/snapshot?t=${Date.now()}`), {
        cache: "no-store",
        headers: { "Cache-Control": "no-cache", Pragma: "no-cache" },
      });
      if (!res.ok) throw new Error("Couldn't take a photo.");
      const blob = await res.blob();
      if (blob.size < 400) throw new Error("That photo was empty. Try again in the light.");
      clearStill();
      stillBlob.current = blob;
      setStillUrl(URL.createObjectURL(blob));
      onTried?.();
      return true;
    } catch (err) {
      setMsg({ ok: false, text: err instanceof Error ? err.message : "Couldn't take a photo." });
      return false;
    }
  }

  async function snap() {
    if (busy) return;
    setBusy("snap");
    setMsg(null);
    try {
      if (camOff) {
        try {
          await fetch(`${HW}/camera/enable`, { method: "POST" });
          setCamOff(false);
        } catch {
          /* wake may still turn it on */
        }
      }
      const woke = await wakeAndPoseForPhoto((line) => setMsg({ ok: true, text: line }));
      if (woke) {
        posedRef.current = true;
        setCamOff(false);
        setStreamError(false);
        setStreamEpoch(Date.now());
      }
      setMsg(null);
      if (speak) void speakGuide("Hold still.");
      for (const n of [3, 2, 1]) {
        setCount(n);
        await new Promise((r) => setTimeout(r, 700));
      }
      setCount(null);
      const ok = await captureStill();
      if (ok && speak) void speakGuide(fixedLabel ? "Got it." : "Got it. Who is this?");
    } finally {
      setBusy(null);
    }
  }

  function newPhoto() {
    clearStill();
    setStreamError(false);
    setStreamEpoch(Date.now());
    setMsg(null);
    if (fileRef.current) fileRef.current.value = "";
    if (speak) void speakGuide("Look at my camera.");
  }

  function pickFromPictures(file: File | undefined) {
    if (!file) return;
    if (!file.type.startsWith("image/")) {
      setMsg({ ok: false, text: "Pick a photo (JPEG or PNG)." });
      return;
    }
    clearStill();
    stillBlob.current = file;
    setStillUrl(URL.createObjectURL(file));
    setMsg(null);
    onTried?.();
  }

  async function enroll() {
    const name = (fixedLabel || label).trim();
    const blob = stillBlob.current;
    if (!name || !blob) return;
    setBusy("enroll");
    setMsg(null);
    try {
      const file = new File([blob], "enroll.jpg", { type: blob.type || "image/jpeg" });
      const b64 = await fileToBase64(file);
      const resp = await fetch(hwUrl("/face/enroll"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ label: name.toLowerCase(), image_base64: b64 }),
      });
      const raw = await resp.text();
      let data: { detail?: string; message?: string } = {};
      try { data = raw ? JSON.parse(raw) : {}; } catch { /* html */ }
      if (!resp.ok) throw new Error(data.detail || data.message || "Enroll failed.");
      setMsg({ ok: true, text: `Saved ${name}.` });
      onEnrolled?.(name);
      onTried?.();
      if (speak) void speakGuide(`Nice to meet you, ${name}.`);
    } catch (err) {
      setMsg({ ok: false, text: err instanceof Error ? err.message : "Couldn't save that face." });
    } finally {
      setBusy(null);
    }
  }

  const streamSrc = hwUrl(`/camera/stream?e=${streamEpoch}`);
  const reviewing = !!stillUrl;
  const who = robotName?.trim() || "the robot";
  const needName = !fixedLabel;

  return (
    <>
      {lead !== "" && (
        <p className="lm-guide-lead">
          {lead ?? `Stand in front of ${who}, take a photo, then say who this is.`}
        </p>
      )}
      <div className="lm-guide-cam-wrap">
        {camOff || (streamError && !reviewing) ? (
          <div className="lm-guide-cam-empty">
            {camOff ? "The camera is off." : "Can't show the camera right now."}
          </div>
        ) : reviewing ? (
          <img key={stillUrl} className="lm-guide-cam" src={stillUrl} alt="Captured photo" />
        ) : (
          <img
            key={streamEpoch}
            className="lm-guide-cam"
            src={streamSrc}
            alt="Live camera"
            onError={() => setStreamError(true)}
            onLoad={() => { setStreamError(false); onTried?.(); }}
          />
        )}
        {count !== null && <div className="lm-guide-count" aria-live="polite">{count}</div>}
      </div>
      <div className="lm-guide-seen">
        {msg?.ok
          ? msg.text
          : reviewing
            ? (needName ? "That's the photo. Type who this is, or retake." : "That's the photo. Save it, or retake.")
            : "Live view — take a photo, or choose one you already have."}
      </div>
      <input
        ref={fileRef}
        type="file"
        accept="image/*"
        hidden
        onChange={(e) => pickFromPictures(e.target.files?.[0])}
      />
      <div style={{ display: "flex", gap: 8, marginBottom: 12, flexWrap: "wrap" }}>
        {camOff && (
          <button type="button" className="lm-guide-primary" disabled={busy === "cam"} onClick={() => void turnOn()}>
            {busy === "cam" ? "…" : "Turn camera on"}
          </button>
        )}
        {!reviewing && (
          <>
            <button type="button" className="lm-guide-primary" disabled={!!busy} onClick={() => void snap()}>
              {busy === "snap" ? "…" : "Take photo"}
            </button>
            <button type="button" className="lm-guide-ghost" disabled={!!busy} onClick={() => fileRef.current?.click()}>
              Choose from pictures
            </button>
          </>
        )}
        {reviewing && (
          <button type="button" className="lm-guide-ghost" disabled={!!busy} onClick={newPhoto}>
            Retake
          </button>
        )}
      </div>
      {reviewing && needName && (
        <>
          <label htmlFor="live-face-name" style={LABEL_STYLE}>This is</label>
          <input
            id="live-face-name"
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            placeholder="their name"
            autoComplete="off"
            style={{ ...INPUT_STYLE, marginBottom: 8 }}
          />
        </>
      )}
      {reviewing && (
        <button
          type="button"
          className="lm-guide-primary"
          disabled={!!busy || !(fixedLabel || label).trim()}
          onClick={() => void enroll()}
        >
          {busy === "enroll" ? "Saving…" : (saveLabel ?? "Remember this face")}
        </button>
      )}
      {msg && !msg.ok && <div className="lm-guide-err">{msg.text}</div>}
    </>
  );
}
