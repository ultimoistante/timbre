<script>
  import { onMount } from 'svelte';
  import QRCode from 'qrcode';
  import { api } from '$lib/api/client.js';

  // token holds the active Subsonic credentials, or null when none is set.
  let token = null;       // { username, token, restUrl }
  let loading = true;
  let busy = false;
  let error = '';
  let revealed = false;
  let copied = '';        // which field was last copied
  let confirm = null;     // 'rotate' | 'revoke'
  let customInput = '';   // user-chosen custom password
  let showQr = false;
  let qrDataUrl = '';
  let appVersion = '';

  onMount(() => {
    loadToken();
    api.get('/version').then(v => appVersion = v.version).catch(() => {});
  });

  function resetQr() { showQr = false; qrDataUrl = ''; }

  async function loadToken() {
    loading = true; error = '';
    try {
      token = await api.get('/me/subsonic-token');
    } catch {
      // 404 = no token yet; any other error is non-fatal for this view.
      token = null;
    }
    loading = false;
  }

  async function generate() {
    busy = true; error = '';
    try {
      token = await api.post('/me/subsonic-token');
      revealed = true;
      resetQr();
    } catch (e) {
      error = e.message;
    }
    busy = false;
    confirm = null;
  }

  async function setCustom() {
    if (customInput.trim().length < 8) { error = 'Token must be at least 8 characters'; return; }
    busy = true; error = '';
    try {
      token = await api.put('/me/subsonic-token', { token: customInput.trim() });
      revealed = true;
      customInput = '';
      resetQr();
    } catch (e) {
      error = e.message;
    }
    busy = false;
  }

  async function toggleQr() {
    showQr = !showQr;
    if (showQr && token) {
      try {
        qrDataUrl = await QRCode.toDataURL(token.token, { margin: 2, width: 220 });
      } catch {
        error = 'QR generation failed';
        showQr = false;
      }
    }
  }

  async function revoke() {
    busy = true; error = '';
    try {
      await api.delete('/me/subsonic-token');
      token = null;
      revealed = false;
      resetQr();
    } catch (e) {
      error = e.message;
    }
    busy = false;
    confirm = null;
  }

  async function copy(field, value) {
    try {
      await navigator.clipboard.writeText(value);
      copied = field;
      setTimeout(() => { if (copied === field) copied = ''; }, 1500);
    } catch {
      error = 'Clipboard not available';
    }
  }

  $: masked = token ? '•'.repeat(Math.min(token.token.length, 40)) : '';
</script>

