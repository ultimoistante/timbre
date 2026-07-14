<script>
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api/client.js';
  import { player } from '$lib/stores/player.js';

  let albums = [], artists = [], tracks = [], searchResults = [];
  let artErrors = {};
  let view = 'albums';
  let searchQ = '';
  let scanning = false;
  let scanMsg = '';

  onMount(() => { loadAll(); return listenSSE(); });

  async function loadAll() {
    [albums, artists, tracks] = await Promise.all([
      api.get('/albums'),
      api.get('/artists'),
      api.get('/tracks')
    ]);
  }

  function openAlbum(album) {
    goto('/library/' + album.albumHash);
  }

  async function doSearch() {
    if (!searchQ.trim()) { searchResults = []; return; }
    searchResults = await api.get(`/search?q=${encodeURIComponent(searchQ)}`);
  }

  async function startScan() {
    scanning = true; scanMsg = 'Starting scan…';
    await api.post('/scan').catch(e => { scanMsg = e.message; scanning = false; });
  }

  function listenSSE() {
    const es = new EventSource('/api/events', { withCredentials: true });
    es.addEventListener('scan', e => {
      const p = JSON.parse(e.data);
      if (p.finished) { scanning = false; scanMsg = `Done — +${p.added} added, ~${p.updated} updated, -${p.removed} removed`; loadAll(); }
      else { scanMsg = `Scanning… ${p.done}/${p.total} — ${p.current}`; }
    });
    return () => es.close();
  }

  function playTrack(trackList, idx) {
    player.play(trackList, idx);
  }

  function fmtDur(sec) {
    if (!sec || isNaN(sec)) return '';
    const s = Math.round(sec);
    const m = Math.floor(s / 60);
    const ss = (s % 60).toString().padStart(2, '0');
    return `${m}:${ss}`;
  }
</script>

