import axios from 'axios'
import { showToast } from '../toast'

const api = axios.create({ baseURL: '/api' })

api.interceptors.response.use(
  (res) => res,
  (err) => {
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
