const LEGACY_TRACK_ID_PREFIX_RE = /^[A-Za-z0-9]{22}\s+—\s+/;

/**
 * Normalizes watch filenames so old "{trackId} — ..." files still match new naming.
 */
export function normalizeWatchFilename(filename: string): string {
  return filename.replace(LEGACY_TRACK_ID_PREFIX_RE, '').trim().toLowerCase();
}