<div class="library">
  <header>
    <div class="tabs">
      {#each ['albums','artists','tracks'] as t}
        <button class:active={view===t} on:click={() => { view=t; }}>
          {t.charAt(0).toUpperCase()+t.slice(1)}
        </button>
      {/each}
    </div>

    <div class="search-row">
      <input placeholder="Search…" bind:value={searchQ} on:input={doSearch} />
      <button on:click={startScan} disabled={scanning}>
        {scanning ? 'Scanning…' : 'Scan'}
      </button>
    </div>
    {#if scanMsg}<p class="scan-msg">{scanMsg}</p>{/if}
  </header>

  {#if searchResults.length}
    <section>
      <h2>Search results</h2>
      <ul class="track-list" role="listbox" aria-label="Search results">
        {#each searchResults as t, i}
          <li
            on:click={() => playTrack(searchResults, i)}
            on:keydown={e => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), playTrack(searchResults, i))}
            role="option"
            aria-selected="false"
            tabindex="0"
          >
            <span class="tn">—</span>
            <span class="t-title">{t.title}</span>
            <span class="t-artist">{t.artists}{t.album ? ' · ' + t.album : ''}</span>
            <span class="t-dur">{fmtDur(t.duration)}</span>
          </li>
        {/each}
      </ul>
    </section>
  {:else if view === 'albums'}
    <div class="grid">
      {#each albums as a}
        <div class="card album-card" on:click={() => openAlbum(a)} on:keypress={() => openAlbum(a)} role="button" tabindex="0">
          <div class="album-placeholder">
            {#if !artErrors[a.albumHash]}
              <img
                src="/api/albums/{a.albumHash}/art"
                alt=""
                class="album-art"
                on:error={() => { artErrors[a.albumHash] = true; artErrors = artErrors; }}
              />
            {:else if a.album}
              <span class="placeholder-letter">{a.album[0].toUpperCase()}</span>
            {:else}
              <svg xmlns="http://www.w3.org/2000/svg" width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>
            {/if}
          </div>
          <div class="card-text">
            <p class="album-title">{a.album || 'Unknown'}</p>
            <p class="album-artist">{a.albumArtist || '—'}</p>
            <p class="album-meta">{a.trackCount} tracks{a.year ? ' · ' + a.year : ''}</p>
          </div>
        </div>
      {/each}
    </div>

  {:else if view === 'artists'}
    <div class="grid">
      {#each artists as a}
        <div class="card artist-card">
          <div class="artist-avatar">
            {#if a.name}
              <span class="placeholder-letter">{a.name[0].toUpperCase()}</span>
            {:else}
              <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
            {/if}
          </div>
          <p class="album-title">{a.name || 'Unknown'}</p>
          <p class="album-meta">{a.albumCount} albums · {a.trackCount} tracks</p>
        </div>
      {/each}
    </div>

  {:else if view === 'tracks'}
    <ul class="track-list" role="listbox" aria-label="Tracks">
      {#each tracks as t, i}
        <li
          on:click={() => playTrack(tracks, i)}
          on:keydown={e => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), playTrack(tracks, i))}
          role="option"
          aria-selected="false"
          tabindex="0"
        >
          <span class="tn">{t.trackNo || '—'}</span>
          <span class="t-title">{t.title}</span>
          <span class="t-artist">{t.artists}</span>
          <span class="t-dur">{fmtDur(t.duration)}</span>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .library { display:flex; flex-direction:column; gap:20px; }
  header { display:flex; flex-direction:column; gap:10px; }
  .tabs { display:flex; gap:6px; }
  .tabs button { background:#222222; }
  .tabs button.active { background:#333333; }
  .search-row { display:flex; gap:10px; }
  .search-row input { flex:1; }
  .scan-msg { font-size:0.82rem; color:#888888; }

  .grid {
    display:grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap:16px;
  }

  /* Album card */
  .card {
    background:#1a1a1a;
    border:1px solid #2a2a2a;
    border-radius:8px;
    padding:0;
    cursor:pointer;
    transition: transform 150ms ease, box-shadow 150ms ease;
    overflow:hidden;
  }
  .card:hover {
    transform: scale(1.02);
    box-shadow: 0 4px 16px rgba(0,0,0,0.4);
  }

  .album-placeholder {
    aspect-ratio: 1/1;
    width:100%;
    background: linear-gradient(135deg, #222222 0%, #1a1a1a 100%);
    border-radius:8px 8px 0 0;
    display:flex;
    align-items:center;
    justify-content:center;
    color:#555555;
  }
  .album-placeholder svg { color:#555555; }
  .album-art {
    width: 100%;
    height: 100%;
    object-fit: cover;
    border-radius: 8px 8px 0 0;
    display: block;
  }

  .card-text {
    padding:10px 10px;
  }

  .album-title { font-weight:600; font-size:0.85rem; margin-bottom:4px; }
  .album-artist { font-size:0.75rem; color:#888888; margin-bottom:2px; }
  .album-meta { font-size:0.75rem; color:#888888; }

  /* Artist card */
  .artist-card {
    padding:16px;
    display:flex;
    flex-direction:column;
    align-items:center;
    gap:8px;
    text-align:center;
  }

  .artist-avatar {
    width:80px;
    height:80px;
    border-radius:50%;
    background:#222222;
    display:flex;
    align-items:center;
    justify-content:center;
    color:#555555;
  }
  .artist-avatar svg { color:#555555; }

  .placeholder-letter { font-size:2rem; color:#555555; line-height:1; }
  .artist-avatar .placeholder-letter { font-size:1.5rem; }

  /* Track list */
  .track-list { list-style:none; display:flex; flex-direction:column; gap:4px; }
  .track-list li {
    display:flex; align-items:center; gap:12px;
    padding:8px 12px;
    background:transparent;
    border:none;
    border-radius:4px;
    cursor:pointer;
  }
  .track-list li:hover { background:#222222; }
  .tn { min-width:28px; text-align:right; color:#555555; font-size:0.85rem; flex-shrink:0; }
  .t-title { font-weight:600; flex:1; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
  .t-artist { font-size:0.8rem; color:#888888; flex:1; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
  .t-dur { font-size:0.8rem; color:#888888; text-align:right; min-width:40px; flex-shrink:0; }

  h2 { font-size:1.2rem; }
</style>
