<script>
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { auth } from '$lib/stores/auth.js';
  import { api } from '$lib/api/client.js';
  import Player from '$lib/components/Player.svelte';
  import { currentTrack } from '$lib/stores/player.js';

  const PUBLIC = ['/auth/login', '/auth/onboarding'];

  onMount(async () => {
    // Check if onboarding needed.
    const status = await api.get('/onboarding').catch(() => null);
    if (status?.needsOnboarding && !$page.url.pathname.startsWith('/auth/onboarding')) {
      goto('/auth/onboarding');
      return;
    }
    if (!$auth.accessToken && !PUBLIC.some(p => $page.url.pathname.startsWith(p))) {
      goto('/auth/login');
    }
  });

  function isActive(href) {
    if (href === '/') return $page.url.pathname === '/';
    return $page.url.pathname.startsWith(href);
  }

  $: userInitial = $auth.user?.username?.[0]?.toUpperCase() ?? 'U';

  let artError = false;
  $: if ($currentTrack) artError = false;

  // Streams have no album hash; use their station favicon (may be empty).
  $: artSrc = !$currentTrack
    ? ''
    : $currentTrack.isStream
      ? ($currentTrack.favicon || '')
      : ($currentTrack.albumHash ? `/api/albums/${$currentTrack.albumHash}/art` : '');
</script>

<div class="app">
  {#if $auth.accessToken}
    <nav class="sidebar">
      <div class="brand-row">
        <span class="brand">Timbre</span>
        <div class="avatar">{userInitial}</div>
      </div>

      <a href="/" class="nav-link" class:active={isActive('/')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>
        Home
      </a>

      <a href="/library" class="nav-link" class:active={isActive('/library')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>
        Library
      </a>

      <a href="/playlists" class="nav-link" class:active={isActive('/playlists')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>
        Playlists
      </a>

      <a href="/streams" class="nav-link" class:active={isActive('/streams')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="2"/><path d="M4.93 19.07a10 10 0 0 1 0-14.14M7.76 16.24a6 6 0 0 1 0-8.49M16.24 7.76a6 6 0 0 1 0 8.49M19.07 4.93a10 10 0 0 1 0 14.14"/></svg>
        Streams
      </a>

      <a href="/files" class="nav-link" class:active={isActive('/files')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
        Files
      </a>

      <a href="/search" class="nav-link" class:active={isActive('/search')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
        Search
      </a>

      {#if $auth.user?.role === 'admin'}
        <a href="/admin" class="nav-link" class:active={isActive('/admin')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.07 4.93a10 10 0 0 1 0 14.14M4.93 4.93a10 10 0 0 0 0 14.14"/></svg>
          Admin
        </a>
      {/if}

      <hr class="nav-sep" />

      <button class="logout-btn" on:click={() => { api.post('/auth/logout'); auth.logout(); goto('/auth/login'); }}>
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
        Logout
      </button>

      <div class="sidebar-bottom">
        <div class="now-playing-art">
          {#if $currentTrack && artSrc && !artError}
            <img
              src={artSrc}
              alt="Album art"
              on:error={() => { artError = true; }}
            />
          {:else}
            <div class="art-placeholder">
              <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>
            </div>
          {/if}
        </div>

      </div>
    </nav>
  {/if}

  <main class="content">
    <slot />
  </main>

  {#if $auth.accessToken}
    <Player />
  {/if}
</div>

<style>
  :global(*, *::before, *::after) { box-sizing: border-box; margin: 0; padding: 0; }
  :global(body) {
    background: #111111;
    color: #ffffff;
    font-family: 'Poppins', system-ui, sans-serif;
    min-height: 100dvh;
  }
  :global(a) { color: #cccccc; text-decoration: none; }
  :global(a:hover) { color: #ffffff; }
  :global(button) {
    cursor: pointer;
    border: none;
    border-radius: 4px;
    padding: 6px 14px;
    background: #2a2a2a;
    color: #ffffff;
    font-family: inherit;
    font-size: inherit;
  }
  :global(button:hover) { background: #333333; }
  :global(input, select, textarea) {
    background: #222222;
    border: 1px solid #2a2a2a;
    border-radius: 4px;
    color: #ffffff;
    padding: 6px 10px;
    font-family: inherit;
    font-size: inherit;
  }

  .app {
    display: grid;
    grid-template-columns: 220px 1fr;
    grid-template-rows: 1fr auto;
    min-height: 100dvh;
  }

  .sidebar {
    grid-row: 1;
    grid-column: 1;
    background: #1a1a1a;
    padding: 16px 12px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    border-right: 1px solid #2a2a2a;
  }

  .brand-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
    padding: 0 4px;
  }

  .brand {
    font-size: 1.2rem;
    font-weight: 700;
    color: #ffffff;
    letter-spacing: 0.02em;
  }

  .avatar {
    width: 30px;
    height: 30px;
    border-radius: 50%;
    background: #333333;
    color: #ffffff;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.8rem;
    font-weight: 600;
    flex-shrink: 0;
  }

  .nav-link {
    display: flex;
    align-items: center;
    gap: 10px;
    background: none;
    color: #cccccc;
    padding: 8px 10px;
    border-radius: 6px;
    text-align: left;
    width: 100%;
    font-size: 0.9rem;
    transition: background 150ms ease, color 150ms ease;
  }
  .nav-link:hover { background: #222222; color: #ffffff; }
  .nav-link.active { background: #222222; color: #ffffff; }

  .nav-sep {
    border: none;
    border-top: 1px solid #2a2a2a;
    margin: 4px 0;
  }

  .sidebar-bottom {
    margin-top: auto;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .now-playing-art {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .now-playing-art img {
    width: 100%;
    aspect-ratio: 1/1;
    object-fit: cover;
    border-radius: 6px;
    display: block;
  }

  .art-placeholder {
    width: 100%;
    aspect-ratio: 1/1;
    background: #222222;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #888888;
  }

  .logout-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    background: none;
    color: #888888;
    padding: 8px 10px;
    border-radius: 6px;
    width: 100%;
    text-align: left;
    font-size: 0.9rem;
    transition: background 150ms ease, color 150ms ease;
  }
  .logout-btn:hover { background: #222222; color: #ffffff; }

  .content {
    grid-row: 1;
    grid-column: 2;
    padding: 24px;
    overflow-y: auto;
  }

  /* Full-width when no sidebar (auth pages) */
  .app:not(:has(nav)) .content {
    grid-column: 1 / -1;
  }
</style>
