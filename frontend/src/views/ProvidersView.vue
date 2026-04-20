<template>
  <div>
    <!-- ── Träger-Verwaltung (collapsible) ─────────────────── -->
    <details class="mb-6 group" :open="providers.length === 0">
      <summary class="flex items-center justify-between cursor-pointer select-none py-1 mb-1">
        <h2 class="text-xl font-semibold">Träger</h2>
        <div class="flex items-center gap-3">
          <button @click.prevent="openForm(null)"
            class="flex items-center gap-1 text-sm bg-brand-500 text-white px-3 py-1.5 rounded-lg hover:bg-brand-600 transition-colors">
            <Plus class="w-4 h-4" /> Neuer Träger
          </button>
          <ChevronDown class="w-4 h-4 text-gray-400 transition-transform group-open:rotate-180" />
        </div>
      </summary>

      <div class="space-y-3 mt-3">
        <div v-if="providers.length === 0" class="text-sm text-gray-400 py-4 text-center">
          Noch keine Träger erfasst
        </div>

        <div v-for="p in providers" :key="p.id"
          class="bg-white rounded-xl shadow-sm border border-gray-100">
          <div class="p-4 flex items-center justify-between">
            <div class="flex items-center gap-3">
              <span class="w-4 h-4 rounded-full shadow-sm shrink-0" :style="{ backgroundColor: p.color_hex }" />
              <span class="font-semibold text-gray-800">{{ p.name }}</span>
              <span class="text-xs text-gray-400">{{ kitasOf(p.id).length }} Kitas</span>
            </div>
            <div class="flex gap-2 items-center">
              <button @click="openForm(p)" class="p-1.5 rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600">
                <Pencil class="w-4 h-4" />
              </button>
              <button @click="confirmDelete(p)" class="p-1.5 rounded-lg hover:bg-red-50 text-gray-400 hover:text-red-500">
                <Trash2 class="w-4 h-4" />
              </button>
            </div>
          </div>

          <div class="px-4 pb-4 flex flex-wrap gap-2 border-t border-gray-50 pt-3">
            <label class="flex items-center gap-1.5 text-xs bg-blue-50 hover:bg-blue-100 text-blue-700 px-3 py-1.5 rounded-lg cursor-pointer transition-colors">
              <Upload class="w-3.5 h-3.5" /> Kitas importieren
              <input type="file" accept=".xlsx" class="hidden" @change="importKitasExcel(p, $event)" />
            </label>
            <label class="flex items-center gap-1.5 text-xs bg-indigo-50 hover:bg-indigo-100 text-indigo-700 px-3 py-1.5 rounded-lg cursor-pointer transition-colors">
              <FileSpreadsheet class="w-3.5 h-3.5" /> Einsatz-Excel
              <input type="file" accept=".xlsx" class="hidden" @change="importExcel(p, $event)" />
            </label>
            <button @click="openRecurring(p)"
              class="flex items-center gap-1.5 text-xs bg-purple-50 hover:bg-purple-100 text-purple-700 px-3 py-1.5 rounded-lg transition-colors">
              <Repeat class="w-3.5 h-3.5" /> Fixe Einsätze
            </button>
          </div>

          <div v-if="importResult[p.id]" class="px-4 pb-3">
            <div class="text-xs rounded-lg px-3 py-2"
              :class="importResult[p.id].warnings?.length ? 'bg-amber-50 text-amber-700' : 'bg-green-50 text-green-700'">
              <template v-if="importResult[p.id].type === 'kitas'">
                {{ importResult[p.id].imported }} Kitas importiert
              </template>
              <template v-else>
                {{ importResult[p.id].created }} neu · {{ importResult[p.id].updated }} aktualisiert · {{ importResult[p.id].skipped }} übersprungen
              </template>
              <span v-if="importResult[p.id].warnings?.length" class="ml-2 text-orange-600">
                · {{ importResult[p.id].warnings.length }} Warnung(en)
              </span>
            </div>
          </div>
        </div>
      </div>
    </details>

    <!-- ── Kita-Verzeichnis ─────────────────────────────────── -->
    <div class="flex items-center justify-between mb-3">
      <h2 class="text-xl font-semibold">Kita-Verzeichnis</h2>
      <button @click="openKitaForm(null)"
        class="flex items-center gap-1 text-sm bg-brand-500 text-white px-3 py-1.5 rounded-lg hover:bg-brand-600 transition-colors">
        <Plus class="w-4 h-4" /> Neue Kita
      </button>
    </div>

    <!-- Search + filter -->
    <div class="flex gap-2 mb-4">
      <div class="relative flex-1">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
        <input v-model="search" type="text" placeholder="Kita suchen…"
          class="w-full pl-9 pr-3 py-2 rounded-lg border border-gray-200 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500" />
      </div>
      <select v-if="providers.length" v-model="filterProvider"
        class="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500">
        <option value="">Alle Träger</option>
        <option v-for="p in providers" :key="p.id" :value="p.id">{{ p.name }}</option>
      </select>
    </div>

    <div v-if="filteredKitas.length === 0" class="text-center text-gray-400 py-12">
      {{ search ? `Keine Kitas gefunden für „${search}"` : 'Noch keine Kitas vorhanden' }}
    </div>

    <!-- Kita list -->
    <div class="space-y-2">
      <button v-for="k in filteredKitas" :key="k.id"
        @click="selectKita(k)"
        :class="['w-full text-left bg-white rounded-xl border px-4 py-3 transition-all hover:shadow-md',
          selectedKita?.id === k.id ? 'border-brand-500 shadow-md ring-1 ring-brand-500' : 'border-gray-100 shadow-sm']">
        <div class="flex items-start justify-between gap-2">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 flex-wrap">
              <span class="font-semibold text-gray-800 truncate">{{ k.name }}</span>
              <span v-if="providerOf(k)" class="text-xs px-2 py-0.5 rounded-full text-white shrink-0"
                :style="{ backgroundColor: providerOf(k)?.color_hex }">
                {{ providerOf(k)?.name }}
              </span>
            </div>
            <div v-if="k.address" class="text-sm text-gray-500 truncate mt-0.5">{{ k.address }}</div>
            <div class="flex items-center gap-3 mt-1 flex-wrap">
              <span v-if="k.stop_name" class="flex items-center gap-1 text-xs text-brand-500">
                <Bus class="w-3 h-3" /> {{ k.stop_name }}
              </span>
              <span v-if="k.phone" class="flex items-center gap-1 text-xs text-gray-400">
                <Phone class="w-3 h-3" /> {{ k.phone }}
              </span>
              <span v-if="k.groups?.length" class="text-xs text-gray-400">
                {{ k.groups.length }} Gruppe{{ k.groups.length !== 1 ? 'n' : '' }}
              </span>
            </div>
          </div>
          <ChevronRight class="w-4 h-4 text-gray-300 mt-1 shrink-0" />
        </div>
      </button>
    </div>

    <!-- ── Kita Detail Pane (right slide-in) ──────────────────── -->
    <Transition name="panel">
      <div v-if="selectedKita" class="fixed inset-0 z-40 flex" @click.self="selectedKita = null">
        <!-- Backdrop (clicking closes panel) -->
        <div class="flex-1" @click="selectedKita = null" />

        <!-- Panel -->
        <div class="w-80 max-w-[90vw] bg-white shadow-2xl h-full overflow-y-auto flex flex-col"
          @click.stop>
          <!-- Header -->
          <div class="flex items-start justify-between p-5 border-b border-gray-100 sticky top-0 bg-white z-10">
            <div>
              <h3 class="font-bold text-lg leading-tight text-gray-900">{{ selectedKita.name }}</h3>
              <span v-if="providerOf(selectedKita)" class="inline-block mt-1 text-xs px-2 py-0.5 rounded-full text-white"
                :style="{ backgroundColor: providerOf(selectedKita)?.color_hex }">
                {{ providerOf(selectedKita)?.name }}
              </span>
            </div>
            <button @click="selectedKita = null"
              class="p-1.5 rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors shrink-0 ml-2">
              <X class="w-5 h-5" />
            </button>
          </div>

          <!-- Body -->
          <div class="p-5 space-y-5 flex-1">
            <!-- Photo -->
            <img v-if="selectedKita.photo_url" :src="selectedKita.photo_url" :alt="selectedKita.name"
              class="w-full h-40 object-cover rounded-xl" />

            <!-- Address -->
            <div v-if="selectedKita.address">
              <div class="flex items-center gap-1.5 text-xs font-semibold text-gray-400 uppercase tracking-wide mb-1.5">
                <MapPin class="w-3.5 h-3.5" /> Adresse
              </div>
              <p class="text-sm text-gray-700">{{ selectedKita.address }}</p>
              <a :href="mapsUrl(selectedKita.address)" target="_blank" rel="noopener"
                class="inline-flex items-center gap-1.5 mt-2 text-xs text-brand-500 hover:text-brand-600 transition-colors">
                <ExternalLink class="w-3.5 h-3.5" /> In Google Maps öffnen
              </a>
            </div>

            <!-- Leitung -->
            <div v-if="selectedKita.leitung_name">
              <div class="flex items-center gap-1.5 text-xs font-semibold text-gray-400 uppercase tracking-wide mb-1.5">
                <Users class="w-3.5 h-3.5" /> Leitung
              </div>
              <p class="text-sm text-gray-700">{{ selectedKita.leitung_name }}</p>
            </div>

            <!-- Contact -->
            <div v-if="selectedKita.phone || selectedKita.email">
              <div class="flex items-center gap-1.5 text-xs font-semibold text-gray-400 uppercase tracking-wide mb-1.5">
                <Phone class="w-3.5 h-3.5" /> Kontakt
              </div>
              <a v-if="selectedKita.phone" :href="`tel:${selectedKita.phone}`"
                class="flex items-center gap-2 text-sm text-gray-700 hover:text-brand-500 transition-colors py-1">
                <Phone class="w-4 h-4 text-gray-400" />
                {{ selectedKita.phone }}
              </a>
              <a v-if="selectedKita.email" :href="`mailto:${selectedKita.email}`"
                class="flex items-center gap-2 text-sm text-gray-700 hover:text-brand-500 transition-colors py-1 break-all">
                <Mail class="w-4 h-4 text-gray-400 shrink-0" />
                {{ selectedKita.email }}
              </a>
            </div>

            <!-- ÖV -->
            <div v-if="selectedKita.stop_name">
              <div class="flex items-center gap-1.5 text-xs font-semibold text-gray-400 uppercase tracking-wide mb-1.5">
                <Bus class="w-3.5 h-3.5" /> ÖV-Haltestelle
              </div>
              <p class="text-sm text-gray-700">{{ selectedKita.stop_name }}</p>
            </div>

            <!-- Groups -->
            <div v-if="selectedKita.groups?.length">
              <div class="flex items-center gap-1.5 text-xs font-semibold text-gray-400 uppercase tracking-wide mb-2">
                <Users class="w-3.5 h-3.5" /> Gruppen
              </div>
              <div class="flex flex-wrap gap-1.5">
                <span v-for="g in selectedKita.groups" :key="g"
                  class="text-xs bg-gray-100 text-gray-600 px-2.5 py-1 rounded-full">{{ g }}</span>
              </div>
            </div>

            <!-- Notes -->
            <div v-if="selectedKita.notes">
              <div class="flex items-center gap-1.5 text-xs font-semibold text-gray-400 uppercase tracking-wide mb-1.5">
                <FileText class="w-3.5 h-3.5" /> Notizen
              </div>
              <p class="text-sm text-gray-600 whitespace-pre-line">{{ selectedKita.notes }}</p>
            </div>
          </div>

          <!-- Footer actions -->
          <div class="p-4 border-t border-gray-100 flex gap-2">
            <button @click="openKitaForm(selectedKita)"
              class="flex-1 flex items-center justify-center gap-1.5 text-sm border border-gray-200 text-gray-600 rounded-lg py-2 hover:bg-gray-50 transition-colors">
              <Pencil class="w-4 h-4" /> Bearbeiten
            </button>
            <button @click="deleteKita(selectedKita)"
              class="flex items-center justify-center gap-1.5 text-sm border border-red-100 text-red-500 rounded-lg py-2 px-3 hover:bg-red-50 transition-colors">
              <Trash2 class="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- ── Provider form modal ─────────────────────────────────── -->
    <div v-if="showForm" class="fixed inset-0 bg-black/40 flex items-end sm:items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl w-full max-w-lg p-6 shadow-xl">
        <h3 class="font-semibold text-lg mb-4">{{ editing ? 'Träger bearbeiten' : 'Neuer Träger' }}</h3>

        <label class="block text-sm text-gray-600 mb-1">Name *</label>
        <input v-model="form.name" type="text" placeholder="Kitas Stadt Bern"
          class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500" />

        <label class="block text-sm text-gray-600 mb-2">Kalenderfarbe</label>
        <div class="flex gap-2 mb-3 flex-wrap">
          <button v-for="c in colors" :key="c" @click="form.color_hex = c"
            :style="{ backgroundColor: c }"
            :class="['w-8 h-8 rounded-full shadow-sm transition-transform', form.color_hex === c ? 'scale-125 ring-2 ring-offset-2 ring-gray-400' : 'hover:scale-110']" />
          <input type="color" v-model="form.color_hex" class="w-8 h-8 rounded-full cursor-pointer border-0" />
        </div>

        <label class="block text-sm text-gray-600 mb-1">Person (Name im Excel)</label>
        <input v-model="form.excel_config.person_name" type="text" placeholder="Natalia"
          class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500" />

        <details class="mb-4">
          <summary class="text-sm text-gray-500 cursor-pointer hover:text-gray-700">Erweiterte Excel-Konfiguration</summary>
          <div class="mt-3 grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs text-gray-500 mb-1">Kopfzeile (Wochentage)</label>
              <input v-model.number="form.excel_config.header_row" type="number" min="1"
                class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500" />
            </div>
            <div>
              <label class="block text-xs text-gray-500 mb-1">Kita/Gruppe-Zeile</label>
              <input v-model.number="form.excel_config.kita_row" type="number" min="1"
                class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500" />
            </div>
            <div>
              <label class="block text-xs text-gray-500 mb-1">Erste Tag-Spalte</label>
              <input v-model="form.excel_config.first_day_col" type="text" placeholder="B"
                class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500" />
            </div>
            <div>
              <label class="block text-xs text-gray-500 mb-1">Spalten pro Tag</label>
              <input v-model.number="form.excel_config.cols_per_day" type="number" min="1"
                class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500" />
            </div>
          </div>
        </details>

        <div class="flex gap-3">
          <button @click="showForm = false" class="flex-1 py-2 rounded-lg border border-gray-200 text-gray-600 hover:bg-gray-50">Abbrechen</button>
          <button @click="save" :disabled="!form.name"
            class="flex-1 py-2 rounded-lg bg-brand-500 text-white hover:bg-brand-600 disabled:opacity-50 transition-colors">Speichern</button>
        </div>
      </div>
    </div>

    <!-- ── Kita form modal ─────────────────────────────────────── -->
    <div v-if="showKitaForm" class="fixed inset-0 bg-black/40 flex items-end sm:items-center justify-center z-50 p-4">
      <div class="bg-white rounded-2xl w-full max-w-md p-6 shadow-xl max-h-[90vh] overflow-y-auto">
        <h3 class="font-semibold text-lg mb-4">{{ editingKita ? 'Kita bearbeiten' : 'Neue Kita' }}</h3>

        <label class="block text-sm text-gray-600 mb-1">Name *</label>
        <input v-model="kitaForm.name" type="text" placeholder="Kita Sonnenschein"
          class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500" />

        <label class="block text-sm text-gray-600 mb-1">Träger</label>
        <select v-model="kitaForm.provider_id"
          class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500">
          <option value="">– kein Träger –</option>
          <option v-for="p in providers" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>

        <label class="block text-sm text-gray-600 mb-1">Adresse</label>
        <input v-model="kitaForm.address" type="text" placeholder="Musterstrasse 1, 3000 Bern"
          class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500" />

        <label class="block text-sm text-gray-600 mb-1">ÖV-Haltestelle</label>
        <StopSearch v-model="kitaForm.stop_name" class="mb-3" />

        <div class="flex gap-3 mb-3">
          <div class="flex-1">
            <label class="block text-sm text-gray-600 mb-1">Telefon</label>
            <input v-model="kitaForm.phone" type="tel"
              class="w-full rounded-lg border border-gray-200 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500" />
          </div>
          <div class="flex-1">
            <label class="block text-sm text-gray-600 mb-1">Email</label>
            <input v-model="kitaForm.email" type="email"
              class="w-full rounded-lg border border-gray-200 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500" />
          </div>
        </div>

        <label class="block text-sm text-gray-600 mb-1">Leitung (Name)</label>
        <input v-model="kitaForm.leitung_name" type="text" placeholder="Maria Muster"
          class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500" />

        <label class="block text-sm text-gray-600 mb-1">Foto-URL</label>
        <input v-model="kitaForm.photo_url" type="url" placeholder="https://…"
          class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500" />

        <label class="block text-sm text-gray-600 mb-1">Gruppen (eine pro Zeile)</label>
        <textarea v-model="groupsText" rows="3" placeholder="Gruppe Sonnenschein&#10;Gruppe Mondschein"
          class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500" />

        <label class="block text-sm text-gray-600 mb-1">Notizen</label>
        <textarea v-model="kitaForm.notes" rows="2"
          class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-4 focus:outline-none focus:ring-2 focus:ring-brand-500" />

        <div class="flex gap-3">
          <button @click="showKitaForm = false" class="flex-1 py-2 rounded-lg border border-gray-200 text-gray-600 hover:bg-gray-50">Abbrechen</button>
          <button @click="saveKita" :disabled="!kitaForm.name"
            class="flex-1 py-2 rounded-lg bg-brand-500 text-white hover:bg-brand-600 disabled:opacity-50 transition-colors">Speichern</button>
        </div>
      </div>
    </div>

    <RecurringForm v-if="recurringProvider" :provider="recurringProvider" @close="recurringProvider = null" />
  </div>
