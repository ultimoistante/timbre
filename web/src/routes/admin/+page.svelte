<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client.js';
  import { auth } from '$lib/stores/auth.js';
  import { goto } from '$app/navigation';

  let users = [];
  let error = '';
  let newUser = { username: '', password: '', role: 'user', quotaBytes: 0 };
  let creating = false;
  let deleteTarget = null;

  onMount(() => {
    if ($auth.user?.role !== 'admin') { goto('/library'); return; }
    loadUsers();
  });

  async function loadUsers() {
    users = await api.get('/admin/users').catch(e => { error = e.message; return []; });
  }

  async function createUser() {
    creating = true; error = '';
    await api.post('/admin/users', {
      username: newUser.username,
      password: newUser.password,
      role: newUser.role,
      quotaBytes: newUser.quotaBytes
    }).catch(e => error = e.message);
    creating = false;
    newUser = { username: '', password: '', role: 'user', quotaBytes: 0 };
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

  <section class="create-section">
    <h2>Create User</h2>
    <form on:submit|preventDefault={createUser}>
      <input bind:value={newUser.username} placeholder="Username" required />
      <input type="password" bind:value={newUser.password} placeholder="Password" required />
      <select bind:value={newUser.role}>
        <option value="user">User</option>
        <option value="admin">Admin</option>
      </select>
      <input type="number" bind:value={newUser.quotaBytes} placeholder="Quota bytes (0 = unlimited)" min="0" />
      <button type="submit" disabled={creating}>{creating ? 'Creating…' : 'Create user'}</button>
    </form>
  </section>

  <section>
    <h2>All Users</h2>
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
            <td class="meta">{new Date(u.createdAt).toLocaleDateString()}</td>
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

{#if deleteTarget}
  <div class="modal-bg" on:click={() => deleteTarget = null} on:keypress={() => deleteTarget = null} role="button" tabindex="0">
    <div class="modal" on:click|stopPropagation on:keypress|stopPropagation role="dialog">
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

  .create-section form {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
    align-items: center;
  }
  .create-section input,
  .create-section select { flex: 1; min-width: 140px; }

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
