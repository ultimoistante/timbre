<script>
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { api } from '$lib/api/client.js';
  import { player, currentTrack, playing } from '$lib/stores/player.js';

  let playlist = null;
  let tracks = [];
  let loading = true;
  let editModal = false;
  let editName = '', editDesc = '', editPinned = false;
  let saving = false;
  let deleteConfirm = false;

  $: id = $page.params.id;

  onMount(load);

  async function load() {
    loading = true;
    try {
      const data = await api.get('/playlists/' + id);
      playlist = { id: data.id, name: data.name, description: data.description, pinned: data.pinned };
      tracks = data.tracks ?? [];
    } catch (e) {
      playlist = null;
    } finally {
      loading = false;
    }
  }

  function playFrom(idx) {
    player.play(tracks, idx);
  }

  async function saveEdit() {
    saving = true;
    try {
      await api.put('/playlists/' + id, { name: editName, description: editDesc, pinned: editPinned });
      playlist = { ...playlist, name: editName, description: editDesc, pinned: editPinned };
      editModal = false;
    } catch (e) {
      console.error(e);
    } finally {
      saving = false;
    }
  }

  async function removeTrack(ptId) {
    await api.delete('/playlists/' + id + '/tracks/' + ptId).catch(console.error);
    await load();
  }

  async function deletePlaylist() {
    await api.delete('/playlists/' + id).catch(console.error);
    goto('/playlists');
  }

  function openEdit() {
    editName = playlist.name;
    editDesc = playlist.description;
    editPinned = playlist.pinned;
    editModal = true;
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
    return h > 0 ? `${h} hr ${m} min` : `${m} min`;
  }

  $: totalDuration = tracks.reduce((s, t) => s + (t.duration || 0), 0);

  // PlaylistTrack ID isn't returned in track objects — we use position index for removal.
  // The backend uses playlist_track.id for deletion. We need to expose pt.id.
  // For now, removal uses track.id (backend matches track_id per playlist).
</script>

