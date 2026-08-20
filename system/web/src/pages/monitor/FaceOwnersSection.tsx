import { useCallback, useEffect, useRef, useState, type CSSProperties } from "react";
import { Users, Mic, ScanFace, UserCheck, UserPlus, RefreshCw } from "lucide-react";
import { S } from "./styles";
import { useTheme } from "@/lib/useTheme";
import { getBehaviors, getDeviceConfig, hwUrl, setMe } from "@/lib/api";
import { talkName } from "@/lib/robotName";
import { useFaceEnroll } from "@/hooks/setup/useFaceEnroll";
import { FriendPhotoModal } from "./face-owners/FriendPhotoModal";
import { mainFacePhoto } from "@/lib/facePhoto";
import { UserTimelineModal } from "./UserTimelineModal";
import { ContactVoiceBar } from "./face-owners/ContactVoiceBar";
import { useStrangers } from "./face-owners/useStrangers";
import { useFilePreview } from "./face-owners/useFilePreview";
import { useFaceData } from "./face-owners/useFaceData";
import { useOwnerActions } from "./face-owners/useOwnerActions";
import { HeroStat } from "./face-owners/HeroStat";
import { EmptyState } from "./face-owners/EmptyState";
import { ConfirmDialog } from "./face-owners/ConfirmDialog";
import { RenameModal } from "./face-owners/RenameModal";
import { EnrollModal } from "./face-owners/EnrollModal";
import { UnknownFacesCard } from "./face-owners/UnknownFacesCard";
import { ClaimUnknownModal } from "./face-owners/ClaimUnknownModal";
import { CooldownsCard } from "./face-owners/CooldownsCard";
import { StrangerClustersCard } from "./face-owners/StrangerClustersCard";
import { PersonCard } from "./face-owners/PersonCard";

