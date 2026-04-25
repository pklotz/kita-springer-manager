<template>
  <Modal :title="assignment ? 'Einsatz bearbeiten' : 'Neuer Einsatz'" @close="$emit('close')">
    <label class="block text-sm text-gray-600 mb-1">Träger</label>
    <select v-model="selectedProvider"
      class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500">
      <option value="">Alle Träger</option>
      <option v-for="p in providers" :key="p.id" :value="p.id">{{ p.name }}</option>
    </select>

    <label class="block text-sm text-gray-600 mb-1">Kita *</label>
    <div class="relative mb-1">
      <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
      <input v-model="kitaSearch" type="text" placeholder="Kita suchen…"
        class="w-full pl-9 pr-3 py-2 rounded-lg border border-gray-200 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500" />
    </div>
    <select v-model="form.kita_id" size="4"
      class="w-full rounded-lg border border-gray-200 px-3 py-1 mb-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500">
      <option value="" disabled>Bitte wählen…</option>
      <option v-for="k in filteredKitas" :key="k.id" :value="k.id">{{ k.name }}</option>
    </select>

    <template v-if="groupOptions.length">
      <label class="block text-sm text-gray-600 mb-1">Gruppe</label>
      <select v-model="form.group_name"
        class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500">
        <option value="">— keine —</option>
        <option v-for="g in groupOptions" :key="g" :value="g">{{ g }}</option>
      </select>
    </template>

    <label class="block text-sm text-gray-600 mb-1">Datum *</label>
    <input type="date" v-model="form.date"
      class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500" />

    <div class="flex gap-3 mb-3">
      <div class="flex-1">
        <label class="block text-sm text-gray-600 mb-1">Beginn (Soll)</label>
        <TimeSelect v-model="form.start_time" />
      </div>
      <div class="flex-1">
        <label class="block text-sm text-gray-600 mb-1">Ende (Soll)</label>
        <TimeSelect v-model="form.end_time" />
      </div>
    </div>

    <!-- Erfasste (Ist-) Arbeitszeit, nur für heute oder vergangene Einsätze -->
    <div v-if="isPastOrToday" class="mb-3 p-3 rounded-lg bg-amber-50 border border-amber-100">
      <div class="flex items-center justify-between mb-2">
        <label class="text-sm font-medium text-amber-900">Erfasste Arbeitszeit</label>
        <button v-if="!hasActual" type="button" @click="copyPlanToActual"
          class="text-xs text-amber-700 hover:text-amber-900 underline">
          aus Soll übernehmen
        </button>
        <button v-else type="button" @click="clearActual"
          class="text-xs text-amber-700 hover:text-amber-900 underline">
          leeren
        </button>
      </div>
      <div class="grid grid-cols-2 gap-3 mb-2">
        <div>
          <label class="block text-xs text-amber-800 mb-1">Beginn</label>
          <TimeSelect v-model="form.actual_start_time" />
        </div>
        <div>
          <label class="block text-xs text-amber-800 mb-1">Pause ab</label>
          <TimeSelect v-model="form.actual_break_start" />
        </div>
        <div>
          <label class="block text-xs text-amber-800 mb-1">Pause bis</label>
          <TimeSelect v-model="form.actual_break_end" />
        </div>
        <div>
          <label class="block text-xs text-amber-800 mb-1">Ende</label>
          <TimeSelect v-model="form.actual_end_time" />
        </div>
      </div>
      <div v-if="hasActual" class="text-xs text-amber-900 flex items-center gap-3 flex-wrap">
        <span>Arbeit <strong>{{ netHm }} h</strong></span>
        <span>·</span>
        <span :class="breakWarn ? 'text-red-600 font-medium' : ''">
          Pause {{ breakHm }} h
          <span v-if="breakWarn" :title="breakWarnTitle">⚠</span>
        </span>
      </div>
    </div>

    <label class="block text-sm text-gray-600 mb-1">Notizen</label>
    <textarea v-model="form.notes" rows="2"
      class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-4 focus:outline-none focus:ring-2 focus:ring-brand-500" />

    <div class="flex gap-3">
      <button v-if="assignment" @click="remove"
        class="p-2 rounded-lg border border-red-100 text-red-500 hover:bg-red-50 transition-colors">
        <Trash2 class="w-4 h-4" />
      </button>
      <button @click="$emit('close')" class="flex-1 py-2 rounded-lg border border-gray-200 text-gray-600 hover:bg-gray-50">Abbrechen</button>
      <button @click="save" :disabled="!form.kita_id || !form.date"
        class="flex-1 py-2 rounded-lg bg-brand-500 text-white hover:bg-brand-600 disabled:opacity-50 transition-colors">Speichern</button>
    </div>
  </Modal>
