import { useCallback, useState } from "react";
import { HW } from "../types";
import { usePolling } from "../../../hooks/usePolling";
import type { StrangersData, FaceStrangerStat } from "./types";

// Unknown voice clusters (/voice/strangers) + face stranger visit stats
// (/face/stranger-stats), with their own polling and delete flows. Independent
// of the enrolled-owners data, so it lives in its own hook.
export function useStrangers() {
  const [strangers, setStrangers] = useState<StrangersData | null>(null);
  const [strangersError, setStrangersError] = useState(false);
  const [expandedCluster, setExpandedCluster] = useState<Record<string, boolean>>({});
  const [deletingCluster, setDeletingCluster] = useState<string | null>(null);
  const [deletingStrangerFile, setDeletingStrangerFile] = useState<string | null>(null); // "hash/filename"

  // Face stranger visit stats. The device tracks each unrecognized face's visit
  // count and surfaces a familiar-stranger enroll prompt to the agent when count
  // crosses FAMILIAR_VISIT_THRESHOLD.
  const [faceStrangers, setFaceStrangers] = useState<FaceStrangerStat[] | null>(null);
  const [faceStrangersError, setFaceStrangersError] = useState(false);

  // Pending themed-confirm targets (null = no dialog open).
  const [confirmCluster, setConfirmCluster] = useState<{ hash: string; sampleCount: number } | null>(null);
  const [confirmStrangerFile, setConfirmStrangerFile] = useState<{ hash: string; filename: string } | null>(null);
  const [confirmForgetFace, setConfirmForgetFace] = useState<string | null>(null);
  const [claimingFace, setClaimingFace] = useState<string | null>(null);
  const [claimingVoice, setClaimingVoice] = useState<string | null>(null);
  const [forgettingFace, setForgettingFace] = useState<string | null>(null);
  const [claimError, setClaimError] = useState("");
  const [claimName, setClaimName] = useState("");
  const [claimTarget, setClaimTarget] = useState<
    { kind: "face"; id: string } | { kind: "voice"; id: string } | null
  >(null);

  const refreshStrangers = useCallback(async (signal?: AbortSignal) => {
    try {
      const res = await fetch(`${HW}/voice/strangers`, { signal });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const j = await res.json();
      setStrangers({
        total: j.total ?? 0,
        clusters: Array.isArray(j.clusters) ? j.clusters : [],
      });
      setStrangersError(false);
    } catch (e) {
      if ((e as Error).name === "AbortError") return;
      setStrangersError(true);
    }
  }, []);

  usePolling(async (signal) => { await refreshStrangers(signal); }, 15_000);

  const refreshFaceStrangers = useCallback(async (signal?: AbortSignal) => {
    try {
      const res = await fetch(`${HW}/face/stranger-stats`, { signal });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const j = (await res.json()) as Record<string, { count?: number; first_seen?: string; last_seen?: string }>;
      const rows: FaceStrangerStat[] = Object.entries(j ?? {}).map(([sid, v]) => ({
        stranger_id: sid,
        count: v?.count ?? 0,
        first_seen: v?.first_seen ?? "",
        last_seen: v?.last_seen ?? "",
      }));
      // Newest activity first.
      rows.sort((a, b) => Date.parse(b.last_seen || "") - Date.parse(a.last_seen || ""));
      setFaceStrangers(rows);
      setFaceStrangersError(false);
    } catch (e) {
      if ((e as Error).name === "AbortError") return;
      setFaceStrangersError(true);
    }
  }, []);

  usePolling(async (signal) => { await refreshFaceStrangers(signal); }, 15_000);

  // Open the themed confirm dialog for deleting a stranger voice cluster.
  const handleDeleteCluster = (hash: string, sampleCount: number) => setConfirmCluster({ hash, sampleCount });

  const confirmDeleteCluster = async () => {
    if (!confirmCluster) return;
    const { hash } = confirmCluster;
    setConfirmCluster(null);
    setDeletingCluster(hash);
    try {
      const res = await fetch(`${HW}/voice/strangers/${hash}`, { method: "DELETE" });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ detail: `HTTP ${res.status}` }));
        alert(`Delete failed: ${err.detail ?? res.status}`);
      }
      await refreshStrangers();
    } catch (e) {
      alert(`Delete failed: ${(e as Error).message}`);
    } finally {
      setDeletingCluster(null);
    }
  };

  // Open the themed confirm dialog for deleting a single stranger sample file.
  const handleDeleteStrangerFile = (hash: string, filename: string) => setConfirmStrangerFile({ hash, filename });

  const confirmDeleteStrangerFile = async () => {
    if (!confirmStrangerFile) return;
    const { hash, filename } = confirmStrangerFile;
    setConfirmStrangerFile(null);
    const key = `${hash}/${filename}`;
    setDeletingStrangerFile(key);
    try {
      const res = await fetch(`${HW}/voice/strangers/${hash}/${encodeURIComponent(filename)}`, { method: "DELETE" });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ detail: `HTTP ${res.status}` }));
        alert(`Delete failed: ${err.detail ?? res.status}`);
      }
      await refreshStrangers();
    } catch (e) {
      alert(`Delete failed: ${(e as Error).message}`);
    } finally {
      setDeletingStrangerFile(null);
    }
  };

  const openClaimFace = (strangerId: string) => {
    setClaimTarget({ kind: "face", id: strangerId });
    setClaimName("");
    setClaimError("");
  };
  const openClaimVoice = (hash: string) => {
    setClaimTarget({ kind: "voice", id: hash });
    setClaimName("");
    setClaimError("");
  };
  const closeClaim = () => {
    setClaimTarget(null);
    setClaimError("");
    setClaimingFace(null);
    setClaimingVoice(null);
  };

  const submitClaim = async () => {
    if (!claimTarget) return;
    const label = claimName.trim().toLowerCase()
      .replace(/[^a-z0-9_-]+/g, "_").replace(/^_+|_+$/g, "").slice(0, 64);
    if (!label || label === "unknown") {
      setClaimError("Pick a real name.");
      return;
    }
    if (claimTarget.kind === "face") {
      setClaimingFace(claimTarget.id);
      setClaimError("");
      try {
        const res = await fetch(`${HW}/face/stranger/claim`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ stranger_id: claimTarget.id, label }),
        });
        if (!res.ok) {
          const err = await res.json().catch(() => ({ detail: `HTTP ${res.status}` }));
          throw new Error(typeof err.detail === "string" ? err.detail : "Could not save this face.");
        }
        setClaimTarget(null);
        await refreshFaceStrangers();
        return true;
      } catch (e) {
        setClaimError((e as Error).message);
        return false;
      } finally {
        setClaimingFace(null);
      }
    }
    setClaimingVoice(claimTarget.id);
    setClaimError("");
    try {
      const res = await fetch(`${HW}/voice/strangers/${encodeURIComponent(claimTarget.id)}/claim`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: label }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ detail: `HTTP ${res.status}` }));
        throw new Error(typeof err.detail === "string" ? err.detail : "Could not save this voice.");
      }
      setClaimTarget(null);
      await refreshStrangers();
      return true;
    } catch (e) {
      setClaimError((e as Error).message);
      return false;
    } finally {
      setClaimingVoice(null);
    }
  };

  const handleForgetFace = (strangerId: string) => setConfirmForgetFace(strangerId);

  const confirmForgetFaceNow = async () => {
    const id = confirmForgetFace;
    if (!id) return;
    setConfirmForgetFace(null);
    setForgettingFace(id);
    try {
      const res = await fetch(`${HW}/face/stranger/${encodeURIComponent(id)}`, { method: "DELETE" });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ detail: `HTTP ${res.status}` }));
        alert(`Could not forget this face: ${err.detail ?? res.status}`);
      }
      await refreshFaceStrangers();
    } catch (e) {
      alert(`Could not forget this face: ${(e as Error).message}`);
    } finally {
      setForgettingFace(null);
    }
  };

  return {
    strangers, strangersError,
    expandedCluster, setExpandedCluster,
    deletingCluster, deletingStrangerFile,
    faceStrangers, faceStrangersError,
    confirmCluster, setConfirmCluster,
    confirmStrangerFile, setConfirmStrangerFile,
    confirmForgetFace, setConfirmForgetFace,
    handleDeleteCluster, confirmDeleteCluster,
    handleDeleteStrangerFile, confirmDeleteStrangerFile,
    handleForgetFace, confirmForgetFaceNow,
    claimTarget, claimName, setClaimName, claimError, closeClaim,
    openClaimFace, openClaimVoice, submitClaim,
    claimingFace, claimingVoice, forgettingFace,
    refreshFaceStrangers, refreshStrangers,
  };
}