export function FaceOwnersSection({
  isDebug = false,
}: {
  isDebug?: boolean;
}) {
  const [, , themeClass] = useTheme();
  const { loadFaceOwners } = useFaceEnroll();
  const [sttLanguage, setSttLanguage] = useState("en");
  const [agentName, setAgentName] = useState("");
  const [meLabel, setMeLabel] = useState("");
  const [settingMe, setSettingMe] = useState(false);
  const [claimAsMe, setClaimAsMe] = useState(false);
  const meLabelRef = useRef("");
  useEffect(() => { meLabelRef.current = meLabel; }, [meLabel]);

  const handleSetMe = useCallback(async (label: string) => {
    setSettingMe(true);
    try {
      const b = await setMe(label);
      setMeLabel((b.config?.me ?? "").toLowerCase());
    } catch {
      /* stay on previous */
    } finally {
      setSettingMe(false);
    }
  }, []);

  useEffect(() => { loadFaceOwners(); }, [loadFaceOwners]);
  useEffect(() => {
    getDeviceConfig()
      .then((c) => {
        setSttLanguage(c.stt_language || "en");
        setAgentName(c.agent_name ?? "");
      })
      .catch(() => {});
    getBehaviors()
      .then((b) => setMeLabel((b.config?.me ?? "").toLowerCase()))
      .catch(() => {});
  }, []);

  const [photoFor, setPhotoFor] = useState<string | null>(null);
  const [voiceFor, setVoiceFor] = useState<string | null>(null);

  // Enrolled-owners list + detection state (cooldowns, current user) + polling
  // and refresh live in their own hook. `refresh` reloads the list after a
  // mutation (enroll / rename / remove).
  const {
    data, error, currentUser,
    cooldowns, cdError, resetting, manualRefreshing,
    refresh, handleManualRefresh, handleResetCooldowns,
  } = useFaceData();

  // Owner-mutation flows (enroll / rename / remove user-photo-voice) + their
  // confirm/in-flight state live in their own hook; it takes `refresh` to reload
  // the list after a change.
  const {
    showEnroll, setShowEnroll,
    renaming, setRenaming,
    renameValue, setRenameValue,
    renameError, setRenameError,
    renameSaving,
    handleRename, submitRename,
    confirmDelete, setConfirmDelete,
    confirmPhoto, setConfirmPhoto,
    confirmVoice, setConfirmVoice,
    deleting, deletingPhoto,
    handleRemove, confirmRemove,
    handleRemovePhoto, confirmRemovePhoto,
    handleRemoveVoiceFile, confirmRemoveVoice,
  } = useOwnerActions(refresh, {
    onRenamed: (oldLabel, newLabel) => {
      if (oldLabel === meLabelRef.current) void handleSetMe(newLabel);
    },
    onRemoved: (label) => {
      if (label === meLabelRef.current) void handleSetMe("");
    },
  });

  // Timeline modal state
  const [timelineUser, setTimelineUser] = useState<string | null>(null);

  // Person card expand state — cards start collapsed so the grid stays dense.
  // Auto-expands the currently-active user the first time it appears.
  const [expandedPerson, setExpandedPerson] = useState<Record<string, boolean>>({});
  // Tracks which card is hovered so its action buttons fade in (cleaner UX
  // than a permanent row of icons cluttering every card).
  const [hoveredPerson, setHoveredPerson] = useState<string | null>(null);
  // Tracks the hovered photo thumbnail so only its delete button shows —
  // identified by "label/filename".
  const [hoveredPhoto, setHoveredPhoto] = useState<string | null>(null);

  // Unknown voice clusters + face stranger visit stats live in their own hook
  // (independent of the enrolled-owners data).
  const {
    strangers, strangersError,
    expandedCluster, setExpandedCluster,
    deletingCluster, deletingStrangerFile,
    faceStrangers, faceStrangersError,
    confirmCluster, setConfirmCluster,
    confirmStrangerFile, setConfirmStrangerFile,
    handleDeleteCluster, confirmDeleteCluster,
    handleDeleteStrangerFile, confirmDeleteStrangerFile,
    confirmForgetFace, setConfirmForgetFace,
    handleForgetFace, confirmForgetFaceNow,
    claimTarget, claimName, setClaimName, claimError, closeClaim,
    openClaimFace, openClaimVoice, submitClaim,
    claimingFace, claimingVoice, forgettingFace,
  } = useStrangers();

  useEffect(() => { if (claimTarget) setClaimAsMe(false); }, [claimTarget]);

  // Per-person file gallery: folder toggle, inline preview, audio playback, and
  // file-open routing live in their own hook.
  const {
    expanded, toggleDir,
    preview, setPreview, previewLoading,
    playingAudio,
    openFile,
  } = useFilePreview();


  // Base card style matching Overview/System: the `.lm-mon-card` class owns the
  // resting + hover box-shadow (and the gradient sheen / amber accent / glow), so
  // we strip the inline boxShadow from S.card to let the class's :hover win.
  const monCard = { ...S.card, boxShadow: undefined };

  // Sizing-only — visual surface/border/hover/focus comes from `.lm-u-input`.
  const inputStyle: CSSProperties = {
    fontSize: 12,
    padding: "8px 11px",
    borderRadius: 7,
    width: "100%",
  };

  // Small uppercase field label for the enroll form, so each input reads as a
  // labelled field rather than a bare placeholder box.
  const fieldLabel: CSSProperties = {
    display: "block", fontSize: 10, fontWeight: 700, letterSpacing: "0.06em",
    textTransform: "uppercase", color: "var(--lm-text-dim)", marginBottom: 5,
  };

  // Sizing-only — visual surface/border/hover/focus comes from `.lm-u-btn`.
  const btnStyle: CSSProperties = {
    fontSize: 10,
    padding: "4px 12px",
    borderRadius: 6,
    fontWeight: 600,
  };

  // Card header row — label on the left, badge/action on the right, matching the
  // Overview/System header pattern (no tinted strip, just spacing + alignment).
  const cardHeader: CSSProperties = {
    display: "flex", justifyContent: "space-between", alignItems: "center",
    marginBottom: 12,
  };

  // Square icon button — used for the per-person action row (Edit / Timeline /
  // Delete / Expand) so each is the same compact size regardless of label width.
  // Surface/border/hover come from `.lm-u-btn`.
  const iconBtnStyle: CSSProperties = {
    width: 26, height: 26,
    display: "inline-flex", alignItems: "center", justifyContent: "center",
    padding: 0, borderRadius: 5,
    color: "var(--lm-text-dim)",
    fontSize: 13,
    lineHeight: 1,
  };

  const allCooldownEntries = [
    ...(cooldowns?.owners ?? []),
    ...(cooldowns?.strangers ?? []),
  ];
  const hasActiveCooldowns = allCooldownEntries.some((e) => e.cooldown_remaining > 0);

  // "Here now" only names a concrete enrolled user; the "unknown" bucket means
  // someone is present but unrecognized, which reads better as a dash on the tile.
  const hereNow = currentUser && currentUser !== "unknown" ? currentUser : null;
  // First enrolled photo of the active user, so the Here-now tile can show a real
  // face avatar instead of the generic icon when we have one.
  const hereNowPhoto = hereNow
    ? mainFacePhoto(data?.persons.find((p) => p.label === hereNow)?.photos) ?? null
    : null;

  const household = (data?.persons.filter((p) => p.label !== "unknown") ?? [])
    .slice()
    .sort((a, b) => {
      if (a.label === meLabel) return -1;
      if (b.label === meLabel) return 1;
      return a.label.localeCompare(b.label);
    });
  const knownCount = household.length;
  const meHere = !!hereNow && hereNow === meLabel;

  return (
    <div className="lm-home">
      <div className="lm-home-stage">
      <div className="lm-mon-hero lm-home-hero">
        <div className="lm-home-copy" style={{ flex: "1 1 100%" }}>
          <div className="lm-home-kicker">{meHere ? "You're here" : hereNow ? "Someone's here" : "Household"}</div>
          <h1 className="lm-home-title">
            {meHere
              ? "You're here"
              : hereNow
                ? `${hereNow[0].toUpperCase()}${hereNow.slice(1)} is here`
                : `Who ${talkName(agentName)} knows`}
          </h1>
          <p className="lm-home-meta">
            {error
              ? "The robot can't see faces right now. You can still add someone from a photo."
              : knownCount > 0
                ? `${knownCount} ${knownCount === 1 ? "person" : "people"}. Mark yourself as Me so this robot knows which card is yours.`
                : `Add a friend so ${talkName(agentName)} can say hello by name.`}
          </p>
          <div className="lm-home-actions">
            <button type="button" className="lm-home-cta" onClick={() => setShowEnroll(true)}>
              <UserPlus size={14} /> Add a friend
            </button>
            <button
              type="button"
              className="lm-home-ghost"
              onClick={handleManualRefresh}
              disabled={manualRefreshing}
              aria-label="Refresh"
            >
              <RefreshCw size={14} className={manualRefreshing ? "lm-spin" : undefined} />
            </button>
          </div>
        </div>
      </div>
      </div>

      {isDebug && (
          <div className="lm-grid-auto">
            <HeroStat icon={<Users size={16} />} label="Enrolled" tone="amber"
              value={data ? data.enrolled_count : "—"} />
            <HeroStat
              icon={hereNowPhoto && hereNow ? (
                <img
                  src={hwUrl(`/face/photo/${encodeURIComponent(hereNow)}/${encodeURIComponent(hereNowPhoto)}`)}
                  alt=""
                  style={{ width: "100%", height: "100%", objectFit: "cover", borderRadius: "inherit" }}
                  onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = "none"; }}
                />
              ) : <UserCheck size={16} />}
              label="Here now" tone="teal" pulse={!!hereNow}
              value={<span style={{ textTransform: "capitalize" }}>{hereNow ?? "—"}</span>} />
            <HeroStat icon={<Mic size={16} />} label="Unknown voices" tone="purple"
              value={strangers ? strangers.total : "—"} />
            <HeroStat icon={<ScanFace size={16} />} label="Unknown faces" tone="red"
              value={faceStrangers ? faceStrangers.length : "—"} />
          </div>
      )}

      {/* Enroll form — Add New User popup modal (keeps the dense person grid
          uncluttered). All enroll state + handleEnroll stay in this component. */}
      {showEnroll && (
        <EnrollModal
          themeClass={themeClass}
          robotName={agentName}
          onClose={() => setShowEnroll(false)}
          onEnrolled={() => { setShowEnroll(false); void refresh(); void loadFaceOwners(); }}
        />
      )}

      {/* Rename modal — themed replacement for the native prompt()/alert(). */}
      {renaming != null && (
        <RenameModal
          themeClass={themeClass}
          renameValue={renameValue} setRenameValue={setRenameValue}
          renameError={renameError} setRenameError={setRenameError}
          renameSaving={renameSaving}
          onClose={() => setRenaming(null)}
          onSubmit={submitRename}
          inputStyle={inputStyle} fieldLabel={fieldLabel} btnStyle={btnStyle}
        />
      )}

      {/* Delete-user confirm — themed replacement for window.confirm(). */}
      {confirmDelete != null && (
        <ConfirmDialog
          danger
          title={`Remove ${confirmDelete}?`}
          message="This robot will forget this face and its photos."
          confirmLabel="Remove"
          onConfirm={confirmRemove}
          onCancel={() => setConfirmDelete(null)}
        />
      )}

      {/* Delete-photo confirm — single face photo from a user. */}
      {confirmPhoto != null && (
        <ConfirmDialog
          danger
          title="Delete this photo?"
          message={<>Remove <code style={{ color: "var(--lm-text)" }}>{confirmPhoto.filename}</code> from <span style={{ color: "var(--lm-text)", textTransform: "capitalize" }}>{confirmPhoto.label}</span>.</>}
          confirmLabel="Delete"
          onConfirm={confirmRemovePhoto}
          onCancel={() => setConfirmPhoto(null)}
        />
      )}

      {/* Delete-voice-sample confirm. */}
      {confirmVoice != null && (
        <ConfirmDialog
          danger
          title="Delete this voice sample?"
          message={<>Remove <code style={{ color: "var(--lm-text)" }}>{confirmVoice.filename}</code> from <span style={{ color: "var(--lm-text)", textTransform: "capitalize" }}>{confirmVoice.label}</span>.</>}
          confirmLabel="Delete"
          onConfirm={confirmRemoveVoice}
          onCancel={() => setConfirmVoice(null)}
        />
      )}

      {/* Delete stranger voice cluster confirm. */}
      {confirmCluster != null && (
        <ConfirmDialog
          danger
          title="Delete this cluster?"
          message={<>Cluster <code style={{ color: "var(--lm-text)" }}>{confirmCluster.hash}</code> ({confirmCluster.sampleCount} sample{confirmCluster.sampleCount !== 1 ? "s" : ""}) and its centroid will be removed.</>}
          confirmLabel="Delete"
          onConfirm={confirmDeleteCluster}
          onCancel={() => setConfirmCluster(null)}
        />
      )}

      {confirmForgetFace != null && (
        <ConfirmDialog
          danger
          title="Forget this face?"
          message="The robot will stop listing this unknown face. If they walk by again, they may show up as new."
          confirmLabel="Forget"
          onConfirm={confirmForgetFaceNow}
          onCancel={() => setConfirmForgetFace(null)}
        />
      )}

      {claimTarget != null && (
        <ClaimUnknownModal
          themeClass={themeClass}
          title="Who is this?"
          lead={claimTarget.kind === "face"
            ? "Give them a name, or pick someone the robot already knows."
            : "Name this voice, or pick someone the robot already knows."}
          photoUrl={claimTarget.kind === "face"
            ? hwUrl(`/face/stranger-photo/${encodeURIComponent(claimTarget.id)}`)
            : null}
          household={household.map((p) => p.label)}
          name={claimName} setName={setClaimName}
          asMe={claimAsMe} setAsMe={setClaimAsMe}
          error={claimError}
          saving={!!claimingFace || !!claimingVoice || settingMe}
          onClose={() => { setClaimAsMe(false); closeClaim(); }}
          onSubmit={() => { void (async () => {
            if (!(await submitClaim())) return;
            const named = claimName.trim().toLowerCase().replace(/[^a-z0-9_-]+/g, "_").replace(/^_+|_+$/g, "");
            if (claimAsMe && named) await handleSetMe(named);
            setClaimAsMe(false);
            void refresh();
            void loadFaceOwners();
          })(); }}
          inputStyle={inputStyle} fieldLabel={fieldLabel} btnStyle={btnStyle}
        />
      )}

      {/* Delete stranger sample file confirm. */}
      {confirmStrangerFile != null && (
        <ConfirmDialog
          danger
          title="Delete this sample?"
          message={<>Remove <code style={{ color: "var(--lm-text)" }}>{confirmStrangerFile.filename}</code> from <code style={{ color: "var(--lm-text)" }}>{confirmStrangerFile.hash}</code>.</>}
          confirmLabel="Delete"
          onConfirm={confirmDeleteStrangerFile}
          onCancel={() => setConfirmStrangerFile(null)}
        />
      )}

      {/* Person cards */}
      {data && (isDebug ? data.persons : household).length > 0 && (
        <div className="lm-grid-4">
          {(isDebug ? data.persons : household).map((person, idx) => (
            <PersonCard
              key={person.label}
              person={person}
              idx={idx}
              currentUser={currentUser}
              isMe={person.label === meLabel}
              settingMe={settingMe}
              onSetMe={person.label === "unknown" ? undefined : handleSetMe}
              expandedPerson={expandedPerson}
              setExpandedPerson={setExpandedPerson}
              hoveredPerson={hoveredPerson}
              setHoveredPerson={setHoveredPerson}
              hoveredPhoto={hoveredPhoto}
              setHoveredPhoto={setHoveredPhoto}
              expanded={expanded}
              toggleDir={toggleDir}
              deleting={deleting}
              deletingPhoto={deletingPhoto}
              preview={preview}
              previewLoading={previewLoading}
              setPreview={setPreview}
              playingAudio={playingAudio}
              onRename={handleRename}
              onRemove={handleRemove}
              onRemovePhoto={handleRemovePhoto}
              onRemoveVoiceFile={handleRemoveVoiceFile}
              onOpenFile={openFile}
              onTimeline={setTimelineUser}
              onRecordVoice={(label) => { setVoiceFor(label); setPhotoFor(null); }}
              onAddPhoto={(label) => { setPhotoFor(label); setVoiceFor(null); }}
              monCard={monCard}
              iconBtnStyle={iconBtnStyle}
            />
          ))}
        </div>
      )}

      {data && household.length === 0 && !showEnroll && !(faceStrangers && faceStrangers.length) && (
        <div className="lm-mon-card" style={monCard}>
          <EmptyState icon={<UserPlus size={20} />} text="Nobody here yet. Add a friend — look at the camera and type their name." />
        </div>
      )}

      {((faceStrangers && faceStrangers.length > 0) || faceStrangersError) && (
        <div style={{ marginTop: 14 }}>
        <UnknownFacesCard
          faceStrangers={faceStrangers}
          faceStrangersError={faceStrangersError}
          claimingId={claimingFace}
          forgettingId={forgettingFace}
          onClaim={openClaimFace}
          onForget={handleForgetFace}
          monCard={monCard}
          cardHeader={cardHeader}
        />
        </div>
      )}

      {((strangers && strangers.total > 0) || (isDebug && strangersError)) && (
        <div style={{ marginTop: 14 }}>
          <StrangerClustersCard
            strangers={strangers}
            strangersError={strangersError}
            expandedCluster={expandedCluster}
            setExpandedCluster={setExpandedCluster}
            deletingCluster={deletingCluster}
            deletingStrangerFile={deletingStrangerFile}
            claimingHash={claimingVoice}
            onDeleteCluster={handleDeleteCluster}
            onDeleteStrangerFile={handleDeleteStrangerFile}
            onClaim={openClaimVoice}
            monCard={monCard}
            cardHeader={cardHeader}
          />
        </div>
      )}

      {isDebug && (
      <div className="lm-grid-3" style={{ marginTop: 14 }}>

      {/* Face Recognition Cooldowns */}
      <CooldownsCard
        allCooldownEntries={allCooldownEntries}
        cdError={cdError}
        hasActiveCooldowns={hasActiveCooldowns}
        resetting={resetting}
        onReset={handleResetCooldowns}
        monCard={monCard}
        cardHeader={cardHeader}
      />

      </div>
      )}

      {photoFor && (
        <FriendPhotoModal
          themeClass={themeClass}
          person={data?.persons.find((p) => p.label === photoFor) ?? { label: photoFor, photos: [], photo_count: 0 }}
          robotName={agentName}
          onClose={() => setPhotoFor(null)}
          onChanged={() => { void refresh(); void loadFaceOwners(); }}
        />
      )}
      {voiceFor && (
        <ContactVoiceBar
          name={voiceFor}
          sttLanguage={sttLanguage}
          onDone={() => { void refresh(); void loadFaceOwners(); }}
          onClose={() => setVoiceFor(null)}
        />
      )}

      {isDebug && (
        <p style={{ fontSize: 12, color: "var(--lm-text-muted)", marginTop: 16 }}>
          Tweaker enroll forms still live under Device → Add a face / My Voice when Advanced is on.
        </p>
      )}

      {timelineUser && (
        <UserTimelineModal user={timelineUser} onClose={() => setTimelineUser(null)} />
      )}
    </div>
  );
}
