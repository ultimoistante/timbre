<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client.js';
  import { player } from '$lib/stores/player.js';

  let stations = [];
  let loading = true;
  let search = '';

  let modal = false;        // add/edit modal open
  let editing = null;       // station being edited, or null for new
  let saving = false;
  let probing = false;
  let probeMsg = '';
  let form = blankForm();

  onMount(load);

  function blankForm() {
    return { name: '', url: '', genre: '', homepage: '', favicon: '' };
  }

  async function load() {
    loading = true;
    try {
      stations = (await api.get('/streams')) ?? [];
    } catch (e) {
      stations = [];
    } finally {
      loading = false;
    }
  }

  function openNew() {
    editing = null;
    form = blankForm();
    probeMsg = '';
    modal = true;
  }

  function openEdit(st) {
    editing = st;
    form = { name: st.name, url: st.url, genre: st.genre || '', homepage: st.homepage || '', favicon: st.favicon || '' };
    probeMsg = '';
    modal = true;
  }

  // Detect station metadata (name, genre, homepage, logo) from the stream URL
  // and overwrite the form fields with the probe result.
  async function probe() {
    if (!form.url.trim()) return;
    probing = true;
    probeMsg = '';
    try {
      const info = await api.get('/streams/probe?url=' + encodeURIComponent(form.url.trim()));
      form.name = info.name || '';
      form.genre = info.genre || '';
      form.homepage = info.homepage || '';
      form.favicon = info.favicon || '';
      const found = [info.name, info.genre, info.homepage, info.favicon].filter(Boolean).length;
      probeMsg = found ? 'Detected station info' : 'No metadata found';
    } catch (e) {
      probeMsg = 'Could not probe stream';
    } finally {
      probing = false;
    }
  }

  async function save() {
    if (!form.name.trim() || !form.url.trim()) return;
    saving = true;
    try {
      const body = {
        name: form.name.trim(),
        url: form.url.trim(),
        genre: form.genre.trim(),
        homepage: form.homepage.trim(),
        favicon: form.favicon.trim()
      };
      if (editing) {
        await api.patch('/streams/' + editing.id, body);
      } else {
        await api.post('/streams', body);
      }
      modal = false;
      await load();
    } catch (e) {
      console.error(e);
      alert(e.message || 'Failed to save station');
    } finally {
      saving = false;
    }
  }

  async function remove(st) {
    if (!confirm(`Delete "${st.name}"?`)) return;
    try {
      await api.delete('/streams/' + st.id);
      await load();
    } catch (e) {
      console.error(e);
    }
  }

  function play(st) {
    player.playStream(st);
  }

  $: filtered = stations.filter(s =>
    s.name.toLowerCase().includes(search.toLowerCase()) ||
    (s.genre || '').toLowerCase().includes(search.toLowerCase())
  );
</script>

