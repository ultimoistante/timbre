<script>
  import { goto } from '$app/navigation';
  import { api } from '$lib/api/client.js';
  import { auth } from '$lib/stores/auth.js';

  let username = '', password = '', confirm = '', error = '', loading = false;

  async function submit() {
    if (password !== confirm) { error = 'Passwords do not match'; return; }
    error = ''; loading = true;
    try {
      const data = await api.post('/onboarding', { username, password });
      auth.setTokens(data.accessToken, data.user);
      goto('/library');
    } catch (e) { error = e.message; }
    finally { loading = false; }
  }
</script>

<div class="auth-page">
  <div class="card">
    <h1>Timbre</h1>
    <p class="sub">Create your admin account to get started.</p>
    <form on:submit|preventDefault={submit}>
      <label>Username <input bind:value={username} required /></label>
      <label>Password <input type="password" bind:value={password} required /></label>
      <label>Confirm  <input type="password" bind:value={confirm}  required /></label>
      {#if error}<p class="error">{error}</p>{/if}
      <button type="submit" disabled={loading}>{loading ? 'Creating…' : 'Create admin'}</button>
    </form>
  </div>
</div>

<style>
  .auth-page { display:flex; align-items:center; justify-content:center; min-height:100dvh; }
  .card { background:#1a1a1a; border:1px solid #2a2a2a; border-radius:12px; padding:40px; min-width:340px; }
  h1 { text-align:center; font-size:1.4rem; margin-bottom:6px; color:#ffffff; letter-spacing:0.04em; }
  .sub { text-align:center; color:#888888; font-size:0.9rem; margin-bottom:24px; }
  form { display:flex; flex-direction:column; gap:14px; }
  label { display:flex; flex-direction:column; gap:4px; font-size:0.9rem; color:#888888; }
  label input { margin-top:4px; width:100%; }
  button { margin-top:8px; padding:10px; font-size:1rem; background:#333333; }
  .error { color:#f87171; font-size:0.85rem; }
</style>
