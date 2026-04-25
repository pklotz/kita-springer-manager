<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 px-4">
    <div class="w-full max-w-sm bg-white rounded-2xl shadow-md border border-gray-100 p-6">
      <h1 class="text-xl font-semibold text-gray-800 mb-1">Kita Springer</h1>
      <p class="text-sm text-gray-500 mb-5">Anmelden mit deinem Passwort.</p>

      <label class="block text-sm text-gray-600 mb-1">Passwort</label>
      <input ref="pwInput" v-model="password" type="password" autocomplete="current-password"
        @keyup.enter="submit"
        class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500" />

      <p v-if="error" class="text-sm text-red-600 mb-3">{{ error }}</p>

      <button @click="submit" :disabled="busy || !password"
        class="w-full py-2.5 bg-brand-500 text-white rounded-xl font-semibold hover:bg-brand-600 disabled:opacity-60 transition-colors">
        {{ busy ? 'Prüfe…' : 'Anmelden' }}
      </button>

      <p class="text-xs text-gray-400 mt-4 text-center">
        Probleme beim Anmelden?
        <a href="/api/auth/reset" class="underline hover:text-gray-600">App-Cache zurücksetzen</a>
      </p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { authApi } from '../api'

const password = ref('')
const error = ref('')
const busy = ref(false)
const pwInput = ref(null)

const submit = async () => {
  if (busy.value || !password.value) return
  busy.value = true
  error.value = ''
  try {
    await authApi.login(password.value)
    // Reload so main.js re-bootstraps and mounts the full App with axios
    // already carrying the new Authorization header.
    window.location.reload()
  } catch (e) {
    error.value = e.message || 'Anmeldung fehlgeschlagen'
    password.value = ''
    pwInput.value?.focus()
  } finally {
    busy.value = false
  }
}

onMounted(() => pwInput.value?.focus())
</script>
