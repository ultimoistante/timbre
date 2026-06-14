import { writable, derived } from 'svelte/store';
import { api } from '$lib/api/client.js';

export const queue    = writable([]);    // list of MediaFile objects
export const queueIdx = writable(-1);    // current position in queue
export const playing  = writable(false);
export const progress = writable(0);     // 0-1
export const volume   = writable(1);
export const quality  = writable('original'); // 'original'|'320k'|'128k'|'64k'
export const container = writable('mp3');
export const nowPlaying = writable('');  // live ICY title for the current web radio

export const currentTrack = derived(
  [queue, queueIdx],
  ([$q, $i]) => ($i >= 0 && $i < $q.length) ? $q[$i] : null
);

export const player = {
  play(tracks, startIndex = 0) {
    queue.set(tracks);
    queueIdx.set(startIndex);
    playing.set(true);
  },
  enqueue(tracks) {
    queue.update(q => [...q, ...tracks]);
  },
  /**
   * Play a single web radio station. Streams are live (no seek/duration/queue),
   * so the queue holds just this one item flagged isStream. The src points at
   * the server proxy endpoint (handles http→https + ICY metadata).
   */
  playStream(station) {
    nowPlaying.set('');
    queue.set([{
      isStream: true,
      id: station.id,
      title: station.name,
      artists: station.genre || 'Web radio',
      streamUrl: `/api/streams/${station.id}/play`,
      favicon: station.favicon || ''
    }]);
    queueIdx.set(0);
    playing.set(true);
  },
  next() {
    queueIdx.update(i => {
      const q = /** @type {any[]} */ ([]);
      // We read queue synchronously via get() — import it if needed.
      return i + 1;
    });
  },
  prev() {
    queueIdx.update(i => Math.max(0, i - 1));
  },
  togglePlay() {
    playing.update(v => !v);
  },
  setProgress(v) {
    progress.set(v);
  },
  getStreamUrl(track, $quality, $container) {
    return api.streamUrl(track.id, $quality, $container);
  }
};
