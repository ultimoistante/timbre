<script>
  import { createEventDispatcher } from 'svelte';
  import { api } from '$lib/api/client.js';

  /** @type {'track' | 'album'} */
  export let mode = 'track';
  /** Track object (track mode) — source of initial field values. */
  export let track = null;
  /** Album hash (album mode). */
  export let hash = '';

  const dispatch = createEventDispatcher();

  // Local editable copy. In album mode only album-level fields are shown.
  let f = {
    title: track?.title ?? '',
    artists: track?.artists ?? '',
    album: track?.album ?? '',
    albumArtist: track?.albumArtist ?? '',
    genres: track?.genres ?? '',
    year: track?.year ?? '',
    trackNo: track?.trackNo ?? '',
    discNo: track?.discNo ?? ''
  };

  let saving = false;
  let error = '';

  // Cover-art search (album mode only)
  let artCandidates = [];
  let artSearched = false;
  let artLoading = false;
  let artError = '';
  let artApplying = '';   // url currently being applied
  let artDone = false;

  async function searchArt() {
    artLoading = true;
    artError = '';
    artSearched = true;
    try {
      artCandidates = await api.get(`/albums/${hash}/art/search`) ?? [];
    } catch (e) {
      artError = e.message || 'Search failed';
      artCandidates = [];
    } finally {
      artLoading = false;
    }
  }

  async function applyArt(url) {
    artApplying = url;
    artError = '';
    try {
      await api.put(`/albums/${hash}/art`, { url });
      artDone = true;
      dispatch('artUpdated');
    } catch (e) {
      artError = e.message || 'Failed to apply';
    } finally {
      artApplying = '';
    }
  }

  function num(v) {
    const n = parseInt(v, 10);
    return isNaN(n) ? 0 : n;
  }

  async function save() {
    saving = true;
    error = '';
    try {
      if (mode === 'album') {
        const body = {
          album: f.album,
          albumArtist: f.albumArtist,
          genres: f.genres,
          year: num(f.year)
        };
        const res = await api.patch(`/albums/${hash}`, body);
        dispatch('saved', res);
      } else {
        const body = {
          title: f.title,
          artists: f.artists,
          album: f.album,
          albumArtist: f.albumArtist,
          genres: f.genres,
          year: num(f.year),
          trackNo: num(f.trackNo),
          discNo: num(f.discNo)
        };
        const res = await api.patch(`/tracks/${track.id}`, body);
        dispatch('saved', res);
      }
    } catch (e) {
      error = e.message || 'Save failed';
    } finally {
      saving = false;
    }
  }

  function close() {
    dispatch('close');
  }

  function onKeydown(e) {
    if (e.key === 'Escape') close();
  }
</script>

<svelte:window on:keydown={onKeydown} />

