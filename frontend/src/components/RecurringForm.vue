<template>
  <Modal title="Fixe / Wiederkehrende Einsätze" @close="$emit('close')">
    <!-- Existing rules for selected provider -->
    <div v-if="rulesForProvider.length" class="mb-4 space-y-2">
      <div v-for="r in rulesForProvider" :key="r.id"
        class="flex items-center justify-between bg-gray-50 rounded-lg px-3 py-2 text-sm">
        <div class="flex-1 min-w-0">
          <span class="font-medium">{{ weekdayName(r.day_of_week) }}</span>
          <span class="text-gray-500 ml-2 truncate">{{ r.kita?.name || r.group_name || '–' }}</span>
          <span v-if="r.start_time" class="text-gray-400 ml-2">{{ r.start_time }}–{{ r.end_time }}</span>
        </div>
        <div class="text-xs text-gray-400 mx-2 shrink-0">{{ r.valid_from }} – {{ r.valid_until }}</div>
        <button @click="deleteRule(r.id)" class="text-gray-400 hover:text-red-500 transition-colors shrink-0">
          <Trash2 class="w-4 h-4" />
        </button>
      </div>
    </div>
    <p v-else-if="activeProvider" class="text-sm text-gray-400 mb-4">Noch keine fixen Einsätze für diesen Träger</p>

    <div :class="rulesForProvider.length ? 'border-t pt-4' : ''">
      <h4 class="text-sm font-semibold text-gray-700 mb-3">Neuer fixer Einsatz</h4>

      <div v-if="!provider">
        <label class="block text-sm text-gray-600 mb-1">Träger *</label>
        <select v-model="selectedProviderId"
          class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500">
          <option value="">Bitte wählen…</option>
          <option v-for="p in providers" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
      </div>

      <label class="block text-sm text-gray-600 mb-1">Wochentag *</label>
      <select v-model.number="form.day_of_week"
        class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500">
        <option v-for="(d, i) in weekdays" :key="i" :value="i">{{ d }}</option>
      </select>

      <label class="block text-sm text-gray-600 mb-1">Kita</label>
      <select v-model="form.kita_id"
        class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500">
        <option value="">– keine Kita –</option>
        <option v-for="k in filteredKitas" :key="k.id" :value="k.id">{{ k.name }}</option>
      </select>

      <label class="block text-sm text-gray-600 mb-1">Gruppe / Bezeichnung</label>
      <input v-model="form.group_name" type="text" placeholder="z.B. Gruppe Blau"
        class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500" />

      <div class="flex gap-3 mb-3">
        <div class="flex-1">
          <label class="block text-sm text-gray-600 mb-1">Beginn</label>
          <TimeSelect v-model="form.start_time" />
        </div>
        <div class="flex-1">
          <label class="block text-sm text-gray-600 mb-1">Ende</label>
          <TimeSelect v-model="form.end_time" />
        </div>
      </div>

      <div class="flex gap-3 mb-4">
        <div class="flex-1">
          <label class="block text-sm text-gray-600 mb-1">Gültig ab *</label>
          <input type="date" v-model="form.valid_from"
            class="w-full rounded-lg border border-gray-200 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500" />
        </div>
        <div class="flex-1">
          <label class="block text-sm text-gray-600 mb-1">Gültig bis *</label>
          <input type="date" v-model="form.valid_until"
            class="w-full rounded-lg border border-gray-200 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500" />
        </div>
      </div>

      <div v-if="saveResult" class="mb-3 text-sm text-green-700 bg-green-50 rounded-lg px-3 py-2">
        {{ saveResult.created }} Einsätze generiert · {{ saveResult.skipped }} übersprungen (Feiertage / bereits vorhanden)
      </div>

      <div class="flex gap-3">
        <button @click="$emit('close')" class="flex-1 py-2 rounded-lg border border-gray-200 text-gray-600 hover:bg-gray-50">Schliessen</button>
        <button @click="save" :disabled="!canSave"
          class="flex-1 py-2 rounded-lg bg-brand-500 text-white hover:bg-brand-600 disabled:opacity-50 transition-colors">
          Generieren
        </button>
      </div>
    </div>
  </Modal>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { Trash2 } from 'lucide-vue-next'
import { recurringApi, kitasApi, providersApi } from '../api'
import Modal from './Modal.vue'
import TimeSelect from './TimeSelect.vue'

const props = defineProps({ provider: { type: Object, default: null } })
const emit = defineEmits(['close', 'saved'])

const rules = ref([])
const kitas = ref([])
const providers = ref([])
const selectedProviderId = ref(props.provider?.id || '')
const saveResult = ref(null)

const weekdays = ['Montag', 'Dienstag', 'Mittwoch', 'Donnerstag', 'Freitag', 'Samstag', 'Sonntag']
const weekdayName = (i) => weekdays[i] ?? '–'

const form = ref({
  day_of_week: 4,
  kita_id: '',
  group_name: '',
  start_time: '',
  end_time: '',
  valid_from: '',
  valid_until: '',
})

const activeProvider = computed(() =>
  props.provider ?? providers.value.find(p => p.id === selectedProviderId.value) ?? null
)

const rulesForProvider = computed(() =>
  selectedProviderId.value
    ? rules.value.filter(r => r.provider_id === selectedProviderId.value)
    : []
)

const filteredKitas = computed(() =>
  selectedProviderId.value
    ? kitas.value.filter(k => k.provider_id === selectedProviderId.value)
    : kitas.value
)

watch(selectedProviderId, () => { form.value.kita_id = '' })

const canSave = computed(() =>
  form.value.valid_from && form.value.valid_until && (props.provider?.id || selectedProviderId.value)
)

const save = async () => {
  const providerId = props.provider?.id || selectedProviderId.value
  const result = await recurringApi.create({ ...form.value, provider_id: providerId })
  saveResult.value = result
  loadRules()
  emit('saved')
  setTimeout(() => { saveResult.value = null }, 6000)
}

const deleteRule = async (id) => {
  if (confirm('Fixe Einsatz-Regel löschen? Bereits erstellte Einträge bleiben erhalten.')) {
    await recurringApi.delete(id)
    loadRules()
  }
}

const loadRules = async () => { rules.value = await recurringApi.list() }

onMounted(async () => {
  if (props.provider) selectedProviderId.value = props.provider.id
  await Promise.all([
    loadRules(),
    kitasApi.list().then(k => { kitas.value = k }),
    providersApi.list().then(p => { providers.value = p }),
  ])
})
</script>