<div class="album-page">
  <button class="back-btn" on:click={() => goto('/playlists')}>
    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
    Playlists
  </button>

  {#if loading}
    <p class="muted">Loading…</p>
  {:else if !playlist}
    <p class="muted">Playlist not found.</p>
  {:else}
    <div class="album-header">
      <div class="art-wrap">
        <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>
      </div>

      <div class="album-meta">
        <p class="meta-label">Playlist{playlist.pinned ? ' · Pinned' : ''}</p>
        <h1 class="album-title">{playlist.name}</h1>
        {#if playlist.description}
          <p class="album-artist">{playlist.description}</p>
        {/if}
        <p class="album-stats">{tracks.length} tracks{tracks.length > 0 ? ' · ' + fmtTotalDur(totalDuration) : ''}</p>
        <div class="actions-row">
          {#if tracks.length > 0}
            <button class="play-all-btn" on:click={() => playFrom(0)}>
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="currentColor" stroke="none"><polygon points="5 3 19 12 5 21 5 3"/></svg>
              Play
            </button>
          {/if}
          <button class="icon-action" on:click={openEdit} title="Edit playlist">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
          </button>
          <button class="icon-action danger" on:click={() => deleteConfirm = true} title="Delete playlist">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6"/><path d="M14 11v6"/><path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/></svg>
          </button>
        </div>
      </div>
    </div>

    {#if tracks.length === 0}
      <p class="muted">No tracks yet. Add tracks from the Library.</p>
    {:else}
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
                {i + 1}
              {/if}
            </span>
            <span class="t-title">{track.title || 'Unknown'}</span>
            <span class="t-artist">{track.artists || ''}{track.album ? ' · ' + track.album : ''}</span>
            <span class="t-dur">{fmtDur(track.duration)}</span>
          </li>
        {/each}
      </ul>
    {/if}
  {/if}
</div>

<!-- Edit modal -->
{#if editModal}
  <div class="modal-bg" on:click={() => editModal = false} on:keypress={() => editModal = false} role="button" tabindex="0">
    <div class="modal" on:click|stopPropagation on:keypress|stopPropagation role="dialog" tabindex="-1">
      <h3>Edit Playlist</h3>
      <input bind:value={editName} placeholder="Playlist name" />
      <input bind:value={editDesc} placeholder="Description (optional)" />
      <label class="pin-label">
        <input type="checkbox" bind:checked={editPinned} />
        Pin to top
      </label>
      <div class="modal-btns">
        <button on:click={saveEdit} disabled={saving || !editName.trim()}>{saving ? 'Saving…' : 'Save'}</button>
        <button class="cancel" on:click={() => editModal = false}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

<!-- Delete confirm modal -->
{#if deleteConfirm}
  <div class="modal-bg" on:click={() => deleteConfirm = false} on:keypress={() => deleteConfirm = false} role="button" tabindex="0">
    <div class="modal" on:click|stopPropagation on:keypress|stopPropagation role="dialog" tabindex="-1">
      <h3>Delete "{playlist?.name}"?</h3>
      <p class="modal-warn">This action cannot be undone.</p>
      <div class="modal-btns">
        <button class="danger" on:click={deletePlaylist}>Delete</button>
        <button class="cancel" on:click={() => deleteConfirm = false}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .album-page { display: flex; flex-direction: column; gap: 24px; }

  .back-btn {
    display: flex; align-items: center; gap: 6px;
    background: none; color: #888888; padding: 0; font-size: 0.9rem;
    width: fit-content; transition: color 150ms ease;
  }
  .back-btn:hover { color: #ffffff; background: none; }

  .album-header { display: flex; gap: 28px; align-items: flex-end; }

  .art-wrap {
    width: 180px; height: 180px; flex-shrink: 0;
    border-radius: 8px; background: #222222;
    display: flex; align-items: center; justify-content: center; color: #555555;
  }

  .album-meta { display: flex; flex-direction: column; gap: 6px; min-width: 0; }

  .meta-label { font-size: 0.75rem; color: #888888; text-transform: uppercase; letter-spacing: 0.06em; margin: 0; }
  .album-title { font-size: 1.8rem; font-weight: 700; color: #ffffff; line-height: 1.1; margin: 0; }
  .album-artist { font-size: 0.95rem; color: #cccccc; margin: 0; }
  .album-stats { font-size: 0.82rem; color: #888888; margin: 0; }

  .actions-row { display: flex; align-items: center; gap: 10px; margin-top: 8px; }

  .play-all-btn {
    display: flex; align-items: center; gap: 8px;
    padding: 10px 24px; background: #ffffff; color: #000000;
    border-radius: 24px; font-weight: 600; font-size: 0.9rem;
    transition: background 150ms ease;
  }
  .play-all-btn:hover { background: #dddddd; }

  .icon-action {
    display: flex; align-items: center; justify-content: center;
    width: 44px; height: 44px; border-radius: 50%;
    background: #2a2a2a; color: #aaaaaa;
    transition: color 150ms ease, background 150ms ease;
  }
  .icon-action:hover { color: #ffffff; background: #383838; }
  .icon-action.danger { color: #aaaaaa; }
  .icon-action.danger:hover { color: #f87171; background: #383838; }

  .track-list { list-style: none; display: flex; flex-direction: column; }
  .track-list li {
    display: flex; align-items: center; gap: 12px;
    padding: 8px 12px; border-radius: 4px; cursor: pointer;
    transition: background 100ms ease;
  }
  .track-list li:hover { background: #222222; }
  .track-list li.active { background: #1e3a2f; color: #1db954; }
  .track-list li.active .t-artist { color: #1db954; opacity: 0.7; }
  .track-list li.active .tn { color: #1db954; }

  .tn {
    min-width: 28px; text-align: right; color: #555555; font-size: 0.85rem;
    flex-shrink: 0; display: flex; align-items: center; justify-content: flex-end;
  }
  .t-title { font-weight: 600; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .t-artist { font-size: 0.8rem; color: #888888; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .t-dur { font-size: 0.8rem; color: #888888; text-align: right; min-width: 40px; flex-shrink: 0; }

  .muted { color: #888888; }

  .modal-bg {
    position: fixed; inset: 0; background: rgba(0,0,0,.7);
    display: flex; align-items: center; justify-content: center; z-index: 100;
  }
  .modal {
    background: #1a1a1a; border: 1px solid #2a2a2a; border-radius: 12px;
    padding: 28px; min-width: 320px; display: flex; flex-direction: column; gap: 14px;
  }
  .modal h3 { font-size: 1.1rem; color: #ffffff; margin: 0; }
  .modal-warn { font-size: 0.85rem; color: #888888; margin: 0; }
  .modal input:not([type]) { width: 100%; }
  .pin-label { display: flex; align-items: center; gap: 8px; font-size: 0.9rem; color: #cccccc; cursor: pointer; }
  .pin-label input { width: auto; cursor: pointer; }
  .modal-btns { display: flex; gap: 8px; justify-content: flex-end; }
  .cancel { background: #222222; }
  .danger { background: #7f1d1d; }
  .danger:hover { background: #991b1b; }
</style>
