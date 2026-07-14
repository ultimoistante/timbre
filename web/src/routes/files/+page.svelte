<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client.js';
  import TreePicker from '$lib/components/TreePicker.svelte';

  let currentPath = '';
  let entries = [];
  let selected = new Set();
  let dragging = false;
  let error = '';

  // Modals
  let mkdirModal = false, mkdirName = '';
  let renameModal = false, renameTarget = null, renameVal = '';
  let moveModal = false, moveDst = '';
  let copyModal = false, copyDst = '';
  let deleteModal = false;

  onMount(() => loadDir(''));

  async function loadDir(path) {
    error = '';
    const data = await api.get(`/fs/list?path=${encodeURIComponent(path)}`).catch(e => { error = e.message; return null; });
    if (!data) return;
    currentPath = path;
    entries = data.entries || [];
    selected.clear();
    selected = selected;
  }

  function nav(entry) {
    if (entry.isDir) loadDir(join(currentPath, entry.name));
  }

  function breadcrumbs(path) {
    const parts = path ? path.split('/') : [];
    const crumbs = [{ label: 'Home', path: '' }];
    parts.forEach((p, i) => crumbs.push({ label: p, path: parts.slice(0, i+1).join('/') }));
    return crumbs;
  }

  function toggleSelect(name) {
    if (selected.has(name)) selected.delete(name); else selected.add(name);
    selected = new Set(selected);
  }

  function join(...parts) { return parts.filter(Boolean).join('/'); }

  $: crumbs = breadcrumbs(currentPath);

  // Upload
  function handleDrop(e) {
    dragging = false;
    const files = [...e.dataTransfer.files];
    if (files.length) upload(files);
  }
  async function upload(files) {
    await api.upload(currentPath, files).catch(e => error = e.message);
    loadDir(currentPath);
  }
  function openFilePicker() {
    const input = document.createElement('input');
    input.type = 'file'; input.multiple = true;
    input.onchange = () => upload([...input.files]);
    input.click();
  }

  // CRUD
  async function doMkdir() {
    await api.post('/fs/mkdir', { path: join(currentPath, mkdirName) }).catch(e => error = e.message);
    mkdirModal = false; mkdirName = '';
    loadDir(currentPath);
  }

  async function doRename() {
    await api.post('/fs/rename', { path: join(currentPath, renameTarget), newName: renameVal }).catch(e => error = e.message);
    renameModal = false; renameTarget = null; renameVal = '';
    loadDir(currentPath);
  }

  async function doDelete() {
    deleteModal = true;
  }

  async function doDeleteConfirmed() {
    deleteModal = false;
    for (const name of selected) {
      await api.post('/fs/delete', { path: join(currentPath, name) }).catch(e => error = e.message);
    }
    loadDir(currentPath);
  }

  async function doMove() {
    for (const name of selected) {
      const src = join(currentPath, name);
      const dst = join(moveDst, name);
      await api.post('/fs/move', { src, dst }).catch(e => error = e.message);
    }
    moveModal = false; moveDst = '';
    loadDir(currentPath);
  }

  async function doCopy() {
    for (const name of selected) {
      const src = join(currentPath, name);
      const dst = join(copyDst, name);
      await api.post('/fs/copy', { src, dst }).catch(e => error = e.message);
    }
    copyModal = false; copyDst = '';
    loadDir(currentPath);
  }

  function download(name) {
    const path = join(currentPath, name);
    window.open(`/api/download?path=${encodeURIComponent(path)}`, '_blank');
  }

  function downloadSelected() {
    for (const name of selected) download(name);
  }

  function fmtSize(bytes) {
    if (!bytes) return '';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024*1024) return `${(bytes/1024).toFixed(1)} KB`;
    return `${(bytes/1024/1024).toFixed(1)} MB`;
  }
</script>