<div class="settings">
  <h1>Settings</h1>
  {#if error}<p class="error">{error}</p>{/if}

  <section class="card">
    <div class="card-head">
      <div>
        <h2>External player apps (OpenSubsonic)</h2>
        <p class="hint">
          Connect any open-standard music player (Symfonium, substreamer, Feishin,
          Amperfy, DSub…) to Timbre. Use your username and the token below as the
          password — most clients append <code>/rest</code> automatically.
        </p>
      </div>
    </div>

    {#if loading}
      <p class="muted">Loading…</p>
    {:else if !token}
      <div class="empty">
        <p class="muted">No API token yet. Generate one to enable external apps.</p>
        <button class="primary" on:click={generate} disabled={busy}>
          {busy ? 'Generating…' : 'Generate token'}
        </button>
      </div>
    {:else}
      <div class="fields">
        <div class="field">
          <span class="label">Server URL</span>
          <div class="value-row">
            <code class="value">{token.restUrl.replace(/\/rest$/, '')}</code>
            <button class="ghost" on:click={() => copy('url', token.restUrl.replace(/\/rest$/, ''))}>
              {copied === 'url' ? 'Copied' : 'Copy'}
            </button>
          </div>
        </div>

        <div class="field">
          <span class="label">Username</span>
          <div class="value-row">
            <code class="value">{token.username}</code>
            <button class="ghost" on:click={() => copy('user', token.username)}>
              {copied === 'user' ? 'Copied' : 'Copy'}
            </button>
          </div>
        </div>

        <div class="field">
          <span class="label">Token (use as password)</span>
          <div class="value-row">
            <code class="value mono">{revealed ? token.token : masked}</code>
            <button class="ghost" on:click={() => revealed = !revealed}>
              {revealed ? 'Hide' : 'Reveal'}
            </button>
            <button class="ghost" on:click={() => copy('token', token.token)}>
              {copied === 'token' ? 'Copied' : 'Copy'}
            </button>
            <button class="ghost" on:click={toggleQr}>
              {showQr ? 'Hide QR' : 'Show QR'}
            </button>
          </div>
        </div>

        {#if showQr && qrDataUrl}
          <div class="qr-box">
            <img src={qrDataUrl} alt="Token QR code" />
            <p class="muted qr-hint">
              Scan with your phone to copy the token, then paste it into the
              app's password field. (Encodes the token only — enter the server
              URL and username manually.)
            </p>
          </div>
        {/if}
      </div>

      <div class="actions">
        <button class="warn" on:click={() => confirm = 'rotate'} disabled={busy}>Regenerate</button>
        <button class="danger" on:click={() => confirm = 'revoke'} disabled={busy}>Revoke</button>
      </div>
    {/if}

    <div class="custom">
      <h3 class="custom-title">Set a custom password</h3>
      <p class="hint">
        Prefer something you can type on a phone? Replace the token with your own
        memorable password (min 8 characters). It works exactly like the token —
        username + this password in your player app.
      </p>
      <form class="custom-form" on:submit|preventDefault={setCustom}>
        <input
          type="text"
          bind:value={customInput}
          placeholder="e.g. my-music-passphrase"
          autocomplete="off"
          autocapitalize="off"
          spellcheck="false"
        />
        <button class="primary" type="submit" disabled={busy || customInput.trim().length < 8}>
          {busy ? 'Saving…' : 'Set password'}
        </button>
      </form>
    </div>
  </section>

  {#if appVersion}
    <p class="version">Timbre v{appVersion}</p>
  {/if}
</div>

{#if confirm}
  <div class="modal-bg" on:click={() => confirm = null} on:keypress={() => confirm = null} role="button" tabindex="0">
    <div class="modal" on:click|stopPropagation on:keypress|stopPropagation role="dialog" tabindex="-1">
      {#if confirm === 'rotate'}
        <h3>Regenerate token?</h3>
        <p class="modal-warn">The current token stops working immediately. Every connected app must be reconfigured with the new token.</p>
        <div class="modal-btns">
          <button class="warn" on:click={generate}>Regenerate</button>
          <button class="cancel" on:click={() => confirm = null}>Cancel</button>
        </div>
      {:else}
        <h3>Revoke token?</h3>
        <p class="modal-warn">This disables external app access until you generate a new token. Connected apps will stop working.</p>
        <div class="modal-btns">
          <button class="danger" on:click={revoke}>Revoke</button>
          <button class="cancel" on:click={() => confirm = null}>Cancel</button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .settings { display: flex; flex-direction: column; gap: 24px; max-width: 720px; }

  h1 { font-size: 1.4rem; font-weight: 700; color: #ffffff; margin: 0; }
  h2 { font-size: 1rem; font-weight: 600; color: #ffffff; margin: 0 0 6px 0; }

  .error { color: #f87171; font-size: 0.85rem; }
  .muted { color: #888888; font-size: 0.88rem; }
  .version { color: #666666; font-size: 0.8rem; margin: 0; text-align: center; }

  .card {
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 12px;
    padding: 22px;
    display: flex;
    flex-direction: column;
    gap: 18px;
  }
  .card-head { display: flex; justify-content: space-between; align-items: flex-start; }
  .hint { color: #888888; font-size: 0.85rem; line-height: 1.5; margin: 0; }
  .hint code { background: #222222; padding: 1px 5px; border-radius: 4px; color: #cccccc; font-size: 0.82rem; }

  .empty { display: flex; flex-direction: column; gap: 12px; align-items: flex-start; }

  .fields { display: flex; flex-direction: column; gap: 14px; }
  .field { display: flex; flex-direction: column; gap: 6px; }
  .label { font-size: 0.78rem; color: #888888; }
  .value-row { display: flex; align-items: center; gap: 8px; }
  .value {
    flex: 1;
    background: #222222;
    border: 1px solid #2a2a2a;
    border-radius: 6px;
    padding: 8px 10px;
    color: #ffffff;
    font-size: 0.85rem;
    overflow-x: auto;
    white-space: nowrap;
  }
  .value.mono { font-family: ui-monospace, monospace; letter-spacing: 0.02em; }

  .actions { display: flex; gap: 10px; }

  .qr-box {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    padding: 12px;
    background: #222222;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
  }
  .qr-box img {
    width: 200px;
    height: 200px;
    border-radius: 6px;
    background: #ffffff;
    padding: 6px;
  }
  .qr-hint { text-align: center; max-width: 320px; line-height: 1.45; }

  .custom {
    border-top: 1px solid #2a2a2a;
    padding-top: 18px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .custom-title { font-size: 0.92rem; font-weight: 600; color: #ffffff; margin: 0; }
  .custom-form { display: flex; gap: 10px; align-items: center; }
  .custom-form input { flex: 1; }

  .primary { background: #2d4e2d; color: #bbf7d0; }
  .primary:hover { background: #356035; }
  .ghost { background: #222222; color: #cccccc; padding: 8px 12px; font-size: 0.82rem; }
  .ghost:hover { background: #2a2a2a; color: #ffffff; }
  .warn { background: #4e3d1f; color: #fde68a; }
  .warn:hover { background: #604c26; }
  .danger { background: #7f1d1d; color: #fecaca; }
  .danger:hover { background: #991b1b; }

  .modal-bg {
    position: fixed; inset: 0; background: rgba(0,0,0,.7);
    display: flex; align-items: center; justify-content: center; z-index: 100;
  }
  .modal {
    background: #1a1a1a; border: 1px solid #2a2a2a; border-radius: 12px;
    padding: 28px; min-width: 340px; max-width: 440px; display: flex; flex-direction: column; gap: 14px;
  }
  .modal h3 { font-size: 1.1rem; color: #ffffff; margin: 0; }
  .modal-warn { font-size: 0.85rem; color: #888888; margin: 0; line-height: 1.5; }
  .modal-btns { display: flex; gap: 8px; justify-content: flex-end; }
  .cancel { background: #222222; }
</style>
