<template>
  <div>
    <h2 class="text-xl font-semibold mb-6">Einstellungen</h2>

    <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-4 mb-4">
      <h3 class="font-semibold text-gray-700 mb-3">Wohnadresse</h3>

      <label class="block text-sm text-gray-600 mb-1">Adresse</label>
      <input v-model="form.home_address" type="text" placeholder="Musterstrasse 1, 3000 Bern"
        class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500" />

      <label class="block text-sm text-gray-600 mb-1">ÖV-Abfahrtshaltestelle</label>
      <StopSearch v-model="form.home_stop" class="mb-1" />
      <p class="text-xs text-gray-400">Die Haltestelle, von der aus du mit dem ÖV fährst.</p>
    </div>

    <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-4 mb-6">
      <h3 class="font-semibold text-gray-700 mb-3">ÖV-Präferenzen</h3>
      <p class="text-sm text-gray-500">ÖV-Präferenzen werden automatisch basierend auf deinem Profil optimiert.</p>
    </div>

    <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-4 mb-4">
      <h3 class="font-semibold text-gray-700 mb-2">Kalender-Abonnement</h3>
      <p class="text-sm text-gray-500 mb-3">Abonniere deinen Einsatzkalender direkt in Apple Calendar oder Google Calendar.</p>
      <button @click="copyCalendarUrl"
        class="w-full text-center py-2 rounded-lg border border-brand-500 text-brand-500 hover:bg-brand-50 text-sm transition-colors">
        {{ copied ? 'Link kopiert!' : 'WebCal-Link kopieren' }}
      </button>
    </div>

    <button @click="save"
      class="w-full py-3 bg-brand-500 text-white rounded-xl font-semibold hover:bg-brand-600 transition-colors">
      Speichern
    </button>

    <p v-if="saved" class="text-center text-green-600 text-sm mt-3">Einstellungen gespeichert</p>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { settingsApi } from '../api'
import StopSearch from '../components/StopSearch.vue'

const form = ref({
  home_address: '',
  home_stop: '',
  transit_prefs: { exclude_types: [], walking_speed: 'normal' },
})
const saved = ref(false)
const copied = ref(false)

const save = async () => {
  await settingsApi.update(form.value)
  saved.value = true
  setTimeout(() => { saved.value = false }, 2500)
}

const copyCalendarUrl = () => {
  const url = `webcal://${location.host}/api/calendar.ics`
  navigator.clipboard.writeText(url)
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}

onMounted(async () => {
  const s = await settingsApi.get()
  form.value = {
    home_address: s.home_address || '',
    home_stop: s.home_stop || '',
    transit_prefs: s.transit_prefs || { exclude_types: [], walking_speed: 'normal' },
  }
})
</script>
