<script>
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { api } from '$lib/api/client.js';
  import { player, currentTrack, playing } from '$lib/stores/player.js';
  import TrackEditModal from '$lib/components/TrackEditModal.svelte';

  let tracks = [];
  let loading = true;
  let artError = false;

  // Tag edit modal state
  let editTrack = null;   // track being edited (track mode)
  let editAlbum = false;  // album-mode modal open
  let artVersion = 0;     // cache-bust for album art after an update

  function onArtUpdated() {
    artError = false;
    artVersion = Date.now();
  }

  function openEditTrack(track, ev) {
    ev.stopPropagation();
    menuTrackId = null;
    editTrack = track;
  }

  function onTrackSaved(ev) {
    const updated = ev.detail;
    editTrack = null;
    if (updated.albumHash !== hash) {
      // Track moved to a different album — drop it from this view.
      tracks = tracks.filter((t) => t.id !== updated.id);
    } else {
      tracks = tracks.map((t) => (t.id === updated.id ? updated : t));
    }
  }

  function onAlbumSaved(ev) {
    const { albumHash: newHash } = ev.detail;
    editAlbum = false;
    if (newHash && newHash !== hash) {
      // goto() re-uses this component instance (same route, new param), so
      // the hash-keyed loader below picks up the navigation and refetches.
      goto('/library/' + newHash);
    } else {
      reload();
    }
  }

  async function reload() {
    try {
      tracks = await api.get('/albums/' + hash) ?? [];
    } catch (e) {
      tracks = [];
    }
  }

  // Playlist picker state
  let menuTrackId = null;   // id of track whose "add to playlist" menu is open
  let playlists = [];        // cached playlist summaries
  let playlistsLoaded = false;
  let addError = '';

  async function openMenu(track, ev) {
    ev.stopPropagation();
    if (menuTrackId === track.id) { menuTrackId = null; return; }
    menuTrackId = track.id;
    addError = '';
    if (!playlistsLoaded) {
      try {
        playlists = await api.get('/playlists') ?? [];
        playlistsLoaded = true;
      } catch (e) {
        addError = 'Failed to load playlists';
      }
    }
  }

  async function addToPlaylist(playlistId, track, ev) {
    ev.stopPropagation();
    try {
      await api.post(`/playlists/${playlistId}/tracks`, { trackIds: [track.id] });
      menuTrackId = null;
      playlistsLoaded = false; // refresh counts next open
    } catch (e) {
      addError = 'Failed to add';
    }
  }

  async function createAndAdd(track, ev) {
    ev.stopPropagation();
    const name = prompt('New playlist name');
    if (!name) return;
    try {
      const pl = await api.post('/playlists', { name });
      await api.post(`/playlists/${pl.id}/tracks`, { trackIds: [track.id] });
      menuTrackId = null;
      playlistsLoaded = false;
    } catch (e) {
      addError = 'Failed to create';
    }
  }

  $: hash = $page.params.hash;
  $: album = tracks[0] ? {
    name: tracks[0].album || 'Unknown Album',
    artist: tracks[0].albumArtist || tracks[0].artists || 'Unknown Artist',
    year: tracks[0].year || null,
    hash: tracks[0].albumHash,
  } : null;

  // Editing name/artist changes the derived albumHash, and onAlbumSaved
  // navigates to the new one — but SvelteKit reuses this component instance
  // for a same-route navigation, so onMount alone would never refetch. Key
  // the load off `hash` instead, so every hash change (initial load or a
  // post-edit navigation) reloads.
  let loadedHash = null;
  $: if (hash && hash !== loadedHash) loadForHash(hash);

  async function loadForHash(h) {
    loading = true;
    try {
      tracks = await api.get('/albums/' + h) ?? [];
    } catch (e) {
      tracks = [];
    } finally {
      loading = false;
      loadedHash = h;
    }
  }

  function playFrom(idx) {
    player.play(tracks, idx);
  }

  function download(track, ev) {
    ev?.stopPropagation();
    window.open(`/api/download?path=${encodeURIComponent(track.relPath)}`, '_blank');
  }

  function downloadAlbum() {
    api.downloadZip(tracks.map(t => t.relPath), album?.name || 'album');
  }

  function fmtDur(sec) {
    if (!sec || isNaN(sec)) return '';
    const s = Math.round(sec);
    return `${Math.floor(s / 60)}:${(s % 60).toString().padStart(2, '0')}`;
  }

  function fmtTotalDur(secs) {
    const total = Math.round(secs);
    const h = Math.floor(total / 3600);
    const m = Math.floor((total % 3600) / 60);
    if (h > 0) return `${h} hr ${m} min`;
    return `${m} min`;
  }

  $: totalDuration = tracks.reduce((s, t) => s + (t.duration || 0), 0);
  $: isCurrentAlbum = $currentTrack?.albumHash === hash;
