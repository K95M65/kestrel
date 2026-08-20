/** Target name for HAL /speaker/record-enroll. Voice samples belong to a
 *  person — refuse to start until a contact is chosen. */
export function voiceEnrollTarget(name: string | null | undefined): string {
  const n = (name ?? "").trim();
  if (!n) {
    throw new Error("Pick a person first");
  }
  return n.toLowerCase();
}

export function canStartVoiceEnroll(name: string | null | undefined): boolean {
  try {
    voiceEnrollTarget(name);
    return true;
  } catch {
    return false;
  }
}
