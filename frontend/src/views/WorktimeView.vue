<template>
  <div>
    <div class="flex items-center gap-3 mb-4">
      <h2 class="text-xl font-semibold">Arbeitszeit</h2>
    </div>

    <!-- Controls -->
    <div class="flex flex-wrap gap-2 mb-4">
      <select v-model="selectedMonth"
        class="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500">
        <option v-for="m in availableMonths" :key="m.key" :value="m.key">{{ m.label }}</option>
      </select>

      <select v-model="filterProvider"
        class="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500">
        <option value="">Alle Träger</option>
        <option v-for="p in providersInMonth" :key="p.id" :value="p.id">{{ p.name }}</option>
      </select>

      <label class="inline-flex items-center gap-2 text-sm text-gray-600 ml-auto cursor-pointer">
        <input type="checkbox" v-model="groupByProvider" class="rounded">
        Nach Träger gruppieren
      </label>

      <button v-if="filteredItems.length" @click="downloadPDF" :disabled="downloading"
        class="inline-flex items-center gap-1 text-sm bg-brand-500 text-white px-3 py-2 rounded-lg hover:bg-brand-600 disabled:opacity-60 transition-colors">
        {{ downloading ? '…' : 'PDF' }}
      </button>
    </div>

    <!-- Empty state -->
    <div v-if="!availableMonths.length" class="text-center text-gray-400 py-16">
      Noch keine erfassten Einsätze
    </div>
    <div v-else-if="!filteredItems.length" class="text-center text-gray-400 py-12">
      Keine Einsätze in diesem Monat{{ filterProvider ? ' für diesen Träger' : '' }}
    </div>

    <template v-if="filteredItems.length">
      <!-- Chart: Stunden pro Tag -->
      <WorktimeChart :items="filteredItems" :month="selectedMonth" />

      <!-- Flat table (ungrouped) -->
      <div v-if="!groupByProvider" class="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
        <WorktimeTable :rows="filteredItems" :totals="totals(filteredItems)" />
      </div>

      <!-- Grouped by provider -->
      <div v-else>
        <div v-for="g in groupedItems" :key="g.providerId" class="mb-4">
          <div class="flex items-center gap-2 mb-2 px-1">
            <span v-if="g.providerColor" class="w-3 h-3 rounded-full"
              :style="{ backgroundColor: g.providerColor }" />
            <h3 class="font-semibold text-gray-700">{{ g.providerName || '– ohne Träger –' }}</h3>
          </div>
          <div class="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
            <WorktimeTable :rows="g.items" :totals="totals(g.items)" />
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import dayjs from 'dayjs'
import 'dayjs/locale/de'
import { assignmentsApi, providersApi, kitasApi, downloadsApi } from '../api'
import {
  netWorkMinutes, breakMinutes, grossWorkMinutes, requiredBreakMinutes,
} from '../utils/time'
import WorktimeTable from '../components/WorktimeTable.vue'
import WorktimeChart from '../components/WorktimeChart.vue'

dayjs.locale('de')

const route = useRoute()
const router = useRouter()
const assignments = ref([])
const providers = ref([])
const kitas = ref([])
const selectedMonth = ref(route.query.month || dayjs().format('YYYY-MM'))
const filterProvider = ref('')
const groupByProvider = ref(false)

// Assignment.provider_id is sometimes empty (manual entries); fall back to the
// Kita's provider so grouping/filtering still works.
const effectiveProvider = (a) => {
  let pid = a.provider_id
  if (!pid) {
    const kita = kitas.value.find(k => k.id === a.kita_id)
    pid = kita?.provider_id || ''
  }
  if (!pid) return null
  return providers.value.find(p => p.id === pid) || null
}

// Keep URL ?month= synced so Historie → Arbeitszeit links preserve context
watch(selectedMonth, (m) => {
  router.replace({ query: { ...route.query, month: m } })
})

const today = dayjs().format('YYYY-MM-DD')

// Augment each assignment with its effective provider so downstream logic
// (filter, group, break validation) works uniformly.
const recorded = computed(() =>
  assignments.value
    .filter(a =>
      a.status !== 'free' &&
      a.date <= today &&
      (a.actual_start_time || a.actual_end_time)
    )
    .map(a => ({ ...a, provider: effectiveProvider(a) || a.provider || null })),
)

const availableMonths = computed(() => {
  const keys = new Set(recorded.value.map(a => a.date.slice(0, 7)))
  return Array.from(keys).sort().reverse().map(key => ({
    key,
    label: dayjs(key + '-01').format('MMMM YYYY'),
  }))
})

const itemsOfMonth = computed(() =>
  recorded.value
    .filter(a => a.date.slice(0, 7) === selectedMonth.value)
    .sort((x, y) => x.date.localeCompare(y.date))
)

const providersInMonth = computed(() => {
  const ids = new Set(itemsOfMonth.value.map(a => a.provider?.id).filter(Boolean))
  return providers.value.filter(p => ids.has(p.id))
})

const filteredItems = computed(() => {
  if (!filterProvider.value) return itemsOfMonth.value
  return itemsOfMonth.value.filter(a => a.provider?.id === filterProvider.value)
})

const downloading = ref(false)
const downloadPDF = async () => {
  if (downloading.value) return
  downloading.value = true
  try {
    await downloadsApi.worktimePDF(selectedMonth.value, filterProvider.value || null)
  } finally {
    downloading.value = false
  }
}

const groupedItems = computed(() => {
  const groups = new Map()
  for (const a of filteredItems.value) {
    const pid = a.provider?.id || ''
    if (!groups.has(pid)) {
      groups.set(pid, {
        providerId: pid,
        providerName: a.provider?.name || '',
        providerColor: a.provider?.color_hex || '',
        items: [],
      })
    }
    groups.get(pid).items.push(a)
  }
  return Array.from(groups.values()).sort((a, b) =>
    (a.providerName || 'zz').localeCompare(b.providerName || 'zz'),
  )
})

const totals = (rows) => {
  let net = 0
  let brk = 0
  let breaches = 0
  for (const a of rows) {
    net += netWorkMinutes(a.actual_start_time, a.actual_break_start, a.actual_break_end, a.actual_end_time)
    brk += breakMinutes(a.actual_break_start, a.actual_break_end)
    const req = requiredBreakMinutes(
      grossWorkMinutes(a.actual_start_time, a.actual_end_time),
      a.provider?.min_break_minutes || 0,
    )
    if (req > 0 && breakMinutes(a.actual_break_start, a.actual_break_end) < req) breaches++
  }
  return { count: rows.length, netMin: net, breakMin: brk, breaches }
}

onMounted(async () => {
  [assignments.value, providers.value, kitas.value] = await Promise.all([
    assignmentsApi.list(
      dayjs().subtract(2, 'year').format('YYYY-MM-DD'),
      today,
    ),
    providersApi.list(),
    kitasApi.list(),
  ])
  if (!availableMonths.value.find(m => m.key === selectedMonth.value) && availableMonths.value.length) {
    selectedMonth.value = availableMonths.value[0].key
  }
})
</script>