</template>

<script setup>
import { ref, computed, reactive, onMounted } from 'vue'
import {
  Plus, Pencil, Trash2, Upload, Repeat, FileSpreadsheet,
  ChevronDown, ChevronRight, Search,
  MapPin, Phone, Mail, Bus, Users, FileText, X, ExternalLink,
} from 'lucide-vue-next'
import { providersApi, kitasApi } from '../api'
import StopSearch from '../components/StopSearch.vue'
import RecurringForm from '../components/RecurringForm.vue'

const providers = ref([])
const kitas = ref([])
const selectedKita = ref(null)
const search = ref('')
const filterProvider = ref('')
const showForm = ref(false)
const editing = ref(null)
const showKitaForm = ref(false)
const editingKita = ref(null)
const groupsText = ref('')
const recurringProvider = ref(null)
const importResult = reactive({})

const colors = ['#6366f1','#2563eb','#16a34a','#dc2626','#ea580c','#9333ea','#0891b2','#db2777']

// ── Provider form ────────────────────────────────────────────
const defaultProviderForm = () => ({
  name: '', color_hex: '#6366f1',
  excel_config: { person_name: '', header_row: 2, kita_row: 3, first_day_col: 'B', cols_per_day: 2, days_per_week: 5, kita_mapping: {} },
})
const form = ref(defaultProviderForm())

