<script>
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api/client.js';

  let playlists = [];
  let loading = true;
  let search = '';
  let newModal = false;
  let newName = '', newDesc = '';
  let creating = false;

  onMount(loadPlaylists);

  async function loadPlaylists() {
    loading = true;
    try {
      playlists = await api.get('/playlists') ?? [];
    } catch (e) {
      playlists = [];
    } finally {
      loading = false;
    }
  }

  async function createPlaylist() {
    if (!newName.trim()) return;
    creating = true;
    try {
      const pl = await api.post('/playlists', { name: newName.trim(), description: newDesc.trim() });
      newModal = false; newName = ''; newDesc = '';
      goto('/playlists/' + pl.id);
    } catch (e) {
      console.error(e);
    } finally {
      creating = false;
    }
  }

  $: filtered = playlists.filter(p => p.name.toLowerCase().includes(search.toLowerCase()));
  $: pinned = filtered.filter(p => p.pinned);
  $: others = filtered.filter(p => !p.pinned);
</script>

<div class="page">
  <div class="page-header">
    <div>
      <h1>Playlists</h1>
      {#if !loading}
        <p class="subtitle">You have {playlists.length} playlist{playlists.length !== 1 ? 's' : ''} in your library</p>
      {/if}
    </div>
    <button class="new-btn" on:click={() => { newModal = true; newName = ''; newDesc = ''; }}>
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
      New Playlist
    </button>
  </div>

  <input class="search" placeholder="Search playlists" bind:value={search} />

  {#if loading}
    <p class="muted">Loading…</p>
  {:else if playlists.length === 0}
    <div class="empty">
      <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>
      <p>No playlists yet</p>
      <button class="new-btn sm" on:click={() => { newModal = true; newName = ''; newDesc = ''; }}>Create your first playlist</button>
    </div>
  {:else}
    {#if pinned.length > 0}
      <section>
        <h2 class="section-title">Pinned</h2>
        <div class="grid">
          {#each pinned as pl}
            <button class="card" on:click={() => goto('/playlists/' + pl.id)}>
              <div class="art">
                {#if pl.albumHashes.length >= 4}
                  <div class="mosaic">
                    {#each pl.albumHashes.slice(0, 4) as h}
                      <img src="/api/albums/{h}/art" alt="" />
                    {/each}
                  </div>
                {:else if pl.albumHashes.length > 0}
                  <img class="art-single" src="/api/albums/{pl.albumHashes[0]}/art" alt="" />
                {:else}
                  <svg xmlns="http://www.w3.org/2000/svg" width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>
                {/if}
              </div>
              <div class="card-info">
                <span class="card-name">{pl.name}</span>
                <span class="card-meta">{pl.trackCount} Track{pl.trackCount !== 1 ? 's' : ''}</span>
              </div>
            </button>
          {/each}
        </div>
      </section>
    {/if}

    {#if others.length > 0}
      <section>
        <h2 class="section-title">{pinned.length > 0 ? 'Other Playlists' : 'All Playlists'}</h2>
        <div class="grid">
          {#each others as pl}
            <button class="card" on:click={() => goto('/playlists/' + pl.id)}>
              <div class="art">
                {#if pl.albumHashes.length >= 4}
                  <div class="mosaic">
                    {#each pl.albumHashes.slice(0, 4) as h}
                      <img src="/api/albums/{h}/art" alt="" />
                    {/each}
                  </div>
                {:else if pl.albumHashes.length > 0}
                  <img class="art-single" src="/api/albums/{pl.albumHashes[0]}/art" alt="" />
                {:else}
                  <svg xmlns="http://www.w3.org/2000/svg" width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>
                {/if}
              </div>
              <div class="card-info">
                <span class="card-name">{pl.name}</span>
                <span class="card-meta">{pl.trackCount} Track{pl.trackCount !== 1 ? 's' : ''}</span>
              </div>
            </button>
          {/each}
        </div>
      </section>
    {/if}
  {/if}
</div>

{#if newModal}
  <div class="modal-bg" on:click={() => newModal = false} on:keypress={() => newModal = false} role="button" tabindex="0">
    <div class="modal" on:click|stopPropagation on:keypress|stopPropagation role="dialog" tabindex="-1">
      <h3>New Playlist</h3>
      <input bind:value={newName} placeholder="Playlist name" on:keydown={e => e.key === 'Enter' && createPlaylist()} />
      <input bind:value={newDesc} placeholder="Description (optional)" />
      <div class="modal-btns">
        <button on:click={createPlaylist} disabled={creating || !newName.trim()}>
          {creating ? 'Creating…' : 'Create'}
        </button>
        <button class="cancel" on:click={() => newModal = false}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .page { display: flex; flex-direction: column; gap: 24px; }

  .page-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
  }

  h1 { font-size: 2rem; font-weight: 700; color: #ffffff; margin: 0; }
  .subtitle { font-size: 0.85rem; color: #888888; margin-top: 4px; }

  .new-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    background: #ffffff;
    color: #000000;
    border-radius: 20px;
    padding: 8px 16px;
    font-weight: 600;
    font-size: 0.85rem;
    white-space: nowrap;
    flex-shrink: 0;
    transition: background 150ms ease;
  }
  .new-btn:hover { background: #dddddd; }
  .new-btn.sm { margin-top: 4px; }

  .search {
    width: 100%;
    max-width: 380px;
    padding: 10px 14px;
    background: #1e1e1e;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
    color: #ffffff;
    font-size: 0.9rem;
  }

  section { display: flex; flex-direction: column; }

  .section-title {
    font-size: 1.1rem;
    font-weight: 600;
    color: #ffffff;
    margin: 0 0 14px 0;
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 16px;
  }

  .card {
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
    padding: 0;
    overflow: hidden;
    cursor: pointer;
    text-align: left;
    transition: transform 150ms ease, box-shadow 150ms ease;
  }
  .card:hover { transform: scale(1.02); box-shadow: 0 4px 16px rgba(0,0,0,0.4); }

  /* Art area */
  .art {
    aspect-ratio: 1;
    background: linear-gradient(135deg, #222222 0%, #1a1a1a 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    color: #555555;
    overflow: hidden;
    border-radius: 8px 8px 0 0;
  }

  .art-single {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }

  .mosaic {
    width: 100%;
    height: 100%;
    display: grid;
    grid-template-columns: 1fr 1fr;
    grid-template-rows: 1fr 1fr;
  }
  .mosaic img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }

  .card-info {
    padding: 8px 10px 10px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .card-name {
    font-size: 0.85rem;
    font-weight: 600;
    color: #ffffff;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .card-meta { font-size: 0.75rem; color: #888888; }

  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    padding: 60px 0;
    color: #555555;
  }
  .empty p { font-size: 1rem; color: #888888; margin: 0; }

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
  .modal input { width: 100%; }
  .modal-btns { display: flex; gap: 8px; justify-content: flex-end; }
  .cancel { background: #222222; }
</style>
