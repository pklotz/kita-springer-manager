<template>
  <div>
    <!-- Filter bar -->
    <div v-if="assignments.length" class="bg-white rounded-xl border border-gray-100 shadow-sm p-3 mb-3 space-y-2">
      <div class="flex gap-2 flex-wrap">
        <select v-model="providerFilter"
          class="flex-1 min-w-[10ch] rounded-lg border border-gray-200 px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500">
          <option value="">Alle Träger</option>
          <option v-for="p in providers" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
        <select v-model="kitaFilter"
          class="flex-1 min-w-[10ch] rounded-lg border border-gray-200 px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500">
          <option value="">Alle Kitas</option>
          <option v-for="k in kitasInList" :key="k.id" :value="k.id">{{ k.name }}</option>
        </select>
      </div>
      <div class="relative">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
        <input v-model="query" type="text" placeholder="In Notizen, Gruppe, Kita suchen…"
          class="w-full pl-9 pr-3 py-1.5 rounded-lg border border-gray-200 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500" />
      </div>
    </div>

    <!-- Bulk action bar -->
    <div v-if="selection.size > 0"
      class="flex items-center justify-between bg-red-50 border border-red-100 rounded-xl px-4 py-2 mb-3">
      <span class="text-sm text-red-700 font-medium">
        {{ selection.size }} ausgewählt
      </span>
      <div class="flex gap-2">
        <button @click="clearSelection"
          class="text-sm text-gray-600 hover:text-gray-800 underline">
          Aufheben
        </button>
        <button @click="bulkDelete"
          class="text-sm bg-red-500 text-white px-3 py-1.5 rounded-lg hover:bg-red-600 transition-colors">
          Löschen
        </button>
      </div>
    </div>

    <div v-if="filtered.length === 0" class="text-center text-gray-400 py-8">
      {{ assignments.length === 0 ? 'Keine bevorstehenden Einsätze' : 'Keine Treffer für diese Filter' }}
    </div>

    <div v-for="a in filtered" :key="a.id"
      :class="['rounded-xl border p-4 mb-3 transition-shadow flex items-start gap-3',
        a.status === 'free'
          ? 'bg-blue-50 border-blue-200'
          : selection.has(a.id)
            ? 'bg-white border-brand-300 shadow-sm ring-1 ring-brand-200'
            : 'bg-white border-gray-100 shadow-sm']">
      <input v-if="a.status !== 'free'" type="checkbox"
        :checked="selection.has(a.id)" @change="toggle(a.id)"
        class="mt-1 shrink-0 w-4 h-4 rounded border-gray-300 text-brand-500 focus:ring-brand-500" />

      <div class="flex-1 min-w-0 cursor-pointer"
        @click="a.status !== 'free' && $emit('open-detail', a)">
        <div class="flex items-center gap-2 flex-wrap">
          <span v-if="a.status === 'free'" class="font-semibold text-blue-700">
            {{ a.notes || 'Freier Tag' }}
          </span>
          <span v-else class="font-semibold text-gray-800">
            {{ a.kita?.name || a.group_name || '–' }}
            <span v-if="a.group_name && a.kita?.name" class="font-normal text-gray-500 text-sm">({{ a.group_name }})</span>
          </span>
          <span v-if="a.provider?.name" class="text-xs px-1.5 py-0.5 rounded-full text-white"
            :style="{ backgroundColor: a.provider?.color_hex }">
            {{ a.provider.name }}
          </span>
        </div>
        <div class="text-sm text-gray-500 mt-0.5">{{ formatDate(a.date) }}</div>
        <div v-if="a.start_time && a.status !== 'free'" class="text-sm text-gray-600 mt-0.5">
          {{ a.start_time }}<span v-if="a.end_time"> – {{ a.end_time }}</span>
        </div>
      </div>
      <button v-if="a.status !== 'free'" @click="$emit('edit', a)"
        class="p-1.5 rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors shrink-0">
        <Pencil class="w-4 h-4" />
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { Pencil, Search } from 'lucide-vue-next'
import dayjs from 'dayjs'
import 'dayjs/locale/de'

dayjs.locale('de')

const props = defineProps({
  assignments: { type: Array, default: () => [] },
  providers: { type: Array, default: () => [] },
})
const emit = defineEmits(['edit', 'open-detail', 'bulk-delete'])

const providerFilter = ref('')
const kitaFilter = ref('')
const query = ref('')
const selection = ref(new Set())

const kitasInList = computed(() => {
  const seen = new Map()
  for (const a of props.assignments) {
    if (a.kita?.name && a.kita_id && !seen.has(a.kita_id)) {
      seen.set(a.kita_id, { id: a.kita_id, name: a.kita.name })
    }
  }
  return [...seen.values()].sort((a, b) => a.name.localeCompare(b.name))
})

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  return props.assignments.filter(a => {
    if (providerFilter.value && a.provider_id !== providerFilter.value) return false
    if (kitaFilter.value && a.kita_id !== kitaFilter.value) return false
    if (q) {
      const hay = [a.kita?.name, a.group_name, a.notes].filter(Boolean).join(' ').toLowerCase()
      if (!hay.includes(q)) return false
    }
    return true
  })
})

const toggle = (id) => {
  const next = new Set(selection.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selection.value = next
}

const clearSelection = () => { selection.value = new Set() }

const bulkDelete = () => {
  const ids = [...selection.value]
  if (!ids.length) return
  if (!confirm(`${ids.length} Einsätze wirklich löschen?`)) return
  emit('bulk-delete', ids)
  selection.value = new Set()
}

const formatDate = (d) => dayjs(d).format('dddd, D. MMMM YYYY')
</script>
