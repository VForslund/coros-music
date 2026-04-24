import { Component, inject, signal, computed } from '@angular/core';
import { SyncService } from '../../core/services/sync.service';
import { FileBridgeService } from '../../core/services/file-bridge.service';
import {
  tracks,
  tracksToSync,
  tracksSynced,
  existingFiles,
  selectedPlaylist,
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
  private fileBridge = inject(FileBridgeService);

  protected tracks = tracks;
  protected tracksToSync = tracksToSync;
  protected tracksSynced = tracksSynced;
  protected selectedPlaylist = selectedPlaylist;
  protected syncStatus = syncStatus;
  protected syncError = syncError;
  protected progress = progress;
  protected completedTracks = completedTracks;
  protected totalTracks = totalTracks;
  protected currentTrackName = currentTrackName;
  protected watchConnected = watchConnected;
  protected existingFiles = existingFiles;

  protected selectedTrackIds = signal<Set<string>>(new Set<string>());
  protected selectedWatchFiles = signal<Set<string>>(new Set<string>());
  protected actionError = signal('');

  protected selectedTrackCount = computed(() => {
    const validTrackIDs = new Set(this.tracks().map(t => t.id));
    let count = 0;
    for (const id of this.selectedTrackIds()) {
      if (validTrackIDs.has(id)) count++;
    }
    return count;
  });
  protected selectedPendingTrackCount = computed(() => {
    const pendingTrackIDs = new Set(this.tracksToSync().map(t => t.id));
    let count = 0;
    for (const id of this.selectedTrackIds()) {
      if (pendingTrackIDs.has(id)) count++;
    }
    return count;
  });
  protected selectedWatchCount = computed(() => {
    const validFiles = new Set(this.existingFiles());
    let count = 0;
    for (const name of this.selectedWatchFiles()) {
      if (validFiles.has(name)) count++;
    }
    return count;
  });

  private syncedIds = new Set<string>();

  isSynced(id: string): boolean {
    // Recompute from tracksSynced signal
    if (this.syncedIds.size !== this.tracksSynced().length) {
      this.syncedIds = new Set(this.tracksSynced().map(t => t.id));
    }
    return this.syncedIds.has(id);
  }

  toggleTrack(id: string): void {
    this.selectedTrackIds.update(current => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  toggleWatchFile(name: string): void {
    this.selectedWatchFiles.update(current => {
      const next = new Set(current);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }

  selectAllTracks(): void {
    this.selectedTrackIds.set(new Set(this.tracksToSync().map(t => t.id)));
  }

  clearTrackSelection(): void {
    this.selectedTrackIds.set(new Set());
  }

  async syncAllPending(): Promise<void> {
    this.actionError.set('');
    await this.syncService.startSync();
  }

  async syncSelected(): Promise<void> {
    this.actionError.set('');
    const pendingIds = this.tracksToSync()
      .filter(track => this.selectedTrackIds().has(track.id))
      .map(track => track.id);

    if (pendingIds.length === 0) {
      this.actionError.set('Select at least one track that is not already on the watch.');
      return;
    }

    await this.syncService.startSync(pendingIds);
  }

  canSyncSelected(): boolean {
    return this.watchConnected() && this.selectedPendingTrackCount() > 0 && this.syncStatus() !== 'syncing';
  }

  cancel(): void {
    this.syncService.cancelSync();
  }

  async removeSelectedWatchTracks(): Promise<void> {
    this.actionError.set('');
    try {
      await this.fileBridge.removeTracks(Array.from(this.selectedWatchFiles()));
      this.selectedWatchFiles.set(new Set());
    } catch (err: any) {
      this.actionError.set(err.message || 'Failed to remove tracks from watch');
    }
  }

  async removeSingleWatchTrack(filename: string): Promise<void> {
    this.actionError.set('');
    try {
      await this.fileBridge.removeTracks([filename]);
      this.selectedWatchFiles.update(current => {
        const next = new Set(current);
        next.delete(filename);
        return next;
      });
    } catch (err: any) {
      this.actionError.set(err.message || 'Failed to remove track from watch');
    }
  }

  watchLabel(filename: string): string {
    return filename.replace(/^[^ ]+\s+—\s+/, '').replace(/\.mp3$/i, '');
  }

  formatDuration(ms: number): string {
    const min = Math.floor(ms / 60000);
    const sec = Math.floor((ms % 60000) / 1000);
    return `${min}:${sec.toString().padStart(2, '0')}`;
  }
}
