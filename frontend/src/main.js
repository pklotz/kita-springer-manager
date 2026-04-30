import { createApp, h } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import App from './App.vue'
import SetupView from './views/SetupView.vue'
import { authApi } from './api'
import './style.css'

// Bumped whenever a backend or SW change makes old caches incompatible.
// On a version mismatch we unregister all service workers and clear all
// caches once, then reload — guarantees the user never sits on a stale
// pre-auth shell.
const APP_VERSION = '5'

async function purgeStaleServiceWorker() {
  if (localStorage.getItem('app_version') === APP_VERSION) return false
  if (!('serviceWorker' in navigator)) {
    localStorage.setItem('app_version', APP_VERSION)
    return false
  }
  try {
    const regs = await navigator.serviceWorker.getRegistrations()
    await Promise.all(regs.map((r) => r.unregister()))
    if (window.caches) {
      const keys = await caches.keys()
      await Promise.all(keys.map((k) => caches.delete(k)))
    }
  } catch {
    // best-effort
  }
  localStorage.setItem('app_version', APP_VERSION)
  // If we actually unregistered something, force a reload so the next page
  // navigation hits the network and gets the new shell.
  return true
}

async function bootstrap() {
  if (await purgeStaleServiceWorker()) {
    window.location.reload()
    return
  }

  let configured = true
  try {
    const status = await authApi.status()
    configured = !!status.configured
  } catch {
    // If /api/auth/status itself fails (network etc.) fall through to the
    // normal app — better to show stale UI than a broken setup screen.
  }

  if (!configured) {
    createApp({ render: () => h(SetupView) }).mount('#app')
    return
  }
  // App.vue handles the logged-in / logged-out switch reactively, so a 401
  // from the server can swap to LoginView without a page reload.
  createApp(App).use(createPinia()).use(router).mount('#app')
}

bootstrap()
