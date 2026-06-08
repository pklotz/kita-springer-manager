<template>
  <Modal title="Abwesenheit / Schliesstag" @close="$emit('close')">
    <label class="block text-sm text-gray-600 mb-1">Typ *</label>
    <select v-model="form.type"
      class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500">
      <option value="springerin">Urlaub Springerin</option>
      <option value="provider">Schliesstag Träger</option>
      <option value="kita">Schliesstag Kita</option>
    </select>

    <div v-if="form.type === 'provider'" class="mb-3">
      <label class="block text-sm text-gray-600 mb-1">Träger *</label>
      <select v-model="form.reference_id"
        class="w-full rounded-lg border border-gray-200 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500">
        <option value="">Bitte wählen…</option>
        <option v-for="p in providers" :key="p.id" :value="p.id">{{ p.name }}</option>
      </select>
    </div>

    <div v-if="form.type === 'kita'" class="mb-3">
      <label class="block text-sm text-gray-600 mb-1">Kita *</label>
      <select v-model="form.reference_id"
        class="w-full rounded-lg border border-gray-200 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500">
        <option value="">Bitte wählen…</option>
        <option v-for="k in kitas" :key="k.id" :value="k.id">{{ k.name }}</option>
      </select>
    </div>

    <div class="flex gap-3 mb-3">
      <div class="flex-1">
        <label class="block text-sm text-gray-600 mb-1">Von *</label>
        <input type="date" v-model="form.date_from" @change="onFromChange"
          class="w-full rounded-lg border border-gray-200 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500" />
      </div>
      <div class="flex-1">
        <label class="block text-sm text-gray-600 mb-1">Bis (inkl.)</label>
        <input type="date" v-model="form.date_to" @change="toTouched = true"
          class="w-full rounded-lg border border-gray-200 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500" />
      </div>
    </div>

    <label class="block text-sm text-gray-600 mb-1">Notiz</label>
    <input v-model="form.note" type="text" placeholder="z.B. Sommerferien"
      class="w-full rounded-lg border border-gray-200 px-3 py-2 mb-4 focus:outline-none focus:ring-2 focus:ring-brand-500" />

    <div class="flex gap-3">
      <button @click="$emit('close')" class="flex-1 py-2 rounded-lg border border-gray-200 text-gray-600 hover:bg-gray-50">Abbrechen</button>
      <button @click="save" :disabled="!canSave"
        class="flex-1 py-2 rounded-lg bg-brand-500 text-white hover:bg-brand-600 disabled:opacity-50 transition-colors">
        Speichern
      </button>
    </div>
  </Modal>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { closuresApi, providersApi, kitasApi } from '../api'
import Modal from './Modal.vue'

const emit = defineEmits(['close', 'saved'])

const providers = ref([])
const kitas = ref([])

const form = ref({
  type: 'springerin',
  reference_id: '',
  date_from: '',
  date_to: '',
  note: '',
})

// Tracks whether the user has manually edited "Bis". As long as it's false we
// keep "Bis" in sync with "Von" — typical 1-day-absence case completes with a
// single keystroke. Once the user touches "Bis" we stop overriding.
const toTouched = ref(false)

const onFromChange = () => {
  if (!toTouched.value || !form.value.date_to || form.value.date_to < form.value.date_from) {
    form.value.date_to = form.value.date_from
  }
}

const canSave = computed(() => {
  if (!form.value.date_from) return false
  if (form.value.type === 'provider' && !form.value.reference_id) return false
  if (form.value.type === 'kita' && !form.value.reference_id) return false
  return true
})

const save = async () => {
  const payload = {
    type: form.value.type,
    date_from: form.value.date_from,
    date_to: form.value.date_to || form.value.date_from,
    note: form.value.note || undefined,
  }
  if (form.value.reference_id) {
    payload.reference_id = form.value.reference_id
  }

  try {
    await closuresApi.createBatch(payload)
  } catch (e) {
    if (e.response?.status === 409) {
      // Conflict: scheduled assignment overlaps — ask user for confirmation
      const msg = e.response.data?.error || 'An einem der Tage ist bereits ein Einsatz geplant — trotzdem als Abwesenheit eintragen?'
      if (!window.confirm(msg)) {
        // User cancelled — do not show an additional error toast
        return
      }
      // User confirmed — retry with force=true
      await closuresApi.createBatch({ ...payload, force: true })
    } else {
      // Non-409 errors: re-throw so the global interceptor shows a toast
      throw e
    }
  }

  emit('saved')
}

onMounted(async () => {
  [providers.value, kitas.value] = await Promise.all([providersApi.list(), kitasApi.list()])
})
</script>
