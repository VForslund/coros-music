import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { Playlist } from '../models/playlist.model';
import { Track } from '../models/track.model';
import {
  playlists,
  tracks,
  selectedPlaylist,
  spotifyConnected,
  hydratePlaylistState,
  persistPlaylistState,
} from '../state/sync.state';

@Injectable({ providedIn: 'root' })
export class SpotifyService {
  private http = inject(HttpClient);

  /** Check backend connectivity. */
  async checkStatus(): Promise<boolean> {
    try {
      const res = await firstValueFrom(
        this.http.get<{ authenticated: boolean }>('/api/spotify/status')
      );
      spotifyConnected.set(res.authenticated);
      return res.authenticated;
    } catch {
      spotifyConnected.set(false);
      return false;
    }
  }

  /** Restore persisted playlist list + current selection after refresh. */
  async restoreFromStorage(): Promise<void> {
    const { persistedPlaylists, selectedId } = hydratePlaylistState();
    if (persistedPlaylists.length === 0) return;

    playlists.set(persistedPlaylists);

    const selected = persistedPlaylists.find(p => p.id === selectedId) ?? persistedPlaylists[0];
    await this.selectPlaylist(selected, { persist: false });
  }

  /**
   * Extract playlist ID from a Spotify URL or plain ID.
   * Supports: https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M?si=...
   */
  parsePlaylistId(input: string): string | null {
    input = input.trim();
    // Direct ID (alphanumeric, 22 chars)
    if (/^[a-zA-Z0-9]{22}$/.test(input)) return input;
    // URL
    const match = input.match(/playlist\/([a-zA-Z0-9]{22})/);
    return match ? match[1] : null;
  }

  /** Add a playlist by URL or ID. Fetches metadata and adds to the list. */
  async addPlaylist(input: string): Promise<void> {
    const id = this.parsePlaylistId(input);
    if (!id) throw new Error('Invalid Spotify playlist URL or ID');

    try {
      const playlist = await firstValueFrom(
        this.http.get<Playlist>(`/api/spotify/playlists/${id}`)
      );

      // Add to list if not already there
      const current = playlists();
      if (!current.find(p => p.id === playlist.id)) {
        playlists.set([...current, playlist]);
      }

      // Auto-select the latest added playlist.
      await this.selectPlaylist(playlist);
    } catch (err) {
      if (err instanceof HttpErrorResponse) {
        throw new Error(typeof err.error === 'string' ? err.error : `Failed to load playlist (${err.status})`);
      }
      throw err;
    }
  }

  /** Fetch tracks for a given playlist, updating state signals. */
  async selectPlaylist(playlist: Playlist, options: { persist?: boolean } = {}): Promise<void> {
    selectedPlaylist.set(playlist);
    const data = await firstValueFrom(
      this.http.get<Track[]>(`/api/spotify/playlists/${playlist.id}/tracks`)
    );
    tracks.set(data);

    if (options.persist !== false) {
      persistPlaylistState();
    }
  }

  /** Remove a playlist from the local list. */
  removePlaylist(id: string): void {
    const next = playlists().filter(p => p.id !== id);
    playlists.set(next);

    if (selectedPlaylist()?.id === id) {
      const fallback = next[0] ?? null;
      selectedPlaylist.set(fallback);
      tracks.set([]);
    }


    persistPlaylistState();
  }
}