const openForm = (p) => {
  editing.value = p
  form.value = p
    ? { ...p, excel_config: { ...defaultProviderForm().excel_config, ...p.excel_config } }
    : defaultProviderForm()
  showForm.value = true
}

const save = async () => {
  editing.value
    ? await providersApi.update(editing.value.id, form.value)
    : await providersApi.create(form.value)
  showForm.value = false
  load()
}

const confirmDelete = async (p) => {
  if (confirm(`Träger "${p.name}" löschen?`)) {
    await providersApi.delete(p.id)
    load()
  }
}

// ── Kita form ─────────────────────────────────────────────────
const kitaForm = ref({ name: '', provider_id: '', address: '', stop_name: '', phone: '', email: '', leitung_name: '', photo_url: '', notes: '', groups: [] })

const openKitaForm = (k) => {
  editingKita.value = k
  if (k) {
    kitaForm.value = { ...k }
    groupsText.value = (k.groups || []).join('\n')
  } else {
    kitaForm.value = { name: '', provider_id: filterProvider.value || '', address: '', stop_name: '', phone: '', email: '', leitung_name: '', photo_url: '', notes: '', groups: [] }
    groupsText.value = ''
  }
  showKitaForm.value = true
}

const saveKita = async () => {
  kitaForm.value.groups = groupsText.value.split('\n').map(g => g.trim()).filter(Boolean)
  if (editingKita.value) {
    await kitasApi.update(editingKita.value.id, kitaForm.value)
    if (selectedKita.value?.id === editingKita.value.id) {
      selectedKita.value = { ...kitaForm.value, id: editingKita.value.id }
    }
  } else {
    await kitasApi.create(kitaForm.value)
  }
  showKitaForm.value = false
  load()
}