<div class="page">
  <div class="page-header">
    <div>
      <h1>Streams</h1>
      {#if !loading}
        <p class="subtitle">{stations.length} web radio{stations.length !== 1 ? 's' : ''} saved</p>
      {/if}
    </div>
    <button class="new-btn" on:click={openNew}>
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
      Add Stream
    </button>
  </div>

  <input class="search" placeholder="Search streams" bind:value={search} />

  {#if loading}
    <p class="muted">Loading…</p>
  {:else if stations.length === 0}
    <div class="empty">
      <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="2"/><path d="M4.93 19.07a10 10 0 0 1 0-14.14M7.76 16.24a6 6 0 0 1 0-8.49M16.24 7.76a6 6 0 0 1 0 8.49M19.07 4.93a10 10 0 0 1 0 14.14"/></svg>
      <p>No streams yet</p>
      <button class="new-btn sm" on:click={openNew}>Add your first web radio</button>
    </div>
  {:else}
    <ul class="list">
      {#each filtered as st (st.id)}
        <li class="row">
          <button class="play-btn" on:click={() => play(st)} title="Play" aria-label="Play {st.name}">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="currentColor" stroke="none"><polygon points="5 3 19 12 5 21 5 3"/></svg>
          </button>
          <div class="logo">
            {#if st.favicon}
              <img src={st.favicon} alt="" on:error={(e) => e.target.style.display = 'none'} />
            {:else}
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="2"/><path d="M7.76 16.24a6 6 0 0 1 0-8.49M16.24 7.76a6 6 0 0 1 0 8.49"/></svg>
            {/if}
          </div>
          <div class="info">
            <span class="name">{st.name}</span>
            <span class="meta">{st.genre || 'Web radio'}</span>
          </div>
          <div class="actions">
            {#if st.homepage}
              <a class="icon-btn" href={st.homepage} target="_blank" rel="noopener noreferrer" title="Homepage" aria-label="Homepage">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
              </a>
            {/if}
            <button class="icon-btn" on:click={() => openEdit(st)} title="Edit" aria-label="Edit">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4z"/></svg>
            </button>
            <button class="icon-btn danger" on:click={() => remove(st)} title="Delete" aria-label="Delete">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
            </button>
          </div>
        </li>
      {/each}
    </ul>
  {/if}
</div>

{#if modal}
  <div class="modal-bg">
    <div class="modal" role="dialog">
      <h3>{editing ? 'Edit Stream' : 'Add Stream'}</h3>
      <label>Stream URL
        <div class="url-row">
          <input bind:value={form.url} placeholder="https://…/stream.mp3" />
          <button class="detect-btn" on:click={probe} disabled={probing || !form.url.trim()} title="Detect station info">
            {#if probing}
              <span class="spinner" aria-label="Detecting"></span>
            {:else}
              Detect
            {/if}
          </button>
        </div>
      </label>
      {#if probeMsg}<span class="probe-msg">{probeMsg}</span>{/if}
      <label>Name<input bind:value={form.name} placeholder="Station name" /></label>
      <label>Genre<input bind:value={form.genre} placeholder="(optional)" /></label>
      <label>Homepage<input bind:value={form.homepage} placeholder="(optional)" /></label>
      <label>Logo URL<input bind:value={form.favicon} placeholder="(optional)" /></label>
      {#if form.favicon}
        <div class="logo-preview">
          <img src={form.favicon} alt="logo preview" on:error={(e) => e.target.style.visibility = 'hidden'} />
          <span class="muted">Logo preview</span>
        </div>
      {/if}
      <div class="modal-btns">
        <button on:click={save} disabled={probing || saving || !form.name.trim() || !form.url.trim()}>
          {saving ? 'Saving…' : (editing ? 'Save' : 'Add')}
        </button>
        <button class="cancel" on:click={() => modal = false} disabled={probing}>Cancel</button>
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

  .list { list-style: none; display: flex; flex-direction: column; gap: 6px; }

  .row {
    display: flex;
    align-items: center;
    gap: 12px;
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
    padding: 8px 12px;
  }
  .row:hover { background: #1e1e1e; }

  .play-btn {
    background: #333333;
    border-radius: 50%;
    width: 36px;
    height: 36px;
    padding: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #ffffff;
    flex-shrink: 0;
  }
  .play-btn:hover { background: #444444; }

  .logo {
    width: 40px;
    height: 40px;
    border-radius: 6px;
    background: #222222;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #888888;
    overflow: hidden;
    flex-shrink: 0;
  }
  .logo img { width: 100%; height: 100%; object-fit: cover; }

  .info { display: flex; flex-direction: column; min-width: 0; flex: 1; }
  .name {
    font-size: 0.9rem;
    font-weight: 600;
    color: #ffffff;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .meta { font-size: 0.75rem; color: #888888; }

  .actions { display: flex; align-items: center; gap: 4px; flex-shrink: 0; }
  .icon-btn {
    background: transparent;
    border: none;
    padding: 6px;
    color: #aaaaaa;
    display: flex;
    align-items: center;
    border-radius: 4px;
  }
  .icon-btn:hover { background: #2a2a2a; color: #ffffff; }
  .icon-btn.danger:hover { color: #e53e3e; }

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
    padding: 28px; min-width: 360px; display: flex; flex-direction: column; gap: 12px;
  }
  .modal h3 { font-size: 1.1rem; color: #ffffff; margin: 0; }
  .modal label { display: flex; flex-direction: column; gap: 4px; font-size: 0.75rem; color: #aaaaaa; }
  .modal input { width: 100%; }
  .modal-btns { display: flex; gap: 8px; justify-content: flex-end; margin-top: 6px; }
  .cancel { background: #222222; }

  .url-row { display: flex; gap: 6px; }
  .url-row input { flex: 1; }
  .detect-btn { background: #2a2a2a; white-space: nowrap; flex-shrink: 0; }
  .detect-btn:hover:not(:disabled) { background: #333333; }
  .detect-btn:disabled { color: #555; cursor: default; }
  .detect-btn .spinner {
    display: inline-block;
    width: 13px;
    height: 13px;
    border: 2px solid #555;
    border-top-color: #fff;
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
    vertical-align: middle;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
  .probe-msg { font-size: 0.72rem; color: #888888; margin-top: -6px; }

  .logo-preview { display: flex; align-items: center; gap: 8px; }
  .logo-preview img {
    width: 40px; height: 40px; border-radius: 6px;
    object-fit: cover; background: #222222;
  }
</style>
