import { Component, inject, signal } from '@angular/core';
import { SpotifyService } from '../../core/services/spotify.service';
import { playlists, selectedPlaylist, syncPlaylist } from '../../core/state/sync.state';
import { Playlist } from '../../core/models/playlist.model';

@Component({
  selector: 'app-playlist-picker',
  templateUrl: './playlist-picker.component.html',
})
export class PlaylistPickerComponent {
  private spotify = inject(SpotifyService);
  protected playlists = playlists;
  protected selectedPlaylist = selectedPlaylist;
  protected syncPlaylist = syncPlaylist;
  protected playlistInput = signal('');
  protected loading = signal(false);
  protected error = signal('');

  async addPlaylist(): Promise<void> {
    const input = this.playlistInput().trim();
    if (!input) return;
    this.loading.set(true);
    this.error.set('');
    try {
      await this.spotify.addPlaylist(input);
      this.playlistInput.set('');
    } catch (err: any) {
      this.error.set(err.message || 'Failed to load playlist');
    } finally {
      this.loading.set(false);
    }
  }

  async select(playlist: Playlist): Promise<void> {
    await this.spotify.selectPlaylist(playlist);
  }

  setSyncTarget(playlist: Playlist, event: Event): void {
    event.stopPropagation();
    this.spotify.setSyncPlaylist(playlist);
  }

  remove(id: string, event: Event): void {
    event.stopPropagation();
    this.spotify.removePlaylist(id);
  }
}