</script>

<svelte:window on:click={() => { if (menuTrackId !== null) menuTrackId = null; }} />

<div class="album-page">
  <button class="back-btn" on:click={() => goto('/library')}>
    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
    Library
  </button>

  {#if loading}
    <p class="muted">Loading…</p>
  {:else if tracks.length === 0}
    <p class="muted">Album not found.</p>
  {:else}
    <div class="album-header">
      <div class="art-wrap">
        {#if !artError && album}
          <img
            src="/api/albums/{album.hash}/art{artVersion ? '?v=' + artVersion : ''}"
            alt=""
            class="art-img"
            on:error={() => artError = true}
          />
        {:else}
          <div class="art-placeholder">
            <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>
          </div>
        {/if}
      </div>

      <div class="album-meta">
        <p class="meta-label">Album</p>
        <h1 class="album-title">{album?.name}</h1>
        <p class="album-artist">{album?.artist}{album?.year ? ' · ' + album.year : ''}</p>
        <p class="album-stats">{tracks.length} tracks · {fmtTotalDur(totalDuration)}</p>
        <div class="album-actions">
          <button class="play-all-btn" on:click={() => playFrom(0)}>
            {#if isCurrentAlbum && $playing}
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="currentColor" stroke="none"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>
              Pause
            {:else}
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="currentColor" stroke="none"><polygon points="5 3 19 12 5 21 5 3"/></svg>
              Play
            {/if}
          </button>
          <button class="edit-album-btn" on:click={() => editAlbum = true}>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
            Edit
          </button>
          <button class="edit-album-btn" on:click={downloadAlbum}>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
            Download
          </button>
        </div>
      </div>
    </div>

    <ul class="track-list" role="listbox" aria-label="Tracks">
      {#each tracks as track, i}
        {@const isPlaying = $currentTrack?.id === track.id}
        <li
          class:active={isPlaying}
          on:click={() => playFrom(i)}
          on:keydown={e => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), playFrom(i))}
          role="option"
          aria-selected={isPlaying}
          tabindex="0"
        >
          <span class="tn">
            {#if isPlaying}
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="currentColor" stroke="none"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>
            {:else}
              {track.trackNo || '—'}
            {/if}
          </span>
          <span class="t-title">{track.title || 'Unknown'}</span>
          <span class="t-artist">{track.artists || ''}</span>
          <span class="t-dur">{fmtDur(track.duration)}</span>

          <div class="add-wrap">
            <button
              class="add-btn"
              title="Edit tags"
              aria-label="Edit tags"
              on:click={(e) => openEditTrack(track, e)}
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
            </button>
            <button
              class="add-btn"
              title="Download"
              aria-label="Download"
              on:click={(e) => download(track, e)}
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
            </button>
            <button
              class="add-btn"
              title="Add to playlist"
              aria-label="Add to playlist"
              on:click={(e) => openMenu(track, e)}
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            </button>

            {#if menuTrackId === track.id}
              <div class="pl-menu" role="menu">
                <p class="pl-menu-title">Add to playlist</p>
                {#if addError}
                  <p class="pl-menu-err">{addError}</p>
                {/if}
                {#each playlists as pl}
                  <button class="pl-menu-item" role="menuitem" on:click={(e) => addToPlaylist(pl.id, track, e)}>
                    {pl.name}
                  </button>
                {/each}
                {#if playlistsLoaded && playlists.length === 0}
                  <p class="pl-menu-empty">No playlists yet</p>
                {/if}
                <button class="pl-menu-item pl-menu-new" role="menuitem" on:click={(e) => createAndAdd(track, e)}>
                  + New playlist
                </button>
              </div>
            {/if}
          </div>
        </li>
      {/each}
    </ul>
  {/if}
</div>

{#if editTrack}
  <TrackEditModal
    mode="track"
    track={editTrack}
    on:saved={onTrackSaved}
    on:close={() => editTrack = null}
  />
{/if}

{#if editAlbum}
  <TrackEditModal
    mode="album"
    {hash}
    track={tracks[0]}
    on:saved={onAlbumSaved}
    on:artUpdated={onArtUpdated}
    on:close={() => editAlbum = false}
  />
{/if}

<style>
  .album-page { display: flex; flex-direction: column; gap: 24px; }

  .back-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    background: none;
    color: #888888;
    padding: 0;
    font-size: 0.9rem;
    width: fit-content;
    transition: color 150ms ease;
  }
  .back-btn:hover { color: #ffffff; background: none; }

  .album-header {
    display: flex;
    gap: 28px;
    align-items: flex-end;
  }

  .art-wrap {
    width: 180px;
    height: 180px;
    flex-shrink: 0;
    border-radius: 8px;
    overflow: hidden;
    background: #222222;
  }

  .art-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }

  .art-placeholder {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #555555;
  }

  .album-meta {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 0;
  }

  .meta-label {
    font-size: 0.75rem;
    color: #888888;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .album-title {
    font-size: 1.8rem;
    font-weight: 700;
    color: #ffffff;
    line-height: 1.1;
  }

  .album-artist {
    font-size: 0.95rem;
    color: #cccccc;
  }

  .album-stats {
    font-size: 0.82rem;
    color: #888888;
  }

  .album-actions {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-top: 8px;
  }

  .play-all-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 24px;
    background: #ffffff;
    color: #000000;
    border-radius: 24px;
    font-weight: 600;
    font-size: 0.9rem;
    width: fit-content;
    transition: background 150ms ease;
  }
  .play-all-btn:hover { background: #dddddd; }

  .edit-album-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 10px 18px;
    background: none;
    border: 1px solid #3a3a3a;
    color: #cccccc;
    border-radius: 24px;
    font-weight: 600;
    font-size: 0.9rem;
    transition: border-color 150ms ease, color 150ms ease;
  }
  .edit-album-btn:hover { border-color: #888888; color: #ffffff; background: none; }

  .track-list {
    list-style: none;
    display: flex;
    flex-direction: column;
  }

  .track-list li {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 12px;
    border-radius: 4px;
    cursor: pointer;
    transition: background 100ms ease;
  }
  .track-list li:hover { background: #222222; }
  .track-list li.active {
    background: #1e3a2f;
    color: #1db954;
  }
  .track-list li.active .t-artist { color: #1db954; opacity: 0.7; }
  .track-list li.active .tn { color: #1db954; }

  .tn {
    min-width: 28px;
    text-align: right;
    color: #555555;
    font-size: 0.85rem;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: flex-end;
  }

  .t-title {
    font-weight: 600;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .t-artist {
    font-size: 0.8rem;
    color: #888888;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .t-dur {
    font-size: 0.8rem;
    color: #888888;
    text-align: right;
    min-width: 40px;
    flex-shrink: 0;
  }

  .add-wrap {
    position: relative;
    flex-shrink: 0;
    display: flex;
    align-items: center;
  }

  .add-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    background: none;
    color: #888888;
    padding: 4px;
    border-radius: 50%;
    opacity: 0;
    transition: color 120ms ease, opacity 120ms ease, background 120ms ease;
  }
  .track-list li:hover .add-btn { opacity: 1; }
  .add-btn:hover { color: #ffffff; background: #333333; }

  .pl-menu {
    position: absolute;
    top: 100%;
    right: 0;
    margin-top: 4px;
    z-index: 20;
    min-width: 180px;
    max-height: 280px;
    overflow-y: auto;
    background: #2a2a2a;
    border: 1px solid #3a3a3a;
    border-radius: 8px;
    padding: 6px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .pl-menu-title {
    font-size: 0.72rem;
    color: #888888;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    padding: 4px 8px;
  }

  .pl-menu-err { font-size: 0.78rem; color: #e06c6c; padding: 2px 8px; }
  .pl-menu-empty { font-size: 0.8rem; color: #888888; padding: 4px 8px; }

  .pl-menu-item {
    text-align: left;
    background: none;
    color: #e0e0e0;
    font-size: 0.85rem;
    padding: 7px 8px;
    border-radius: 4px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    transition: background 100ms ease;
  }
  .pl-menu-item:hover { background: #3a3a3a; }
  .pl-menu-new { color: #1db954; border-top: 1px solid #3a3a3a; margin-top: 2px; }

  .muted { color: #888888; }
</style>
