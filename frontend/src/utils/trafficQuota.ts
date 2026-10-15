export function bytesToGb(value: number): number {
  if (!Number.isFinite(value) || value <= 0) {
    return 0;
  }
  return Number((value / 1024 ** 3).toFixed(2));
}

// Preserve the stored byte limit when only another field was edited. An explicit
// quota edit still wins, including changing a rounded tiny limit to unlimited.
export function resolveTotalBytes(
  originalBytes: number | null | undefined,
  displayedGB: number,
  wasEdited = false,
): number {
  if (
    originalBytes != null &&
    Number.isFinite(originalBytes) &&
    originalBytes >= 0 &&
    !wasEdited &&
    displayedGB === bytesToGb(originalBytes)
  ) {
    return originalBytes;
  }
  if (!Number.isFinite(displayedGB) || displayedGB <= 0) {
    return 0;
  }
  return Math.round(displayedGB * 1024 ** 3);
}
