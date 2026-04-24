import { parseContentDispositionFilename } from './content-disposition.util';

describe('parseContentDispositionFilename', () => {
  it('prefers UTF-8 filename* values', () => {
    const header = `attachment; filename="15ZsviIr4apvdoQUup7yFn _ Civil War - Hj_ltar ifr_n Dalarna.mp3"; filename*=UTF-8''15ZsviIr4apvdoQUup7yFn%20%E2%80%94%20Civil%20War%20-%20Hj%C3%A4ltar%20ifr%C3%A5n%20Dalarna.mp3`;

    expect(parseContentDispositionFilename(header)).toBe(
      '15ZsviIr4apvdoQUup7yFn — Civil War - Hjältar ifrån Dalarna.mp3',
    );
  });

  it('falls back to a quoted filename when filename* is absent', () => {
    expect(parseContentDispositionFilename('attachment; filename="playlist.zip"')).toBe('playlist.zip');
  });

  it('returns the provided fallback when no filename is present', () => {
    expect(parseContentDispositionFilename('attachment', 'download.zip')).toBe('download.zip');
  });
});
