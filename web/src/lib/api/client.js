/**
 * Typed API client. Reads the access token from the auth store and attaches it
 * as a Bearer header. On 401, attempts a token refresh once then retries.
 */

import { auth } from '$lib/stores/auth.js';
import { get } from 'svelte/store';

const BASE = '/api';

let refreshing = null;

async function refreshTokens() {
  if (refreshing) return refreshing;
  refreshing = fetch(`${BASE}/auth/refresh`, {
    method: 'POST',
    credentials: 'include'
  }).then(async r => {
    if (!r.ok) throw new Error('refresh failed');
    const data = await r.json();
    auth.setTokens(data.accessToken, data.user);
    return data.accessToken;
  }).finally(() => { refreshing = null; });
  return refreshing;
}

async function request(method, path, body, opts = {}) {
  const token = get(auth).accessToken;
  const headers = { 'Content-Type': 'application/json', ...(opts.headers || {}) };
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const init = { method, credentials: 'include', headers };
  if (body != null && !(body instanceof FormData)) {
    init.body = JSON.stringify(body);
    if (body instanceof FormData) delete headers['Content-Type'];
  } else if (body instanceof FormData) {
    delete headers['Content-Type'];
    init.body = body;
  }

  let res = await fetch(BASE + path, init);

  if (res.status === 401 && !opts._retry) {
    try {
      const newToken = await refreshTokens();
      headers['Authorization'] = `Bearer ${newToken}`;
      res = await fetch(BASE + path, { ...init, headers });
    } catch {
      auth.logout();
      throw new Error('Session expired');
    }
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }));
    throw new Error(err.message || res.statusText);
  }

  const ct = res.headers.get('Content-Type') || '';
  if (ct.includes('application/json')) return res.json();
  return res;
}

// fetch() has no upload-progress event, so uploads use XMLHttpRequest instead.
function uploadWithProgress(path, formData, onProgress) {
  const send = (token, retried) => new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open('POST', BASE + path);
    xhr.withCredentials = true;
    if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`);
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded / e.total);
    };
    xhr.onload = async () => {
      if (xhr.status === 401 && !retried) {
        try {
          const newToken = await refreshTokens();
          resolve(await send(newToken, true));
        } catch {
          auth.logout();
          reject(new Error('Session expired'));
        }
        return;
      }
      if (xhr.status >= 200 && xhr.status < 300) {
        const ct = xhr.getResponseHeader('Content-Type') || '';
        if (ct.includes('application/json')) {
          try { resolve(JSON.parse(xhr.responseText)); } catch { resolve(null); }
        } else {
          resolve(xhr.response);
        }
      } else {
        let message = xhr.statusText;
        try { message = JSON.parse(xhr.responseText).message || message; } catch { /* not JSON */ }
        reject(new Error(message));
      }
    };
    xhr.onerror = () => reject(new Error('Network error'));
    xhr.send(formData);
  });
  return send(get(auth).accessToken, false);
}

export const api = {
  get:    (path, opts)       => request('GET',    path, null, opts),
  post:   (path, body, opts) => request('POST',   path, body, opts),
  patch:  (path, body, opts) => request('PATCH',  path, body, opts),
  delete: (path, opts)       => request('DELETE', path, null, opts),
  put:    (path, body, opts) => request('PUT',    path, body, opts),

  /**
   * Upload files to destPath. Returns saved filenames.
   * onProgress, if given, is called with a 0..1 fraction as the upload streams.
   */
  upload(destPath, files, onProgress) {
    const fd = new FormData();
    for (const f of files) fd.append('file', f);
    if (!onProgress) return request('POST', `/upload?path=${encodeURIComponent(destPath)}`, fd);
    return uploadWithProgress(`/upload?path=${encodeURIComponent(destPath)}`, fd, onProgress);
  },

  /**
   * Stream URL for a track. The <audio> element sends cookies automatically
   * (same-origin), so auth is handled via the access_token cookie set at login.
   */
  streamUrl(id, quality = 'original', container = 'mp3') {
    if (quality === 'original') return `${BASE}/stream/${id}`;
    return `${BASE}/stream/${id}?quality=${quality}&container=${container}`;
  }
};
