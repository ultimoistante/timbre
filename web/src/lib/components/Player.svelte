<script>
  import { tick, onMount, onDestroy } from 'svelte';
  import { get } from 'svelte/store';
  import { queue, queueIdx, playing, progress, volume, quality, container, currentTrack, nowPlaying } from '$lib/stores/player.js';
  import { auth } from '$lib/stores/auth.js';
  import { api } from '$lib/api/client.js';

  let audio;
  let seeking = false;
  let currentTime = 0;
  let duration = 0;

  $: isStream = !!$currentTrack?.isStream;
  // When native playback of a stream fails (browser can't decode the source
  // codec, e.g. raw AAC), retry once via the server's MP3 transcode fallback.
  let streamFallback = false;

  // Stub: favorite state (no backend yet)
  let favorited = false;
  function toggleFavorite() { favorited = !favorited; }

  // Quality/container popover
  let showQualityMenu = false;
  function toggleQualityMenu() { showQualityMenu = !showQualityMenu; }

  // True while focus is on something that should keep its own keystrokes
  // (text fields, selects) — playback shortcuts are suppressed there.
  function isTypingTarget(el) {
    if (!el) return false;
    const tag = el.tagName;
    return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || el.isContentEditable;
  }

  // Same shortcuts as the Spotify desktop app: Space play/pause,
  // Ctrl/Cmd+Left/Right prev/next, Ctrl/Cmd+Up/Down volume.
  function handleWindowKeydown(e) {
    if (e.key === 'Escape' && showQualityMenu) { showQualityMenu = false; return; }
    if (isTypingTarget(e.target)) return;

    const mod = e.ctrlKey || e.metaKey;

    // Skip Space when a focused button would already toggle on its own
    // native activation (e.g. the play/pause button itself) to avoid a
    // double-toggle.
    if (!mod && e.code === 'Space' && e.target.tagName !== 'BUTTON') {
      e.preventDefault();
      if ($currentTrack) playing.update(v => !v);
      return;
    }

    if (mod && e.key === 'ArrowRight') {
      e.preventDefault();
      if ($currentTrack && !isStream) { playing.set(true); queueIdx.update(i => (i + 1) % get(queue).length); }
    } else if (mod && e.key === 'ArrowLeft') {
      e.preventDefault();
      if ($currentTrack && !isStream) { playing.set(true); queueIdx.update(i => Math.max(0, i - 1)); }
    } else if (mod && e.key === 'ArrowUp') {
      e.preventDefault();
      volume.update(v => Math.min(1, +(v + 0.1).toFixed(2)));
    } else if (mod && e.key === 'ArrowDown') {
      e.preventDefault();
      volume.update(v => Math.max(0, +(v - 0.1).toFixed(2)));
    }
  }

  function handleQualityMenuBlur(e) {
    if (!e.relatedTarget || !e.currentTarget.contains(e.relatedTarget)) {
      showQualityMenu = false;
    }
  }

  $: src = !$currentTrack
    ? null
    : isStream
      ? $currentTrack.streamUrl + (streamFallback ? '?transcode=mp3' : '')
      : api.streamUrl($currentTrack.id, $quality, $container);

  async function onAudioError() {
    if (isStream && !streamFallback) {
      // Switch to the transcoded source and resume.
      streamFallback = true;
      await tick();
      audio?.play().catch(() => playing.set(false));
    } else {
      playing.set(false);
    }
  }

  $: if (audio) {
    if ($playing) audio.play().catch(() => playing.set(false));
    else audio.pause();
  }

  $: if (audio) audio.volume = $volume;

  function onTimeUpdate() {
    currentTime = audio.currentTime;
    duration = audio.duration || 0;
    if (!seeking && duration) progress.set(currentTime / duration);
  }

  function onLoadedMetadata() {
    duration = audio.duration || 0;
  }

  function onEnded() {
    const nextIdx = get(queueIdx) + 1;
    if (nextIdx < get(queue).length) {
      queueIdx.set(nextIdx);
    } else {
      playing.set(false);
      progress.set(0);
    }
  }

  function seekTo(e) {
    if (!audio?.duration) return;
    const pct = e.target.value / 100;
    audio.currentTime = pct * audio.duration;
    progress.set(pct);
  }

  async function onTrackChange() {
    currentTime = 0;
    duration = 0;
    progress.set(0);
    nowPlaying.set('');
    streamFallback = false;
    if (!$currentTrack) return;
    await tick(); // wait for Svelte to flush audio.src to the DOM
    if (!audio) return;
    if ($playing) audio.play().catch(() => playing.set(false));
    updateMediaSession();
  }

  $: $currentTrack && onTrackChange();

  function updateMediaSession() {
    if (!('mediaSession' in navigator) || !$currentTrack) return;
    navigator.mediaSession.metadata = new MediaMetadata({
      title:  $currentTrack.title  || 'Unknown',
      artist: $currentTrack.artists || 'Unknown',
      album:  $currentTrack.album  || '',
    });
    navigator.mediaSession.setActionHandler('play',           () => playing.set(true));
    navigator.mediaSession.setActionHandler('pause',          () => playing.set(false));
    navigator.mediaSession.setActionHandler('previoustrack',  () => queueIdx.update(i => Math.max(0, i - 1)));
    navigator.mediaSession.setActionHandler('nexttrack',      () => queueIdx.update(i => Math.min(get(queue).length - 1, i + 1)));
  }

  // Subscribe to live ICY now-playing titles pushed by the radio proxy.
  let es;
  onMount(() => {
    if (!get(auth).accessToken) return;
    es = new EventSource('/api/events', { withCredentials: true });
    es.addEventListener('nowplaying', (e) => {
      try {
        const { title } = JSON.parse(e.data);
        if (isStream) nowPlaying.set(title || '');
      } catch { /* ignore malformed */ }
    });
  });
  onDestroy(() => es?.close());

  function fmtTime(sec) {
    if (!sec || isNaN(sec)) return '0:00';
    const m = Math.floor(sec / 60);
    const s = Math.floor(sec % 60).toString().padStart(2, '0');
    return `${m}:${s}`;
  }
