/** Newest enrolled still is the one the card shows (filenames are timestamps). */
export function mainFacePhoto(photos?: string[] | null): string | undefined {
  if (!photos?.length) return undefined;
  return photos[photos.length - 1];
}
