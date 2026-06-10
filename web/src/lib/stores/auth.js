import { writable } from 'svelte/store';

const STORAGE_KEY = 'ms_auth';

function loadStored() {
  try {
    return JSON.parse(localStorage.getItem(STORAGE_KEY) || 'null') || { accessToken: null, user: null };
  } catch { return { accessToken: null, user: null }; }
}

function createAuthStore() {
  const { subscribe, set, update } = writable(loadStored());

  return {
    subscribe,
    setTokens(accessToken, user) {
      const val = { accessToken, user };
      localStorage.setItem(STORAGE_KEY, JSON.stringify(val));
      set(val);
    },
    logout() {
      localStorage.removeItem(STORAGE_KEY);
      set({ accessToken: null, user: null });
    }
  };
}

export const auth = createAuthStore();
