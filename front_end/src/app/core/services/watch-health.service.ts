import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { signal, computed } from '@angular/core';

export interface HealthStatus {
  ffmpeg: boolean;
  ytdlp: boolean;
  spotify: boolean;
}

@Injectable({ providedIn: 'root' })
export class WatchHealthService {
  private http = inject(HttpClient);

  readonly health = signal<HealthStatus | null>(null);
  readonly healthy = computed(() => {
    const h = this.health();
    return h ? h.ffmpeg && h.ytdlp : false;
  });

  async checkHealth(): Promise<void> {
    try {
      const data = await firstValueFrom(
        this.http.get<HealthStatus>('/api/health')
      );
      this.health.set(data);
    } catch {
      this.health.set(null);
    }
  }
}

