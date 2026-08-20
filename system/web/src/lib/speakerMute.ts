/** Speaker MUTED on Home. Prefer /voice/status so a body without music still reports it. */

export function speakerMutedFromVoice(
  voice: { speaker_muted?: boolean } | null | undefined,
): boolean | undefined {
  if (!voice || voice.speaker_muted === undefined) return undefined;
  return Boolean(voice.speaker_muted);
}