<div class="backdrop">
  <div class="modal" role="dialog" aria-modal="true" tabindex="-1">
    <h2>{mode === 'album' ? 'Edit album' : 'Edit track'}</h2>

    {#if error}<p class="err">{error}</p>{/if}

    <div class="fields">
      {#if mode === 'track'}
        <label>
          <span>Title</span>
          <input bind:value={f.title} />
        </label>
        <label>
          <span>Artist</span>
          <input bind:value={f.artists} />
        </label>
      {/if}

      <label>
        <span>Album</span>
        <input bind:value={f.album} />
      </label>
      <label>
        <span>Album artist</span>
        <input bind:value={f.albumArtist} />
      </label>
      <label>
        <span>Genre</span>
        <input bind:value={f.genres} />
      </label>

      <div class="row">
        <label>
          <span>Year</span>
          <input type="number" bind:value={f.year} />
        </label>
        {#if mode === 'track'}
          <label>
            <span>Track #</span>
            <input type="number" bind:value={f.trackNo} />
          </label>
          <label>
            <span>Disc #</span>
            <input type="number" bind:value={f.discNo} />
          </label>
        {/if}
      </div>
    </div>

    {#if mode === 'album'}
      <div class="art-section">
        <div class="art-head">
          <span class="art-label">Cover art</span>
          <button class="art-search-btn" on:click={searchArt} disabled={artLoading}>
            {artLoading ? 'Searching…' : 'Search web'}
          </button>
        </div>

        {#if artDone}
          <p class="art-ok">Cover applied to all tracks.</p>
        {/if}
        {#if artError}<p class="err">{artError}</p>{/if}

        {#if artSearched && !artLoading && artCandidates.length === 0 && !artError}
          <p class="art-empty">No covers found.</p>
        {/if}

        {#if artCandidates.length > 0}
          <div class="art-grid">
            {#each artCandidates as cand}
              <button
                class="art-cand"
                title={`${cand.title} — ${cand.artist}`}
                on:click={() => applyArt(cand.url)}
                disabled={artApplying !== ''}
              >
                <img src={cand.thumb} alt="" loading="lazy" />
                {#if artApplying === cand.url}<span class="art-spinner">…</span>{/if}
              </button>
            {/each}
          </div>
        {/if}
      </div>
    {/if}

    <div class="actions">
      <button class="cancel" on:click={close} disabled={saving}>Cancel</button>
      <button class="save" on:click={save} disabled={saving}>
        {saving ? 'Saving…' : 'Save'}
      </button>
    </div>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
    padding: 20px;
  }

  .modal {
    background: #1e1e1e;
    border: 1px solid #3a3a3a;
    border-radius: 12px;
    padding: 24px;
    width: 100%;
    max-width: 440px;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.6);
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  h2 {
    font-size: 1.2rem;
    font-weight: 700;
    color: #ffffff;
  }

  .err {
    color: #e06c6c;
    font-size: 0.85rem;
  }

  .fields {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .row {
    display: flex;
    gap: 12px;
  }
  .row label { flex: 1; }

  label {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  label span {
    font-size: 0.72rem;
    color: #888888;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  input {
    background: #2a2a2a;
    border: 1px solid #3a3a3a;
    border-radius: 6px;
    padding: 8px 10px;
    color: #e0e0e0;
    font-size: 0.9rem;
    width: 100%;
  }
  input:focus {
    outline: none;
    border-color: #1db954;
  }

  .art-section {
    display: flex;
    flex-direction: column;
    gap: 10px;
    border-top: 1px solid #3a3a3a;
    padding-top: 14px;
  }
  .art-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .art-label {
    font-size: 0.72rem;
    color: #888888;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  .art-search-btn {
    background: #2a2a2a;
    border: 1px solid #3a3a3a;
    color: #cccccc;
    padding: 6px 14px;
    border-radius: 16px;
    font-size: 0.8rem;
    font-weight: 600;
  }
  .art-search-btn:hover { color: #ffffff; border-color: #888888; }
  .art-search-btn:disabled { opacity: 0.5; }

  .art-ok { color: #1db954; font-size: 0.82rem; }
  .art-empty { color: #888888; font-size: 0.82rem; }

  .art-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(72px, 1fr));
    gap: 8px;
    max-height: 200px;
    overflow-y: auto;
  }
  .art-cand {
    position: relative;
    padding: 0;
    background: #2a2a2a;
    border: 1px solid #3a3a3a;
    border-radius: 6px;
    overflow: hidden;
    aspect-ratio: 1;
    cursor: pointer;
  }
  .art-cand:hover { border-color: #1db954; }
  .art-cand img { width: 100%; height: 100%; object-fit: cover; display: block; }
  .art-cand:disabled { opacity: 0.6; cursor: default; }
  .art-spinner {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.5);
    color: #ffffff;
    font-size: 1.4rem;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    margin-top: 4px;
  }

  .actions button {
    padding: 9px 20px;
    border-radius: 20px;
    font-weight: 600;
    font-size: 0.88rem;
  }
  .cancel {
    background: none;
    color: #cccccc;
  }
  .cancel:hover { background: #2a2a2a; }
  .save {
    background: #1db954;
    color: #000000;
  }
  .save:hover { background: #1ed760; }
  .save:disabled, .cancel:disabled { opacity: 0.5; cursor: default; }
</style>
