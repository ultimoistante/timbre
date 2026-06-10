<script>
  import { api } from '$lib/api/client.js';

  export let node;      // { name, path, children: null|[], loading, open }
  export let selected;  // currently selected path string
  export let onSelect;  // callback(path)
  export let depth = 0;

  async function toggle(e) {
    e.stopPropagation();
    node.open = !node.open;
    if (node.open && node.children === null) {
      node.loading = true;
      node = node;
      const data = await api.get('/fs/list?path=' + encodeURIComponent(node.path)).catch(() => ({ entries: [] }));
      node.children = (data.entries || [])
        .filter(e => e.isDir)
        .map(e => ({
          name: e.name,
          path: node.path ? node.path + '/' + e.name : e.name,
          children: null,
          loading: false,
          open: false,
        }));
      node.loading = false;
    }
    node = node;
  }
</script>

<div class="node">
  <div
    class="row"
    class:sel={selected === node.path}
    style="padding-left: {depth * 16 + 8}px"
    on:click={() => onSelect(node.path)}
    on:keypress={() => onSelect(node.path)}
    role="button"
    tabindex="0"
  >
    <button class="arrow" on:click={toggle} tabindex="-1">
      {#if node.loading}
        <span class="spin">⟳</span>
      {:else if node.open}
        <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"/></svg>
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>
      {/if}
    </button>

    <svg class="folder-icon" xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
    </svg>

    <span class="label">{node.name || 'Home'}</span>
  </div>

  {#if node.open && node.children}
    {#each node.children as child (child.path)}
      <svelte:self node={child} {selected} {onSelect} depth={depth + 1} />
    {/each}
    {#if node.children.length === 0}
      <div class="empty-dir" style="padding-left: {(depth + 1) * 16 + 8}px">No subfolders</div>
    {/if}
  {/if}
</div>

<style>
  .node { display: flex; flex-direction: column; }

  .row {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 32px;
    border-radius: 4px;
    cursor: pointer;
    color: #cccccc;
    user-select: none;
    transition: background 100ms ease;
  }
  .row:hover { background: #2a2a2a; }
  .row.sel { background: #1e3a2f; color: #1db954; }
  .row.sel .folder-icon { stroke: #1db954; }

  .arrow {
    width: 20px;
    height: 20px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: none;
    padding: 0;
    color: #555555;
    flex-shrink: 0;
    border-radius: 3px;
  }
  .arrow:hover { background: #333333; color: #aaaaaa; }

  .folder-icon { flex-shrink: 0; color: #888888; }

  .label { font-size: 0.875rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  .spin { font-size: 0.75rem; color: #888888; }

  .empty-dir { font-size: 0.78rem; color: #555555; padding-top: 4px; padding-bottom: 4px; }
</style>
