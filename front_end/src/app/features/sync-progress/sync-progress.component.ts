import { Component, inject } from '@angular/core';
import { SyncService } from '../../core/services/sync.service';
import {
  tracks,
  tracksToSync,
  tracksSynced,
  selectedPlaylist,
  syncPlaylist,
  syncStatus,
  syncError,
  progress,
  completedTracks,
  totalTracks,
  currentTrackName,
  watchConnected,
} from '../../core/state/sync.state';

@Component({
  selector: 'app-sync-progress',
  templateUrl: './sync-progress.component.html',
})
export class SyncProgressComponent {
  private syncService = inject(SyncService);

  protected tracks = tracks;
  protected tracksToSync = tracksToSync;
  protected tracksSynced = tracksSynced;
  protected selectedPlaylist = selectedPlaylist;
  protected syncPlaylist = syncPlaylist;
  protected syncStatus = syncStatus;
  protected syncError = syncError;
  protected progress = progress;
  protected completedTracks = completedTracks;
  protected totalTracks = totalTracks;
  protected currentTrackName = currentTrackName;
  protected watchConnected = watchConnected;

  private syncedIds = new Set<string>();

  isSynced(id: string): boolean {
    // Recompute from tracksSynced signal
    if (this.syncedIds.size !== this.tracksSynced().length) {
      this.syncedIds = new Set(this.tracksSynced().map(t => t.id));
    }
    return this.syncedIds.has(id);
  }

  async sync(): Promise<void> {
    await this.syncService.startSync();
  }

  cancel(): void {
    this.syncService.cancelSync();
  }

  formatDuration(ms: number): string {
    const min = Math.floor(ms / 60000);
    const sec = Math.floor((ms % 60000) / 1000);
    return `${min}:${sec.toString().padStart(2, '0')}`;
  }
}

