import { Injectable } from '@angular/core';
import { watchDirHandle, existingFiles } from '../state/sync.state';

@Injectable({ providedIn: 'root' })
export class FileBridgeService {

  /** Prompt user to select the COROS watch mount point. */
  async connectWatch(): Promise<void> {
    const handle = await (window as any).showDirectoryPicker({ mode: 'readwrite' });
    watchDirHandle.set(handle);
    await this.verifyMusicDir();
    await this.scanExistingFiles();
  }

  /** Disconnect from the watch. */
  disconnect(): void {
    watchDirHandle.set(null);
    existingFiles.set([]);
  }

  /** Verify /Music exists on the selected directory. */
  private async verifyMusicDir(): Promise<void> {
    const root = watchDirHandle()!;
    try {
      await root.getDirectoryHandle('Music');
    } catch {
      watchDirHandle.set(null);
      throw new Error('Selected directory does not contain a /Music folder. Is this a COROS watch?');
    }
  }

  /** Read existing filenames for delta sync. */
  async scanExistingFiles(): Promise<void> {
    const root = watchDirHandle();
    if (!root) return;
    const music = await root.getDirectoryHandle('Music');
    const names: string[] = [];
    for await (const [name] of (music as any).entries()) {
      if (name.endsWith('.mp3')) names.push(name);
    }
    existingFiles.set(names);
  }

  /** Write a single MP3 blob to the watch. */
  async writeTrack(filename: string, data: Uint8Array): Promise<void> {
    const root = watchDirHandle()!;
    const music = await root.getDirectoryHandle('Music');
    const fileHandle = await music.getFileHandle(filename, { create: true });
    const writable = await fileHandle.createWritable();
    await writable.write(data as any);
    await writable.close();
  }
}

