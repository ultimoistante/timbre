<script>
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api/client.js';

  let albums = [];
  let artErrors = {};
  let loading = true;

  onMount(async () => {
    try {
      albums = await api.get('/recently-added') ?? [];
    } catch (e) {
      albums = [];
    } finally {
      loading = false;
    }
  });
</script>

<div class="home">
  <h2 class="section-title">Recently Added</h2>

  {#if loading}
    <p class="muted">Loading…</p>
  {:else if albums.length === 0}
    <p class="muted">No music yet. Go to <a href="/library">Library</a> to scan your files.</p>
  {:else}
    <div class="grid">
      {#each albums as album}
        <button class="card" on:click={() => goto('/library/' + album.albumHash)}>
          <div class="art">
            {#if !artErrors[album.albumHash]}
              <img
                src="/api/albums/{album.albumHash}/art"
                alt=""
                class="album-art"
                on:error={() => { artErrors[album.albumHash] = true; artErrors = artErrors; }}
              />
            {:else if album.album}
              <span class="initial">{album.album[0]}</span>
            {:else}
              <svg xmlns="http://www.w3.org/2000/svg" width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>
            {/if}
          </div>
          <div class="info">
            <span class="album-name">{album.album || 'Unknown Album'}</span>
            <span class="muted">{album.albumArtist || 'Unknown Artist'}</span>
            <span class="muted">{album.trackCount} track{album.trackCount !== 1 ? 's' : ''}{album.year ? ' · ' + album.year : ''}</span>
          </div>
        </button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .home {
    padding: 0;
  }

  .section-title {
    font-size: 1.1rem;
    font-weight: 600;
    color: #ffffff;
    margin: 0 0 16px 0;
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 16px;
  }

  .card {
    padding: 0;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
    background: #1a1a1a;
    overflow: hidden;
    cursor: pointer;
    text-align: left;
    transition: transform 150ms ease, box-shadow 150ms ease;
  }

  .card:hover {
    transform: scale(1.02);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
  }

  .art {
    aspect-ratio: 1;
    background: linear-gradient(135deg, #222222 0%, #1a1a1a 100%);
    border-radius: 8px 8px 0 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #555555;
    user-select: none;
  }

  .initial {
    font-size: 2rem;
    font-weight: 700;
    color: #555555;
    text-transform: uppercase;
  }

  .album-art {
    width: 100%;
    height: 100%;
    object-fit: cover;
    border-radius: 8px 8px 0 0;
    display: block;
  }

  .info {
    padding: 8px 10px 10px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .album-name {
    font-size: 0.85rem;
    font-weight: 700;
    color: #ffffff;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .muted {
    font-size: 0.75rem;
    color: #888888;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  a {
    color: #ffffff;
  }
</style>