const deleteKita = async (k) => {
  if (confirm(`Kita "${k.name}" löschen?`)) {
    await kitasApi.delete(k.id)
    selectedKita.value = null
    load()
  }
}

// ── Helpers ───────────────────────────────────────────────────
const providerOf = (k) => providers.value.find(p => p.id === k?.provider_id)
const kitasOf = (pid) => kitas.value.filter(k => k.provider_id === pid)
const mapsUrl = (address) => `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(address)}`

const filteredKitas = computed(() => {
  let list = kitas.value
  if (filterProvider.value) list = list.filter(k => k.provider_id === filterProvider.value)
  if (search.value.trim()) {
    const q = search.value.toLowerCase()
    list = list.filter(k =>
      k.name.toLowerCase().includes(q) ||
      k.address?.toLowerCase().includes(q) ||
      k.stop_name?.toLowerCase().includes(q) ||
      k.groups?.some(g => g.toLowerCase().includes(q))
    )
  }
  return list
})

const selectKita = (k) => {
  selectedKita.value = selectedKita.value?.id === k.id ? null : k
}

// ── Imports ───────────────────────────────────────────────────
const importKitasExcel = async (p, event) => {
  const file = event.target.files[0]; if (!file) return
  const fd = new FormData(); fd.append('file', file)
  const res = await fetch(`/api/kitas/import?provider_id=${p.id}`, { method: 'POST', body: fd })
  importResult[p.id] = { ...await res.json(), type: 'kitas' }
  setTimeout(() => { delete importResult[p.id] }, 8000)
  event.target.value = ''
  load()
}

const importExcel = async (p, event) => {
  const file = event.target.files[0]; if (!file) return
  try {
    const result = await providersApi.importExcel(p.id, file, new Date().getFullYear())
    importResult[p.id] = result
    setTimeout(() => { delete importResult[p.id] }, 8000)
  } catch (e) { alert('Import-Fehler: ' + (e.response?.data?.error || e.message)) }
  event.target.value = ''
}

const load = async () => {
  [providers.value, kitas.value] = await Promise.all([providersApi.list(), kitasApi.list()])
}
onMounted(load)
</script>

<style scoped>
.panel-enter-active,
.panel-leave-active {
  transition: opacity 0.2s ease;
}
.panel-enter-active > div:last-child,
.panel-leave-active > div:last-child {
  transition: transform 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}
.panel-enter-from,
.panel-leave-to {
  opacity: 0;
}
.panel-enter-from > div:last-child,
.panel-leave-to > div:last-child {
  transform: translateX(100%);
}
</style>
