import axios from 'axios'
import { showToast } from '../toast'

const api = axios.create({ baseURL: '/api' })

// Auth credentials are kept as a base64-encoded "admin:password" token in
// localStorage. We use localStorage (not sessionStorage) so reopening the tab
// or restoring after a reboot doesn't force a re-login — single-user app on
// a personal device, the convenience is worth the marginal extra exposure.
// Logout explicitly clears the value.
const TOKEN_KEY = 'auth_token'

const restoredToken = localStorage.getItem(TOKEN_KEY)
if (restoredToken) {
  api.defaults.headers.common['Authorization'] = 'Basic ' + restoredToken
}

function setToken(password) {
  const token = btoa('admin:' + password)
  localStorage.setItem(TOKEN_KEY, token)
  api.defaults.headers.common['Authorization'] = 'Basic ' + token
}

function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
  delete api.defaults.headers.common['Authorization']
}

export const isLoggedIn = () => !!localStorage.getItem(TOKEN_KEY)

let lastAuthToastAt = 0

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      // Stored token is no longer valid — drop it and bounce to login.
      // Avoid loops: only redirect if we're not already on the login screen.
      const url = err.config?.url || ''
      const isAuthCall = url.includes('/auth/')
      if (!isAuthCall) {
        clearToken()
        const now = Date.now()
        if (now - lastAuthToastAt > 3000) {
          lastAuthToastAt = now
          showToast('Sitzung abgelaufen — bitte erneut anmelden.', 'error')
        }
        // Defer a tick so any in-flight requests can settle.
        setTimeout(() => window.location.replace('/'), 250)
      }
      return Promise.reject(err)
    }
    const msg = err.response?.data?.error || err.message || 'Unbekannter Fehler'
    const method = err.config?.method?.toUpperCase() || ''
    const url = err.config?.url || ''
    showToast(`${method} ${url}: ${msg}`, 'error')
    return Promise.reject(err)
  }
)

export const providersApi = {
  list: () => api.get('/providers').then(r => r.data),
  create: (data) => api.post('/providers', data).then(r => r.data),
  update: (id, data) => api.put(`/providers/${id}`, data).then(r => r.data),
  delete: (id) => api.delete(`/providers/${id}`),
  seedKitas: (id, seed) => api.post(`/providers/${id}/seed-kitas?seed=${seed}`).then(r => r.data),
  importExcel: (id, file, { year, month, kitaId } = {}) => {
    const form = new FormData()
    form.append('file', file)
    if (year)   form.append('year', year)
    if (month)  form.append('month', month)
    if (kitaId) form.append('kita_id', kitaId)
    return api.post(`/providers/${id}/import-excel`, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }).then(r => r.data)
  },
}

export const kitasApi = {
  list: () => api.get('/kitas').then(r => r.data),
  get: (id) => api.get(`/kitas/${id}`).then(r => r.data),
  create: (data) => api.post('/kitas', data).then(r => r.data),
  update: (id, data) => api.put(`/kitas/${id}`, data).then(r => r.data),
  delete: (id) => api.delete(`/kitas/${id}`),
  lookupStops: (id) => api.post(`/kitas/${id}/lookup-stops`).then(r => r.data),
}

export const assignmentsApi = {
  list: (from, to) => api.get('/assignments', { params: { from, to } }).then(r => r.data),
  get: (id) => api.get(`/assignments/${id}`).then(r => r.data),
  create: (data) => api.post('/assignments', data).then(r => r.data),
  update: (id, data) => api.put(`/assignments/${id}`, data).then(r => r.data),
  delete: (id) => api.delete(`/assignments/${id}`),
  bulkDelete: (ids) => api.post('/assignments/bulk-delete', { ids }).then(r => r.data),
}

export const recurringApi = {
  list: () => api.get('/recurring').then(r => r.data),
  create: (data) => api.post('/recurring', data).then(r => r.data),
  delete: (id) => api.delete(`/recurring/${id}`),
}

export const closuresApi = {
  list: (params) => api.get('/closures', { params }).then(r => r.data),
  create: (data) => api.post('/closures', data).then(r => r.data),
  delete: (id) => api.delete(`/closures/${id}`),
}

export const settingsApi = {
  get: () => api.get('/settings').then(r => r.data),
  update: (data) => api.put('/settings', data).then(r => r.data),
}

export const transitApi = {
  connections: (params) => api.get('/transit/connections', { params }).then(r => r.data),
  stops: (q) => api.get('/transit/stops', { params: { q } }).then(r => r.data),
}

export const authApi = {
  status: () => api.get('/auth/status').then(r => r.data),
  setup: async (password) => {
    const r = await api.post('/auth/setup', { password })
    // We just defined the password — cache it so the user lands directly in
    // the app after the post-setup reload.
    setToken(password)
    return r.data
  },
  // Validates the password by issuing one request with the candidate token,
  // bypassing the stored token. On success persist; on 401 throw a friendly
  // error without falling into the global 401 handler.
  login: async (password) => {
    const token = btoa('admin:' + password)
    try {
      await api.get('/settings', {
        headers: { Authorization: 'Basic ' + token },
      })
    } catch (e) {
      if (e.response?.status === 401) {
        throw new Error('Falsches Passwort')
      }
      throw e
    }
    setToken(password)
  },
  changePassword: async (oldPassword, newPassword) => {
    await api.put('/auth/password', {
      old_password: oldPassword,
      new_password: newPassword,
    })
    // Server now expects the new password — refresh our cached token so the
    // next request doesn't 401.
    setToken(newPassword)
  },
  logout: async () => {
    try { await api.post('/auth/logout') } catch { /* ignore */ }
    clearToken()
    window.location.replace('/')
  },
  getDownloadToken: () => api.get('/auth/download-token').then(r => r.data.token),
  regenerateDownloadToken: () => api.post('/auth/download-token').then(r => r.data.token),
}

// Blob helpers — needed because <a href> downloads can't carry our
// JS-managed Authorization header. Each helper fetches the file with auth,
// then triggers a client-side download.
async function blobDownload(url, filename, params) {
  const r = await api.get(url, { params, responseType: 'blob' })
  const objUrl = URL.createObjectURL(r.data)
  const a = document.createElement('a')
  a.href = objUrl
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(objUrl)
}

export const downloadsApi = {
  worktimePDF: (month, providerId) =>
    blobDownload('/worktime/export', `Arbeitszeiten_${month}.pdf`,
      providerId ? { month, provider_id: providerId } : { month }),
  calendarICS: () => blobDownload('/calendar.ics', 'kita-einsaetze.ics'),
}

// Full DB backup/restore. Restore wipes the password on the server side;
// caller MUST clear the local token and reload right after.
export const backupApi = {
  download: () => {
    const today = new Date().toISOString().slice(0, 10)
    return blobDownload('/backup', `kita-springer-${today}.db`)
  },
  restore: async (file) => {
    const fd = new FormData()
    fd.append('file', file)
    await api.post('/restore', fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
}
