import { signal, computed } from '@angular/core';
import { Playlist } from '../models/playlist.model';
import { Track } from '../models/track.model';

const PLAYLISTS_STORAGE_KEY = 'coros.playlists.v1';
const SELECTED_PLAYLIST_ID_KEY = 'coros.selectedPlaylistId.v1';
const SYNC_PLAYLIST_ID_KEY = 'coros.syncPlaylistId.v1';

// --- Core Signals ---
export const spotifyConnected = signal(false);
export const playlists = signal<Playlist[]>([]);
export const selectedPlaylist = signal<Playlist | null>(null);
// Sync target can be overridden independently from current viewing selection.
export const syncPlaylist = signal<Playlist | null>(null);
export const tracks = signal<Track[]>([]);

// Sync progress
export const totalTracks = signal(0);
export const completedTracks = signal(0);
export const currentTrackName = signal('');
export const syncStatus = signal<'idle' | 'syncing' | 'done' | 'error'>('idle');
export const syncError = signal('');

// Watch connection
export const watchDirHandle = signal<FileSystemDirectoryHandle | null>(null);
export const watchConnected = computed(() => watchDirHandle() !== null);
export const existingFiles = signal<string[]>([]);

// Derived
export const progress = computed(() =>
  totalTracks() === 0 ? 0 : Math.round((completedTracks() / totalTracks()) * 100)
);
export const tracksToSync = computed(() => {
  const existing = new Set(existingFiles().map(f => f.split(' —')[0]));
  return tracks().filter(t => !existing.has(t.id));
});
export const tracksSynced = computed(() => {
  const existing = new Set(existingFiles().map(f => f.split(' —')[0]));
  return tracks().filter(t => existing.has(t.id));
});

export function persistPlaylistState(): void {
  if (typeof localStorage === 'undefined') return;
  localStorage.setItem(PLAYLISTS_STORAGE_KEY, JSON.stringify(playlists()));
  localStorage.setItem(SELECTED_PLAYLIST_ID_KEY, selectedPlaylist()?.id ?? '');
  localStorage.setItem(SYNC_PLAYLIST_ID_KEY, syncPlaylist()?.id ?? '');
}

export function hydratePlaylistState(): {
  persistedPlaylists: Playlist[];
  selectedId: string | null;
  syncId: string | null;
} {
  if (typeof localStorage === 'undefined') {
    return { persistedPlaylists: [], selectedId: null, syncId: null };
  }

  try {
    const persistedPlaylists = JSON.parse(localStorage.getItem(PLAYLISTS_STORAGE_KEY) ?? '[]') as Playlist[];
    const selectedId = localStorage.getItem(SELECTED_PLAYLIST_ID_KEY) || null;
    const syncId = localStorage.getItem(SYNC_PLAYLIST_ID_KEY) || null;
    return { persistedPlaylists, selectedId, syncId };
  } catch {
    return { persistedPlaylists: [], selectedId: null, syncId: null };
  }
}
