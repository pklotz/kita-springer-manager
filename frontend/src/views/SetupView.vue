<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 px-4">
    <div class="w-full max-w-sm bg-white rounded-2xl shadow-md border border-gray-100 p-6">
      <h1 class="text-xl font-semibold text-gray-800 mb-1">Erste Einrichtung</h1>
      <p class="text-sm text-gray-500 mb-5">
        Lege ein Passwort fest. Der Benutzername ist <code class="px-1 bg-gray-100 rounded">admin</code>.
        Nach dem Speichern fragt der Browser einmal nach den Zugangsdaten.
      </p>

      <label class="block text-sm text-gray-600 mb-1">Passwort (mind. 8 Zeichen)</label>
      <input v-model="password" type="password" autocomplete="new-password"
        class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500" />

      <label class="block text-sm text-gray-600 mb-1">Passwort bestätigen</label>
      <input v-model="confirm" type="password" autocomplete="new-password"
        class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-4 focus:outline-none focus:ring-2 focus:ring-brand-500" />

      <p v-if="error" class="text-sm text-red-600 mb-3">{{ error }}</p>

      <button @click="submit" :disabled="busy"
        class="w-full py-2.5 bg-brand-500 text-white rounded-xl font-semibold hover:bg-brand-600 disabled:opacity-60 transition-colors">
        {{ busy ? 'Speichere…' : 'Einrichten' }}
      </button>

      <p class="text-xs text-gray-400 mt-4 text-center">
        Achtung: Verwende die App über das Internet ausschliesslich hinter
        HTTPS (z.&nbsp;B. Reverse-Proxy mit TLS).
      </p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { authApi } from '../api'

const password = ref('')
const confirm = ref('')
const error = ref('')
const busy = ref(false)

const submit = async () => {
  error.value = ''
  if (password.value.length < 8) {
    error.value = 'Passwort muss mindestens 8 Zeichen haben'
    return
  }
  if (password.value !== confirm.value) {
    error.value = 'Passwörter stimmen nicht überein'
    return
  }
  busy.value = true
  try {
    await authApi.setup(password.value)
    // Re-load: the next request requires auth, the browser will prompt.
    window.location.reload()
  } catch (e) {
    error.value = e.response?.data?.error || 'Einrichtung fehlgeschlagen'
  } finally {
    busy.value = false
  }
}
</script>