</template>

<script setup>
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { Search, Trash2 } from 'lucide-vue-next'
import dayjs from 'dayjs'
import { kitasApi, providersApi, assignmentsApi } from '../api'
import Modal from './Modal.vue'
import TimeSelect from './TimeSelect.vue'
import {
  netWorkMinutes, breakMinutes, grossWorkMinutes,
  requiredBreakMinutes, legalMinBreakMinutes, formatHm,
} from '../utils/time'

const props = defineProps({ assignment: { type: Object, default: null } })
const emit = defineEmits(['close', 'saved', 'deleted'])

const kitas = ref([])
const providers = ref([])
const selectedProvider = ref('')
const kitaSearch = ref('')
const form = ref({
  kita_id: '', date: '',
  start_time: '07:00', end_time: '17:00',
  actual_start_time: '', actual_break_start: '', actual_break_end: '', actual_end_time: '',
  group_name: '',
  notes: '',
})

const isPastOrToday = computed(() =>
  form.value.date && form.value.date <= dayjs().format('YYYY-MM-DD')
)
const hasActual = computed(() => form.value.actual_start_time || form.value.actual_end_time)

const copyPlanToActual = () => {
  form.value.actual_start_time = form.value.start_time
  form.value.actual_end_time = form.value.end_time
  form.value.actual_break_start = ''
  form.value.actual_break_end = ''
}
const clearActual = () => {
  form.value.actual_start_time = ''
  form.value.actual_break_start = ''
  form.value.actual_break_end = ''
  form.value.actual_end_time = ''
}

// Live computations for the Ist-block
const currentProvider = computed(() => {
  const kita = kitas.value.find(k => k.id === form.value.kita_id)
  if (!kita?.provider_id) return null
  return providers.value.find(p => p.id === kita.provider_id) || null
})
const netHm = computed(() => formatHm(netWorkMinutes(
  form.value.actual_start_time, form.value.actual_break_start,
  form.value.actual_break_end, form.value.actual_end_time,
)))
const breakHm = computed(() => formatHm(breakMinutes(
  form.value.actual_break_start, form.value.actual_break_end,
)))
const requiredMin = computed(() => requiredBreakMinutes(
  grossWorkMinutes(form.value.actual_start_time, form.value.actual_end_time),
  currentProvider.value?.min_break_minutes || 0,
))
const actualBreakMin = computed(() => breakMinutes(
  form.value.actual_break_start, form.value.actual_break_end,
))
const breakWarn = computed(() => requiredMin.value > 0 && actualBreakMin.value < requiredMin.value)
const breakWarnTitle = computed(() => {
  const gross = grossWorkMinutes(form.value.actual_start_time, form.value.actual_end_time)
  const legal = legalMinBreakMinutes(gross)
  const provMin = currentProvider.value?.min_break_minutes || 0
  const parts = []
  if (legal > 0) parts.push(`${legal} min laut ArG Art. 15`)
  if (provMin > 0) parts.push(`${provMin} min Trägervorgabe`)
  return `Mindestpause: ${parts.join(', ') || '0'}`
})

