import { Component, inject, OnInit } from '@angular/core';
import { SpotifyService } from '../../core/services/spotify.service';
import { FileBridgeService } from '../../core/services/file-bridge.service';
import { WatchHealthService } from '../../core/services/watch-health.service';
import { PlaylistPickerComponent } from '../playlist-picker/playlist-picker.component';
import { SyncProgressComponent } from '../sync-progress/sync-progress.component';
import {
  watchConnected,
  existingFiles,
} from '../../core/state/sync.state';

@Component({
  selector: 'app-dashboard',
  imports: [PlaylistPickerComponent, SyncProgressComponent],
  templateUrl: './dashboard.component.html',
})
export class DashboardComponent implements OnInit {
  private spotify = inject(SpotifyService);
  private fileBridge = inject(FileBridgeService);
  protected health = inject(WatchHealthService);

  protected watchConnected = watchConnected;
  protected existingFiles = existingFiles;

  async ngOnInit(): Promise<void> {
    await this.health.checkHealth();
    const ok = await this.spotify.checkStatus();
    if (ok) {
      await this.spotify.restoreFromStorage();
    }
  }

  async connectWatch(): Promise<void> {
    try {
      await this.fileBridge.connectWatch();
    } catch (err: any) {
      alert(err.message);
    }
  }

  disconnectWatch(): void {
    this.fileBridge.disconnect();
  }
}
