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
        Alle Einsätze als iCal-Datei herunterladen oder als Abo-Link am iPhone abonnieren.
      </p>
      <div class="flex gap-2 mb-3">
        <button @click="downloadICS" :disabled="icsBusy"
          class="flex-1 py-2 rounded-lg bg-brand-500 text-white hover:bg-brand-600 disabled:opacity-60 text-sm transition-colors">
          {{ icsBusy ? '…' : '.ics herunterladen' }}
        </button>
        <button @click="copyCalendarUrl" :disabled="!webcalUrl"
          class="flex-1 py-2 rounded-lg border border-brand-500 text-brand-500 hover:bg-brand-50 disabled:opacity-60 text-sm transition-colors">
          {{ copied ? 'Link kopiert!' : 'WebCal-Link kopieren' }}
        </button>
      </div>
      <p class="text-xs text-gray-400 mb-2">
        Der WebCal-Link enthält ein Geheim-Token, das nur den Kalender freigibt — nicht das Passwort.
        Falls du ihn weitergegeben hast und zurückziehen willst, neuen Token erzeugen.
      </p>
      <button @click="rotateToken" class="text-xs text-gray-500 underline hover:text-gray-700">
        Token neu erzeugen
      </button>
      <p class="text-xs text-gray-400 mt-2">
        WebCal-Abo funktioniert nur über einen echten Hostnamen — nicht über localhost.
      </p>
    </div>

    <button @click="save"
      class="w-full py-3 bg-brand-500 text-white rounded-xl font-semibold hover:bg-brand-600 transition-colors">
      Speichern
    </button>

    <p v-if="saved" class="text-center text-green-600 text-sm mt-3">Einstellungen gespeichert</p>

    <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-4 mt-6">
      <h3 class="font-semibold text-gray-700 mb-3">Passwort ändern</h3>
      <p class="text-xs text-gray-400 mb-3">
        Benutzername: <code class="px-1 bg-gray-100 rounded">admin</code>
      </p>

      <label class="block text-sm text-gray-600 mb-1">Aktuelles Passwort</label>
      <input v-model="pw.old" type="password" autocomplete="current-password"
        class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500" />

      <label class="block text-sm text-gray-600 mb-1">Neues Passwort (mind. 8 Zeichen)</label>
      <input v-model="pw.new" type="password" autocomplete="new-password"
        class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500" />

      <label class="block text-sm text-gray-600 mb-1">Neues Passwort bestätigen</label>
      <input v-model="pw.confirm" type="password" autocomplete="new-password"
        class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-4 focus:outline-none focus:ring-2 focus:ring-brand-500" />

      <p v-if="pw.error" class="text-sm text-red-600 mb-3">{{ pw.error }}</p>
      <p v-if="pw.success" class="text-sm text-green-600 mb-3">Passwort aktualisiert — beim nächsten Reload anmelden.</p>

      <button @click="changePassword" :disabled="pw.busy"
        class="w-full py-2.5 bg-gray-700 text-white rounded-xl font-medium hover:bg-gray-800 disabled:opacity-60 transition-colors">
        {{ pw.busy ? 'Aktualisiere…' : 'Passwort aktualisieren' }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { settingsApi, authApi, downloadsApi } from '../api'
import StopSearch from '../components/StopSearch.vue'

const pw = ref({
  old: '', new: '', confirm: '',
  error: '', success: false, busy: false,
})

const changePassword = async () => {
  pw.value.error = ''
  pw.value.success = false
  if (pw.value.new.length < 8) {
    pw.value.error = 'Neues Passwort muss mindestens 8 Zeichen haben'
    return
  }
  if (pw.value.new !== pw.value.confirm) {
    pw.value.error = 'Passwörter stimmen nicht überein'
    return
  }
  pw.value.busy = true
  try {
    await authApi.changePassword(pw.value.old, pw.value.new)
    pw.value.success = true
    pw.value.old = pw.value.new = pw.value.confirm = ''
  } catch (e) {
    pw.value.error = e.response?.data?.error || 'Aktualisierung fehlgeschlagen'
  } finally {
    pw.value.busy = false
  }
}

const form = ref({
  home_address: '',
  home_stop: '',
  user_name: '',
  canton: 'BE',
  transit_prefs: { exclude_types: [], walking_speed: 'normal' },
})
const saved = ref(false)
const copied = ref(false)
const icsBusy = ref(false)
const downloadToken = ref('')

// WebCal subscription URL embeds the long-random download token. The host is
// the user-facing hostname (whatever they typed in the address bar), so iCal
// clients hit the same endpoint they would in a browser.
const webcalUrl = computed(() => {
  if (!downloadToken.value) return ''
  return `webcal://${location.host}/api/calendar.ics?token=${downloadToken.value}`
})

const downloadICS = async () => {
  if (icsBusy.value) return
  icsBusy.value = true
  try { await downloadsApi.calendarICS() } finally { icsBusy.value = false }
}

const rotateToken = async () => {
  if (!confirm('Bestehende Kalender-Abos werden ungültig. Fortfahren?')) return
  downloadToken.value = await authApi.regenerateDownloadToken()
}

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
  if (!webcalUrl.value) return
  navigator.clipboard.writeText(webcalUrl.value)
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}

onMounted(async () => {
  const [s, tok] = await Promise.all([
    settingsApi.get(),
    authApi.getDownloadToken().catch(() => ''),
  ])
  form.value = {
    home_address: s.home_address || '',
    home_stop: s.home_stop || '',
    user_name: s.user_name || '',
    canton: s.canton || 'BE',
    transit_prefs: s.transit_prefs || { exclude_types: [], walking_speed: 'normal' },
  }
  downloadToken.value = tok
})
</script>
