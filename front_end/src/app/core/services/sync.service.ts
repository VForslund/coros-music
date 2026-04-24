import { Injectable, inject } from '@angular/core';
import { FileBridgeService } from './file-bridge.service';
import {
  selectedPlaylist,
  tracks,
  existingFiles,
  totalTracks,
  completedTracks,
  currentTrackName,
  syncStatus,
  syncError,
} from '../state/sync.state';
import { Track } from '../models/track.model';
import { parseContentDispositionFilename } from '../utils/content-disposition.util';
import { normalizeWatchFilename } from '../utils/watch-filename.util';

@Injectable({ providedIn: 'root' })
export class SyncService {
  private fileBridge = inject(FileBridgeService);
  private abortController: AbortController | null = null;

  async startSync(trackIds?: string[]): Promise<void> {
    const playlist = selectedPlaylist();
    if (!playlist) return;

    let toSync: Track[] = [];

    try {
      toSync = this.resolveTracksForSync(tracks(), trackIds);
      if (toSync.length === 0) return;

      totalTracks.set(toSync.length);
      completedTracks.set(0);
      syncStatus.set('syncing');
      syncError.set('');

      this.abortController = new AbortController();

      const resp = await fetch('/api/sync/stream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          playlistId: playlist.id,
          playlistName: playlist.name,
          trackIds: toSync.map(t => t.id),
          tracks: toSync.map(t => ({
            id: t.id,
            title: t.title,
            artist: t.artist,
            album: t.album,
            syncFilename: t.syncFilename,
          })),
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


  private resolveTracksForSync(sourceTracks: Track[], trackIds?: string[]) {
    const ids = trackIds && trackIds.length > 0 ? new Set(trackIds) : null;
    const existingNames = new Set(existingFiles().map(normalizeWatchFilename));
    const pendingById = new Set(
      sourceTracks
        .filter(t => !existingNames.has(normalizeWatchFilename(t.syncFilename)))
        .map(t => t.id)
    );
    return (ids ? sourceTracks.filter(t => ids.has(t.id)) : sourceTracks)
      .filter(t => pendingById.has(t.id));
  }

  private async parseMultipartStream(body: ReadableStream<Uint8Array>, boundary: string): Promise<void> {
    const reader = body.getReader();
    let buffer: Uint8Array;
    // Keep a 1:1 byte-to-char mapping while scanning multipart headers.
    const latin1Decoder = new TextDecoder('latin1');

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

    const fullText = latin1Decoder.decode(buffer);
    const marker = `--${boundary}`;
    const markerWithCrlf = `\r\n${marker}`;

    let boundaryStart = fullText.indexOf(marker);
    while (boundaryStart !== -1) {
      const afterMarker = boundaryStart + marker.length;

      // Closing boundary: --boundary--
      if (fullText.startsWith('--', afterMarker)) {
        break;
      }

      // Skip CRLF after boundary line
      let headerStart = afterMarker;
      if (fullText.startsWith('\r\n', headerStart)) {
        headerStart += 2;
      }

      const headerEnd = fullText.indexOf('\r\n\r\n', headerStart);
      if (headerEnd === -1) break;

      const nextBoundary = fullText.indexOf(markerWithCrlf, headerEnd + 4);
      if (nextBoundary === -1) break;

      const headerSection = fullText.slice(headerStart, headerEnd);
      const filename = parseContentDispositionFilename(headerSection);
      if (!filename) {
        boundaryStart = nextBoundary + 2; // skip leading CRLF
        continue;
      }

      const errorMatch = headerSection.match(/X-Track-Error:\s*(.+)/i);
      if (errorMatch) {
        completedTracks.update(n => n + 1);
        boundaryStart = nextBoundary + 2;
        continue;
      }

      currentTrackName.set(filename.replace('.mp3', ''));

      const bodyStart = headerEnd + 4;
      const bodyBytes = buffer.slice(bodyStart, nextBoundary); // boundary begins at CRLF before marker

      // Ignore empty payloads instead of writing corrupt 0-byte files.
      if (bodyBytes.length > 0) {
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
      }

      completedTracks.update(n => n + 1);
      boundaryStart = nextBoundary + 2;
    }
  }
}