<div class="files-page">
  <div class="toolbar">
    <nav class="breadcrumb" aria-label="File path">
      {#each crumbs as crumb, i (crumb.path)}
        {#if i > 0}<span class="sep" aria-hidden="true">›</span>{/if}
        {#if i === crumbs.length - 1}
          <span class="crumb crumb-current" aria-current="page">{crumb.label}</span>
        {:else}
          <button class="crumb" on:click={() => loadDir(crumb.path)}>{crumb.label}</button>
        {/if}
      {/each}
    </nav>

    <div class="actions">
      <button on:click={openFilePicker}>Upload</button>
      <button on:click={() => { mkdirModal=true; mkdirName=''; }}>New Folder</button>
      {#if selected.size > 0}
        <button on:click={downloadSelected}>Download</button>
        <button on:click={() => { renameModal=true; renameTarget=[...selected][0]; renameVal=''; }} disabled={selected.size !== 1}>Rename</button>
        <button on:click={() => { moveModal=true; moveDst=''; }}>Move</button>
        <button on:click={() => { copyModal=true; copyDst=''; }}>Copy</button>
        <button class="danger" on:click={doDelete}>Delete</button>
      {/if}
    </div>
  </div>

  {#if error}<p class="error">{error}</p>{/if}

  <!-- Drop zone -->
  <div
    class="drop-zone"
    class:dragging
    on:dragover|preventDefault={() => dragging = true}
    on:dragleave={() => dragging = false}
    on:drop|preventDefault={handleDrop}
    role="region"
    aria-label="Drop files here to upload"
  >
    {#if dragging}
      <div class="drop-hint">Drop files to upload</div>
    {/if}

    <table class="file-table">
      <thead>
        <tr>
          <th><input type="checkbox" on:change={e => { if (e.target.checked) selected = new Set(entries.map(e => e.name)); else selected = new Set(); }} /></th>
          <th>Name</th>
          <th>Size</th>
          <th>Modified</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        {#each entries as entry}
          <tr class:dir={entry.isDir} class:sel={selected.has(entry.name)}>
            <td><input type="checkbox" checked={selected.has(entry.name)} on:change={() => toggleSelect(entry.name)} /></td>
            <td>
              <button class="name-btn" on:click={() => nav(entry)}>
                {#if entry.isDir}
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="vertical-align:-2px;margin-right:6px;color:#888888"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
                {:else}
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="vertical-align:-2px;margin-right:6px;color:#555555"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>
                {/if}
                {entry.name}
              </button>
            </td>
            <td class="meta">{entry.isDir ? '' : fmtSize(entry.size)}</td>
            <td class="meta">{new Date(entry.modTime).toISOString().slice(0, 10)}</td>
            <td>
              <button class="sm" on:click={() => download(entry.name)} title="Download">↓</button>
              <button class="sm" on:click={() => { renameTarget=entry.name; renameVal=entry.name; renameModal=true; }} title="Rename">✎</button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>

<!-- Modals -->
{#if mkdirModal}
  <div class="modal-bg" on:click={() => mkdirModal=false} on:keypress={() => mkdirModal=false} role="button" tabindex="0">
    <div class="modal" on:click|stopPropagation on:keypress|stopPropagation role="dialog">
      <h3>New Folder</h3>
      <input bind:value={mkdirName} placeholder="folder name" on:keydown={e => e.key==='Enter' && doMkdir()} />
      <div class="modal-btns">
        <button on:click={doMkdir}>Create</button>
        <button class="cancel" on:click={() => mkdirModal=false}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

{#if renameModal}
  <div class="modal-bg" on:click={() => renameModal=false} on:keypress={() => renameModal=false} role="button" tabindex="0">
    <div class="modal" on:click|stopPropagation on:keypress|stopPropagation role="dialog">
      <h3>Rename</h3>
      <input bind:value={renameVal} placeholder="new name" on:keydown={e => e.key==='Enter' && doRename()} />
      <div class="modal-btns">
        <button on:click={doRename}>Rename</button>
        <button class="cancel" on:click={() => renameModal=false}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

{#if moveModal}
  <div class="modal-bg" on:click={() => moveModal=false} on:keypress={() => moveModal=false} role="button" tabindex="0">
    <div class="modal modal-wide" on:click|stopPropagation on:keypress|stopPropagation role="dialog">
      <h3>Move to</h3>
      <TreePicker bind:value={moveDst} />
      <div class="modal-btns">
        <button on:click={doMove}>Move</button>
        <button class="cancel" on:click={() => moveModal=false}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

{#if deleteModal}
  <div class="modal-bg" on:click={() => deleteModal=false} on:keypress={() => deleteModal=false} role="button" tabindex="0">
    <div class="modal" on:click|stopPropagation on:keypress|stopPropagation role="dialog">
      <h3>Delete {selected.size} item{selected.size !== 1 ? 's' : ''}?</h3>
      <p class="modal-warn">This action cannot be undone.</p>
      <div class="modal-btns">
        <button class="danger" on:click={doDeleteConfirmed}>Delete</button>
        <button class="cancel" on:click={() => deleteModal=false}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

{#if copyModal}
  <div class="modal-bg" on:click={() => copyModal=false} on:keypress={() => copyModal=false} role="button" tabindex="0">
    <div class="modal modal-wide" on:click|stopPropagation on:keypress|stopPropagation role="dialog">
      <h3>Copy to</h3>
      <TreePicker bind:value={copyDst} />
      <div class="modal-btns">
        <button on:click={doCopy}>Copy</button>
        <button class="cancel" on:click={() => copyModal=false}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .files-page { display:flex; flex-direction:column; gap:16px; }
  .toolbar { display:flex; flex-direction:column; gap:10px; }
  .breadcrumb { display:flex; align-items:center; gap:4px; flex-wrap:wrap; }
  .crumb { background:none; padding:2px 4px; color:#888888; font-size:0.9rem; transition:color 150ms ease; }
  .crumb:hover { color:#ffffff; background:none; }
  .crumb-current { color:#ffffff; font-weight:500; cursor:default; }
  .sep { color:#444444; font-size:0.85rem; user-select:none; }
  .actions { display:flex; gap:6px; flex-wrap:wrap; }
  .error { color:#f87171; font-size:0.85rem; }
  .danger { background:#7f1d1d; }
  .danger:hover { background:#991b1b; }

  .drop-zone { position:relative; border-radius:8px; border:2px dashed transparent; transition:border-color .2s; }
  .drop-zone.dragging { border-color:#555555; background:rgba(255,255,255,.04); }
  .drop-hint {
    position:absolute; inset:0; display:flex; align-items:center; justify-content:center;
    font-size:1.3rem; color:#888888; pointer-events:none; z-index:1;
  }

  .file-table { width:100%; border-collapse:collapse; }
  .file-table th { text-align:left; padding:8px 12px; color:#888888; font-size:0.8rem; border-bottom:1px solid #2a2a2a; }
  .file-table td { padding:8px 12px; border-bottom:1px solid #222222; }
  .file-table tr:hover { background:#222222; }
  .file-table tr.sel { background:#2a2a2a; }
  .name-btn { background:none; color:#ffffff; padding:0; text-align:left; font-size:0.82rem; }
  .name-btn:hover { color:#cccccc; background:none; }
  .meta { color:#888888; font-size:0.82rem; }
  .sm { padding:3px 8px; font-size:0.8rem; background:#222222; margin:0 2px; }

  .modal-bg {
    position:fixed; inset:0; background:rgba(0,0,0,.7);
    display:flex; align-items:center; justify-content:center; z-index:100;
  }
  .modal {
    background:#1a1a1a; border:1px solid #2a2a2a; border-radius:12px;
    padding:28px; min-width:300px; display:flex; flex-direction:column; gap:14px;
  }
  .modal-wide { min-width: 400px; width: 440px; }
  .modal h3 { font-size:1.1rem; color:#ffffff; }
  .modal-warn { font-size:0.85rem; color:#888888; }
  .modal input { width:100%; }
  .modal-btns { display:flex; gap:8px; justify-content:flex-end; }
  .cancel { background:#222222; }
</style>
