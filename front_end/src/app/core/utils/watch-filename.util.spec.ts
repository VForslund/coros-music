import { normalizeWatchFilename } from './watch-filename.util';

describe('normalizeWatchFilename', () => {
  it('strips legacy track ID prefixes', () => {
    expect(normalizeWatchFilename('4NPeA5XXs6tjHrtanLYJxf — Artist - Title.mp3'))
      .toBe('artist - title.mp3');
  });

  it('keeps modern filenames as-is except normalization', () => {
    expect(normalizeWatchFilename('Artist - Title.mp3')).toBe('artist - title.mp3');
  });
});

