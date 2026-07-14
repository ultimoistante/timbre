<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client.js';
  import { auth } from '$lib/stores/auth.js';
  import { goto } from '$app/navigation';

  let users = [];
  let error = '';
  let createError = '';
  let newUser = { username: '', password: '', confirmPassword: '', role: 'user', quotaMb: 0 };
  let creating = false;
  let deleteTarget = null;
  let showCreateModal = false;
  let showPassword = false;
  let showConfirmPassword = false;

  onMount(() => {
    if ($auth.user?.role !== 'admin') { goto('/library'); return; }
    loadUsers();
  });

  async function loadUsers() {
    users = await api.get('/admin/users').catch(e => { error = e.message; return []; });
  }

  async function createUser() {
    if (newUser.password !== newUser.confirmPassword) {
      createError = 'Passwords do not match';
      return;
    }
    creating = true; createError = '';
    await api.post('/admin/users', {
      username: newUser.username,
      password: newUser.password,
      role: newUser.role,
      quotaBytes: Math.round(newUser.quotaMb * 1024 * 1024)
    }).catch(e => createError = e.message);
    creating = false;
    if (!createError) {
      newUser = { username: '', password: '', confirmPassword: '', role: 'user', quotaMb: 0 };
      showPassword = false;
      showConfirmPassword = false;
      showCreateModal = false;
    }
    loadUsers();
  }

  async function doDeleteUser() {
    await api.delete(`/admin/users/${deleteTarget.id}`).catch(e => error = e.message);
    deleteTarget = null;
    loadUsers();
  }

  function fmtQuota(bytes) {
    if (!bytes) return 'Unlimited';
    return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`;
  }
</script>

<div class="admin">
  <h1>Admin — Users</h1>
  {#if error}<p class="error">{error}</p>{/if}

  <section>
    <div class="section-header">
      <h2>All Users</h2>
      <button class="add-btn" on:click={() => { createError = ''; showCreateModal = true; }}>Add user</button>
    </div>
    <table class="user-table">
      <thead>
        <tr>
          <th>ID</th>
          <th>Username</th>
          <th>Role</th>
          <th>Quota</th>
          <th>Created</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {#each users as u}
          <tr>
            <td class="meta">{u.id}</td>
            <td class="uname">{u.username}</td>
            <td><span class="badge" class:is-admin={u.role === 'admin'}>{u.role}</span></td>
            <td class="meta">{fmtQuota(u.quotaBytes)}</td>
            <td class="meta">{new Date(u.createdAt).toISOString().slice(0, 10)}</td>
            <td>
              {#if u.id !== $auth.user?.id}
                <button class="del-btn" on:click={() => deleteTarget = u}>Delete</button>
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </section>
</div>

{#if showCreateModal}
  <div class="modal-bg">
    <div class="modal" role="dialog" tabindex="-1">
      <h3>Create user</h3>
      {#if createError}<p class="error">{createError}</p>{/if}
      <form class="create-form" on:submit|preventDefault={createUser}>
        <label class="field">
          Username
          <input bind:value={newUser.username} placeholder="Username" autocomplete="off" required />
        </label>
        <label class="field">
          Password
          <div class="password-input">
            <input type={showPassword ? 'text' : 'password'} bind:value={newUser.password} placeholder="Password" autocomplete="new-password" required />
            <button type="button" class="reveal-btn" on:click={() => showPassword = !showPassword} aria-label={showPassword ? 'Hide password' : 'Show password'}>
              {#if showPassword}
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.94 17.94A10.94 10.94 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
              {:else}
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
              {/if}
            </button>
          </div>
        </label>
        <label class="field">
          Confirm password
          <div class="password-input">
            <input type={showConfirmPassword ? 'text' : 'password'} bind:value={newUser.confirmPassword} placeholder="Confirm password" autocomplete="new-password" required />
            <button type="button" class="reveal-btn" on:click={() => showConfirmPassword = !showConfirmPassword} aria-label={showConfirmPassword ? 'Hide password' : 'Show password'}>
              {#if showConfirmPassword}
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.94 17.94A10.94 10.94 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
              {:else}
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
              {/if}
            </button>
          </div>
        </label>
        <label class="field">
          Role
          <select bind:value={newUser.role}>
            <option value="user">User</option>
            <option value="admin">Admin</option>
          </select>
        </label>
        <label class="field">
          Disk quota (MB, 0 = unlimited)
          <input type="number" bind:value={newUser.quotaMb} placeholder="0" min="0" />
        </label>
        <div class="modal-btns">
          <button type="submit" disabled={creating}>{creating ? 'Creating…' : 'Create user'}</button>
          <button type="button" class="cancel" on:click={() => { showCreateModal = false; createError = ''; }}>Cancel</button>
        </div>
      </form>
    </div>
  </div>
{/if}

{#if deleteTarget}
  <div class="modal-bg" on:click={() => deleteTarget = null} on:keypress={() => deleteTarget = null} role="button" tabindex="0">
    <div class="modal" on:click|stopPropagation on:keypress|stopPropagation role="dialog" tabindex="-1">
      <h3>Delete "{deleteTarget.username}"?</h3>
      <p class="modal-warn">This will permanently delete the user and all their files.</p>
      <div class="modal-btns">
        <button class="danger" on:click={doDeleteUser}>Delete</button>
        <button class="cancel" on:click={() => deleteTarget = null}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .admin { display: flex; flex-direction: column; gap: 28px; }

  h1 { font-size: 1.4rem; font-weight: 700; color: #ffffff; margin: 0; }
  h2 { font-size: 1rem; font-weight: 600; color: #ffffff; margin: 0 0 14px 0; }

  .error { color: #f87171; font-size: 0.85rem; }

  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 14px;
  }
  .section-header h2 { margin: 0; }

  .add-btn { background: #2a2a2a; }
  .add-btn:hover { background: #333333; }

  .create-form {
    display: flex;
    flex-direction: column;
    gap: 10px;
    min-width: 280px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 0.8rem;
    color: #888888;
  }

  .password-input {
    position: relative;
    display: flex;
  }
  .password-input input { width: 100%; padding-right: 36px; }

  .create-form input:-webkit-autofill,
  .create-form input:-webkit-autofill:hover,
  .create-form input:-webkit-autofill:focus {
    -webkit-text-fill-color: #ffffff;
    -webkit-box-shadow: 0 0 0 1000px #222222 inset;
    box-shadow: 0 0 0 1000px #222222 inset;
    transition: background-color 9999s ease-in-out 0s;
  }

  .reveal-btn {
    position: absolute;
    top: 50%;
    right: 4px;
    transform: translateY(-50%);
    background: none;
    padding: 4px;
    color: #888888;
  }
  .reveal-btn:hover { background: none; color: #ffffff; }

  .user-table { width: 100%; border-collapse: collapse; }
  .user-table th {
    text-align: left;
    padding: 8px 12px;
    color: #888888;
    font-size: 0.8rem;
    font-weight: 500;
    border-bottom: 1px solid #2a2a2a;
  }
  .user-table td { padding: 10px 12px; border-bottom: 1px solid #222222; }
  .user-table tr:hover td { background: #1e1e1e; }

  .uname { font-weight: 600; }
  .meta { color: #888888; font-size: 0.82rem; }

  .badge {
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 0.78rem;
    background: #2a2a2a;
    color: #888888;
  }
  .badge.is-admin { background: #2d1f4e; color: #c4b5fd; }

  .del-btn {
    padding: 4px 10px;
    font-size: 0.8rem;
    background: #2a2a2a;
    color: #f87171;
    border-radius: 4px;
    transition: background 150ms ease;
  }
  .del-btn:hover { background: #3a2020; }

  .modal-bg {
    position: fixed; inset: 0; background: rgba(0,0,0,.7);
    display: flex; align-items: center; justify-content: center; z-index: 100;
  }
  .modal {
    background: #1a1a1a; border: 1px solid #2a2a2a; border-radius: 12px;
    padding: 28px; min-width: 320px; display: flex; flex-direction: column; gap: 14px;
  }
  .modal h3 { font-size: 1.1rem; color: #ffffff; margin: 0; }
  .modal-warn { font-size: 0.85rem; color: #888888; margin: 0; }
  .modal-btns { display: flex; gap: 8px; justify-content: flex-end; }
  .danger { background: #7f1d1d; }
  .danger:hover { background: #991b1b; }
  .cancel { background: #222222; }
</style>
