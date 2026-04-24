import { Injectable, inject } from '@angular/core';
import { FileBridgeService } from './file-bridge.service';
import {
  syncPlaylist,
  tracksToSync,
  existingFiles,
  totalTracks,
  completedTracks,
  currentTrackName,
  syncStatus,
  syncError,
} from '../state/sync.state';

@Injectable({ providedIn: 'root' })
export class SyncService {
  private fileBridge = inject(FileBridgeService);
  private abortController: AbortController | null = null;

  async startSync(): Promise<void> {
    const playlist = syncPlaylist();
    if (!playlist) return;

    const toSync = tracksToSync();
    if (toSync.length === 0) return;

    totalTracks.set(toSync.length);
    completedTracks.set(0);
    syncStatus.set('syncing');
    syncError.set('');

    this.abortController = new AbortController();

    try {
      const resp = await fetch('/api/sync/stream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          playlistId: playlist.id,
          trackIds: toSync.map(t => t.id),
          existingFiles: existingFiles(),
        }),
        signal: this.abortController.signal,
      });

      if (!resp.ok) {
        throw new Error(`Server returned ${resp.status}`);
      }

      const contentType = resp.headers.get('Content-Type') || '';
      const boundaryMatch = contentType.match(/boundary=(.+)/);
      if (!boundaryMatch) {
        throw new Error('No multipart boundary in response');
      }
      const boundary = boundaryMatch[1];

      await this.parseMultipartStream(resp.body!, boundary);

      await this.fileBridge.scanExistingFiles();
      syncStatus.set('done');
    } catch (err: any) {
      if (err.name === 'AbortError') {
        syncStatus.set('idle');
      } else {
        syncError.set(err.message || 'Unknown sync error');
        syncStatus.set('error');
      }
    } finally {
      this.abortController = null;
    }
  }

  cancelSync(): void {
    this.abortController?.abort();
  }

  private async parseMultipartStream(body: ReadableStream<Uint8Array>, boundary: string): Promise<void> {
    const reader = body.getReader();
    const decoder = new TextDecoder();
    let buffer = new Uint8Array(0);
    const boundaryBytes = new TextEncoder().encode('--' + boundary);
    const endBoundaryBytes = new TextEncoder().encode('--' + boundary + '--');

    // Read entire response and split by boundary
    // For simplicity, accumulate then split
    const chunks: Uint8Array[] = [];
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      chunks.push(value);
    }

    // Combine all chunks
    const totalLength = chunks.reduce((sum, c) => sum + c.length, 0);
    buffer = new Uint8Array(totalLength);
    let offset = 0;
    for (const chunk of chunks) {
      buffer.set(chunk, offset);
      offset += chunk.length;
    }

    const fullText = decoder.decode(buffer);
    const parts = fullText.split('--' + boundary).filter(p => p.trim() !== '' && p.trim() !== '--');

    for (const part of parts) {
      // Parse headers and body
      const headerEnd = part.indexOf('\r\n\r\n');
      if (headerEnd === -1) continue;

      const headerSection = part.substring(0, headerEnd);
      const bodyStart = headerEnd + 4;

      // Extract filename
      const filenameMatch = headerSection.match(/filename="([^"]+)"/);
      if (!filenameMatch) continue;
      const filename = filenameMatch[1];

      // Check for error header
      const errorMatch = headerSection.match(/X-Track-Error:\s*(.+)/i);
      if (errorMatch) {
        completedTracks.update(n => n + 1);
        continue;
      }

      currentTrackName.set(filename.replace('.mp3', ''));

      // Get body bytes (re-encode from the original buffer)
      const bodyText = part.substring(bodyStart);
      // We need the raw bytes, so find this part in the original buffer
      const partStart = fullText.indexOf(part);
      const bodyBytes = buffer.slice(
        new TextEncoder().encode(fullText.substring(0, partStart + bodyStart)).length,
        new TextEncoder().encode(fullText.substring(0, partStart + bodyStart + bodyText.length)).length
      );

      try {
        await this.fileBridge.writeTrack(filename, bodyBytes);
      } catch (err: any) {
        if (err.name === 'NotFoundError') {
          syncError.set('Watch disconnected during sync');
          syncStatus.set('error');
          return;
        }
        // Skip track on other write errors
      }

      completedTracks.update(n => n + 1);
    }
  }
}