const filteredKitas = computed(() => {
  let list = kitas.value
  if (selectedProvider.value) list = list.filter(k => k.provider_id === selectedProvider.value)
  if (kitaSearch.value.trim()) {
    const q = kitaSearch.value.toLowerCase()
    list = list.filter(k => k.name.toLowerCase().includes(q))
  }
  return list
})

watch(selectedProvider, () => {
  if (form.value.kita_id && !filteredKitas.value.find(k => k.id === form.value.kita_id)) {
    form.value.kita_id = ''
  }
})

// Auto-set des Trägers, wenn eine Kita gewählt wird (UX-Spiegel des
// serverseitig autoritativ abgeleiteten provider_id).
watch(() => form.value.kita_id, (id) => {
  if (!id) return
  const kita = kitas.value.find(k => k.id === id)
  if (kita?.provider_id && selectedProvider.value !== kita.provider_id) {
    selectedProvider.value = kita.provider_id
  }
  // Wenn aktuell ausgewählte Gruppe nicht zur neuen Kita gehört, leeren.
  if (form.value.group_name && !groupOptions.value.includes(form.value.group_name)) {
    form.value.group_name = ''
  }
})

// Gruppen-Auswahl basiert auf den Stammdaten der gewählten Kita.
// Falls eine bestehende Zuordnung einen Wert enthält, der nicht (mehr) in
// den Stammdaten steht, wird er trotzdem als Option mitgeführt.
const groupOptions = computed(() => {
  const kita = kitas.value.find(k => k.id === form.value.kita_id)
  const fromKita = kita?.groups || []
  const set = new Set(fromKita)
  if (form.value.group_name && !set.has(form.value.group_name)) {
    return [form.value.group_name, ...fromKita]
  }
  return fromKita
})

// Wenn Ist-Zeit dem alten Soll entsprach (also nie manuell abweichend gesetzt),
// soll sie einer Korrektur des Soll folgen. Sonst nicht anfassen.
// Erst nach onMounted aktiv, damit das initiale form-Setup nicht triggert.
const autoSyncReady = ref(false)
watch(() => form.value.start_time, (newVal, oldVal) => {
  if (!autoSyncReady.value) return
  if (form.value.actual_start_time === oldVal) {
    form.value.actual_start_time = newVal
  }
})
watch(() => form.value.end_time, (newVal, oldVal) => {
  if (!autoSyncReady.value) return
  if (form.value.actual_end_time === oldVal) {
    form.value.actual_end_time = newVal
  }
})

const save = async () => {
  if (props.assignment) {
    await assignmentsApi.update(props.assignment.id, form.value)
  } else {
    await assignmentsApi.create(form.value)
  }
  emit('saved')
}

const remove = async () => {
  if (confirm('Einsatz wirklich löschen?')) {
    await assignmentsApi.delete(props.assignment.id)
    // Distinct from 'saved' so the parent doesn't refetch an entity that no
    // longer exists — that 404 used to bubble up as a misleading toast.
    emit('deleted')
  }
}

onMounted(async () => {
  [kitas.value, providers.value] = await Promise.all([kitasApi.list(), providersApi.list()])

  if (props.assignment) {
    const a = props.assignment
    form.value = {
      kita_id: a.kita_id || '',
      date: a.date,
      start_time: a.start_time || '',
      end_time: a.end_time || '',
      actual_start_time: a.actual_start_time || '',
      actual_break_start: a.actual_break_start || '',
      actual_break_end: a.actual_break_end || '',
      actual_end_time: a.actual_end_time || '',
      group_name: a.group_name || '',
      notes: a.notes || '',
    }
    const kita = kitas.value.find(k => k.id === a.kita_id)
    if (kita?.provider_id) selectedProvider.value = kita.provider_id

    // Pre-fill actual from planned when recording hours for today or a past day
    if (isPastOrToday.value && !hasActual.value) {
      copyPlanToActual()
    }
  }
  await nextTick()
  autoSyncReady.value = true
})
</script>
