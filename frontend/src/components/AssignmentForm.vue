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
      <div class="flex gap-3">
        <div class="flex-1">
          <label class="block text-xs text-amber-800 mb-1">Beginn (Ist)</label>
          <TimeSelect v-model="form.actual_start_time" />
        </div>
        <div class="flex-1">
          <label class="block text-xs text-amber-800 mb-1">Ende (Ist)</label>
          <TimeSelect v-model="form.actual_end_time" />
        </div>
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
import { ref, computed, watch, onMounted } from 'vue'
import { Search, Trash2 } from 'lucide-vue-next'
import dayjs from 'dayjs'
import { kitasApi, providersApi, assignmentsApi } from '../api'
import Modal from './Modal.vue'
import TimeSelect from './TimeSelect.vue'

const props = defineProps({ assignment: { type: Object, default: null } })
const emit = defineEmits(['close', 'saved'])

const kitas = ref([])
const providers = ref([])
const selectedProvider = ref('')
const kitaSearch = ref('')
const form = ref({
  kita_id: '', date: '',
  start_time: '07:00', end_time: '17:00',
  actual_start_time: '', actual_end_time: '',
  notes: '',
})

const isPastOrToday = computed(() =>
  form.value.date && form.value.date <= dayjs().format('YYYY-MM-DD')
)
const hasActual = computed(() => form.value.actual_start_time || form.value.actual_end_time)

const copyPlanToActual = () => {
  form.value.actual_start_time = form.value.start_time
  form.value.actual_end_time = form.value.end_time
}
const clearActual = () => {
  form.value.actual_start_time = ''
  form.value.actual_end_time = ''
}

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
    emit('saved')
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
      actual_end_time: a.actual_end_time || '',
      notes: a.notes || '',
    }
    const kita = kitas.value.find(k => k.id === a.kita_id)
    if (kita?.provider_id) selectedProvider.value = kita.provider_id

    // Pre-fill actual from planned when recording hours for today or a past day
    if (isPastOrToday.value && !hasActual.value) {
      copyPlanToActual()
    }
  }
})
</script>
