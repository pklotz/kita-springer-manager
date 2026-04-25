<template>
  <div>
    <h2 class="text-xl font-semibold mb-6">Einstellungen</h2>

    <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-4 mb-4">
      <h3 class="font-semibold text-gray-700 mb-3">Persönlich</h3>

      <label class="block text-sm text-gray-600 mb-1">Name im Einsatzplan</label>
      <input v-model="form.user_name" type="text" placeholder="Natalia Majer"
        class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-1 focus:outline-none focus:ring-2 focus:ring-brand-500" />
      <p class="text-xs text-gray-400">Voller Name — beim Import wird sowohl der ganze Name als auch jeder Vor-/Nachname einzeln erkannt.</p>
    </div>

    <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-4 mb-4">
      <h3 class="font-semibold text-gray-700 mb-3">Wohnadresse</h3>

      <label class="block text-sm text-gray-600 mb-1">Adresse</label>
      <input v-model="form.home_address" type="text" placeholder="Musterstrasse 1, 3000 Bern"
        class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500" />

      <label class="block text-sm text-gray-600 mb-1">ÖV-Abfahrtshaltestelle</label>
      <StopSearch v-model="form.home_stop" class="mb-1" />
      <p class="text-xs text-gray-400">Die Haltestelle, von der aus du mit dem ÖV fährst.</p>
    </div>

    <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-4 mb-4">
      <h3 class="font-semibold text-gray-700 mb-3">Feiertage</h3>
      <label class="block text-sm text-gray-600 mb-1">Kanton</label>
      <select v-model="form.canton"
        class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-1 focus:outline-none focus:ring-2 focus:ring-brand-500">
        <option v-for="c in cantons" :key="c.code" :value="c.code">{{ c.code }} – {{ c.name }}</option>
      </select>
      <p class="text-xs text-gray-400">Bestimmt die gesetzlichen Feiertage im Kalender.</p>
    </div>

    <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-4 mb-6">
      <h3 class="font-semibold text-gray-700 mb-3">ÖV-Präferenzen</h3>
      <p class="text-sm text-gray-500">ÖV-Präferenzen werden automatisch basierend auf deinem Profil optimiert.</p>
    </div>

    <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-4 mb-4">
      <h3 class="font-semibold text-gray-700 mb-2">Kalender-Export</h3>
      <p class="text-sm text-gray-500 mb-3">
        Alle Einsätze als iCal-Datei herunterladen oder als Abo-Link kopieren.
      </p>
      <div class="flex gap-2">
        <a :href="icsUrl" download="kita-einsaetze.ics"
          class="flex-1 text-center py-2 rounded-lg bg-brand-500 text-white hover:bg-brand-600 text-sm transition-colors">
          .ics herunterladen
        </a>
        <button @click="copyCalendarUrl"
          class="flex-1 py-2 rounded-lg border border-brand-500 text-brand-500 hover:bg-brand-50 text-sm transition-colors">
          {{ copied ? 'Link kopiert!' : 'WebCal-Link kopieren' }}
        </button>
      </div>
      <p class="text-xs text-gray-400 mt-2">
        WebCal-Abo funktioniert nur, wenn der Server über einen echten Hostnamen erreichbar ist — nicht über localhost.
      </p>
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
  user_name: '',
  canton: 'BE',
  transit_prefs: { exclude_types: [], walking_speed: 'normal' },
})
const saved = ref(false)
const copied = ref(false)
const icsUrl = `${location.protocol}//${location.host}/api/calendar.ics`

const cantons = [
  { code: 'AG', name: 'Aargau' },
  { code: 'AI', name: 'Appenzell Innerrhoden' },
  { code: 'AR', name: 'Appenzell Ausserrhoden' },
  { code: 'BE', name: 'Bern' },
  { code: 'BL', name: 'Basel-Landschaft' },
  { code: 'BS', name: 'Basel-Stadt' },
  { code: 'FR', name: 'Freiburg' },
  { code: 'GE', name: 'Genf' },
  { code: 'GL', name: 'Glarus' },
  { code: 'GR', name: 'Graubünden' },
  { code: 'JU', name: 'Jura' },
  { code: 'LU', name: 'Luzern' },
  { code: 'NE', name: 'Neuenburg' },
  { code: 'NW', name: 'Nidwalden' },
  { code: 'OW', name: 'Obwalden' },
  { code: 'SG', name: 'St. Gallen' },
  { code: 'SH', name: 'Schaffhausen' },
  { code: 'SO', name: 'Solothurn' },
  { code: 'SZ', name: 'Schwyz' },
  { code: 'TG', name: 'Thurgau' },
  { code: 'TI', name: 'Tessin' },
  { code: 'UR', name: 'Uri' },
  { code: 'VD', name: 'Waadt' },
  { code: 'VS', name: 'Wallis' },
  { code: 'ZG', name: 'Zug' },
  { code: 'ZH', name: 'Zürich' },
]

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
    user_name: s.user_name || '',
    canton: s.canton || 'BE',
    transit_prefs: s.transit_prefs || { exclude_types: [], walking_speed: 'normal' },
  }
})
</script>
