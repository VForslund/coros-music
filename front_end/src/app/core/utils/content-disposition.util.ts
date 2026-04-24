const RFC5987_FILENAME_RE = /filename\*\s*=\s*([^;\r\n]+)/i;
const BASIC_FILENAME_RE = /filename\s*=\s*(?:"([^"]+)"|([^;\r\n]+))/i;

export function parseContentDispositionFilename(value: string | null | undefined, fallback = ''): string {
  if (!value) return fallback;

  const extendedMatch = value.match(RFC5987_FILENAME_RE)?.[1];
  const extendedFilename = extendedMatch ? decodeExtendedFilename(extendedMatch) : '';
  if (extendedFilename) {
    return extendedFilename;
  }

  const basicMatch = value.match(BASIC_FILENAME_RE);
  const basicFilename = basicMatch?.[1] ?? basicMatch?.[2];
  return basicFilename?.trim() || fallback;
}

function decodeExtendedFilename(rawValue: string): string {
  const value = stripQuotes(rawValue.trim());
  const sections = value.split("'");
  const encodedFilename = sections.length >= 3 ? sections.slice(2).join("'") : value;

  try {
    return decodeURIComponent(encodedFilename);
  } catch {
    return '';
  }
}

function stripQuotes(value: string): string {
  if (value.startsWith('"') && value.endsWith('"')) {
    return value.slice(1, -1);
  }

  return value;
}