</script>

<svelte:window on:keydown={handleWindowKeydown} />

<!-- Hidden audio element -->
{#if src}
  <audio
    bind:this={audio}
    {src}
    on:timeupdate={onTimeUpdate}
    on:loadedmetadata={onLoadedMetadata}
    on:ended={onEnded}
    on:error={onAudioError}
    preload="metadata"
  ></audio>
{/if}

<div class="player">

  <!-- Left zone: heart + track info -->
  <div class="left-zone">
    <button
      class="icon-btn heart-btn"
      class:favorited
      on:click={toggleFavorite}
      title={favorited ? 'Remove from favorites' : 'Add to favorites'}
      aria-label={favorited ? 'Remove from favorites' : 'Add to favorites'}
    >
      {#if favorited}
        <!-- Heart filled -->
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="currentColor" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>
      {:else}
        <!-- Heart empty -->
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>
      {/if}
    </button>

    <div class="track-info">
      {#if $currentTrack}
        <span class="title">
          {#if isStream}<span class="live-badge">LIVE</span>{/if}
          {$currentTrack.title || '—'}
        </span>
        <span class="artist">
          {#if isStream}{$nowPlaying || $currentTrack.artists || ''}{:else}{$currentTrack.artists || ''}{/if}
        </span>
      {:else}
        <span class="title muted">No track selected</span>
      {/if}
    </div>
  </div>

  <!-- Center zone: prev / play / next + seek bar -->
  <div class="center-zone">
    <div class="controls">
      <button
        class="icon-btn"
        on:click={() => { playing.set(true); queueIdx.update(i => Math.max(0, i - 1)); }}
        title="Previous"
        aria-label="Previous track"
        disabled={!$currentTrack || isStream}
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="19 20 9 12 19 4 19 20"/><line x1="5" y1="19" x2="5" y2="5"/></svg>
      </button>

      <button
        class="icon-btn play-btn"
        on:click={() => playing.update(v => !v)}
        title="Play/Pause"
        aria-label={$playing ? 'Pause' : 'Play'}
        disabled={!$currentTrack}
      >
        {#if $playing}
          <!-- Pause -->
          <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="currentColor" stroke="none"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>
        {:else}
          <!-- Play -->
          <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="currentColor" stroke="none"><polygon points="5 3 19 12 5 21 5 3"/></svg>
        {/if}
      </button>

      <button
        class="icon-btn"
        on:click={() => { playing.set(true); queueIdx.update(i => (i + 1) % get(queue).length); }}
        title="Next"
        aria-label="Next track"
        disabled={!$currentTrack || isStream}
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="5 4 15 12 5 20 5 4"/><line x1="19" y1="5" x2="19" y2="19"/></svg>
      </button>
    </div>

    {#if isStream}
      <div class="seek-row live-row">
        <span class="live-dot" class:on={$playing}></span>
        <span class="time live-text">Live stream</span>
      </div>
    {:else}
      <div class="seek-row">
        <span class="time">{fmtTime(currentTime)}</span>
        <input
          type="range" min="0" max="100"
          value={$progress * 100}
          on:mousedown={() => seeking = true}
          on:mouseup={(e) => { seeking = false; seekTo(e); }}
          on:change={seekTo}
          class="seek"
        />
        <span class="time">{fmtTime(duration)}</span>
      </div>
    {/if}
  </div>

  <!-- Right zone: volume + repeat + lyrics + more -->
  <div class="right-zone">
    <!-- Volume -->
    <span class="vol-icon" aria-hidden="true">
      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/><path d="M19.07 4.93a10 10 0 0 1 0 14.14M15.54 8.46a5 5 0 0 1 0 7.07"/></svg>
    </span>
    <input type="range" min="0" max="1" step="0.05"
      bind:value={$volume} class="vol" title="Volume" aria-label="Volume" />

    <!-- Repeat (stub — no action) -->
    <button class="icon-btn" title="Repeat" aria-label="Repeat">
      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="17 1 21 5 17 9"/><path d="M3 11V9a4 4 0 0 1 4-4h14"/><polyline points="7 23 3 19 7 15"/><path d="M21 13v2a4 4 0 0 1-4 4H3"/></svg>
    </button>

    <!-- Lyrics (stub — no action) -->
    <button class="icon-btn" title="Lyrics" aria-label="Lyrics">
      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
    </button>

    <!-- More / quality popover -->
    <div
      class="quality-menu-container"
      tabindex="-1"
      on:focusout={handleQualityMenuBlur}
    >
      <button
        class="icon-btn"
        on:click={toggleQualityMenu}
        title="More options"
        aria-label="More options"
        aria-expanded={showQualityMenu}
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/></svg>
      </button>

      {#if showQualityMenu}
        <div class="quality-menu" role="dialog" aria-label="Playback options" tabindex="-1">
          <label class="menu-label">
            Quality
            <select bind:value={$quality} title="Quality">
              <option value="original">Original</option>
              <option value="320k">320k</option>
              <option value="128k">128k</option>
              <option value="64k">64k</option>
            </select>
          </label>
          <label class="menu-label">
            Format
            <select bind:value={$container} title="Format">
              <option value="mp3">MP3</option>
              <option value="aac">AAC</option>
              <option value="ogg">OGG</option>
              <option value="flac">FLAC</option>
            </select>
          </label>
        </div>
      {/if}
    </div>
  </div>

</div>

<style>
  .player {
    grid-column: 1 / -1;
    background: #1a1a1a;
    border-top: 1px solid #2a2a2a;
    display: grid;
    grid-template-columns: 1fr 420px 1fr;
    align-items: center;
    gap: 16px;
    padding: 8px 20px;
    min-height: 80px;
  }

  /* ── Left zone ── */
  .left-zone {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
  }

  .track-info {
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .title {
    font-weight: 600;
    white-space: normal;
    word-break: break-word;
  }

  .artist {
    font-size: 0.8rem;
    color: #888888;
  }

  .muted { color: #555555; }

  .live-badge {
    display: inline-block;
    background: #e53e3e;
    color: #fff;
    font-size: 0.6rem;
    font-weight: 700;
    letter-spacing: 0.05em;
    padding: 1px 5px;
    border-radius: 3px;
    vertical-align: middle;
    margin-right: 4px;
  }

  .live-row { justify-content: center; gap: 8px; }

  .live-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #555;
  }
  .live-dot.on {
    background: #e53e3e;
    animation: live-pulse 1.4s ease-in-out infinite;
  }
  .live-text { min-width: auto; }

  @keyframes live-pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.3; }
  }

  /* ── Center zone ── */
  .center-zone {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
  }

  .controls {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .seek-row {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
  }

  .seek {
    flex: 1;
    accent-color: #ffffff;
  }

  .time {
    font-size: 0.75rem;
    color: #888888;
    min-width: 36px;
    text-align: center;
  }

  /* ── Right zone ── */
  .right-zone {
    display: flex;
    align-items: center;
    gap: 8px;
    justify-content: flex-end;
  }

  .vol-icon {
    color: #888888;
    display: flex;
    align-items: center;
  }

  .vol {
    width: 80px;
    accent-color: #ffffff;
  }

  /* ── Buttons ── */
  .icon-btn {
    background: transparent;
    border: none;
    padding: 6px;
    cursor: pointer;
    color: #cccccc;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
    transition: color 0.15s;
  }

  .icon-btn:hover {
    color: #ffffff;
  }

  .icon-btn:disabled {
    color: #444444;
    cursor: default;
  }

  .play-btn {
    background: #333333;
    border-radius: 50%;
    width: 42px;
    height: 42px;
    padding: 0;
  }

  .play-btn:hover {
    background: #444444;
  }

  .heart-btn.favorited {
    color: #e53e3e;
  }

  /* ── Quality popover ── */
  .quality-menu-container {
    position: relative;
  }

  .quality-menu {
    position: absolute;
    bottom: calc(100% + 8px);
    right: 0;
    background: #2a2a2a;
    border: 1px solid #3a3a3a;
    border-radius: 8px;
    padding: 12px 14px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    min-width: 160px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.5);
    z-index: 100;
  }

  .menu-label {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 0.75rem;
    color: #aaaaaa;
  }

  .menu-label select {
    font-size: 0.8rem;
    padding: 4px 6px;
    background: #1a1a1a;
    border: 1px solid #3a3a3a;
    border-radius: 4px;
    color: #eeeeee;
  }
</style>
