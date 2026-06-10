<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client.js';
  import TreeNode from './TreeNode.svelte';

  export let value = ''; // selected path (bind:value)

  let root = { name: 'Home', path: '', children: null, loading: false, open: true };

  onMount(async () => {
    root.loading = true;
    root = root;
    const data = await api.get('/fs/list?path=').catch(() => ({ entries: [] }));
    root.children = (data.entries || [])
      .filter(e => e.isDir)
      .map(e => ({
        name: e.name,
        path: e.name,
        children: null,
        loading: false,
        open: false,
      }));
    root.loading = false;
    root = root;
  });

  function select(path) {
    value = path;
  }

  $: displayPath = value === '' ? 'Home' : 'Home / ' + value.split('/').join(' / ');
</script>

<div class="picker">
  <div class="tree">
    <TreeNode node={root} selected={value} onSelect={select} depth={0} />
  </div>
  <div class="selected-bar">
    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
    <span class="path-text">{displayPath}</span>
  </div>
</div>

<style>
  .picker {
    display: flex;
    flex-direction: column;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
    overflow: hidden;
  }

  .tree {
    overflow-y: auto;
    max-height: 260px;
    padding: 6px 4px;
    background: #161616;
  }

  .selected-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    background: #1a1a1a;
    border-top: 1px solid #2a2a2a;
    color: #888888;
    font-size: 0.8rem;
  }

  .path-text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: #cccccc;
  }
</style>
