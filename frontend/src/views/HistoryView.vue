<template>
  <div>
    <div class="flex items-center gap-3 mb-6">
      <button @click="$router.back()" class="p-2 rounded-lg hover:bg-gray-200 transition-colors">
        <ArrowLeft class="w-5 h-5" />
      </button>
      <h2 class="text-xl font-semibold">Historie</h2>
    </div>

    <div v-if="pastAssignments.length" class="bg-white rounded-xl border border-gray-100 shadow-sm p-3 mb-3 space-y-2">
      <div class="flex gap-2 flex-wrap">
        <select v-model="providerFilter"
          class="flex-1 min-w-[10ch] rounded-lg border border-gray-200 px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500">
          <option value="">Alle Träger</option>
          <option v-for="p in providersInHistory" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
        <select v-model="kitaFilter"
          class="flex-1 min-w-[10ch] rounded-lg border border-gray-200 px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500">
          <option value="">Alle Kitas</option>
          <option v-for="k in kitasInHistory" :key="k.id" :value="k.id">{{ k.name }}</option>
        </select>
      </div>
      <div class="relative">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
        <input v-model="query" type="text" placeholder="In Notizen, Gruppe, Kita suchen…"
          class="w-full pl-9 pr-3 py-1.5 rounded-lg border border-gray-200 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500" />
      </div>
    </div>

    <div v-if="months.length === 0" class="text-center text-gray-400 py-16">
      {{ pastAssignments.length === 0 ? 'Keine vergangenen Einsätze' : 'Keine Treffer für diese Filter' }}
    </div>

    <div v-for="month in months" :key="month.key" class="mb-6">
      <div class="flex items-baseline justify-between mb-2 px-1 gap-2">
        <h3 class="font-semibold text-gray-700">{{ month.label }}</h3>
        <div class="flex items-baseline gap-3 text-xs text-gray-500">
          <span class="font-medium">{{ month.count }} Einsätze</span>
          <span v-if="month.totalHours">{{ month.totalHours }} h netto</span>
          <span v-if="month.plannedHours && month.plannedHours !== month.totalHours"
            class="text-gray-400">(Soll {{ month.plannedHours }})</span>
          <RouterLink :to="`/worktime?month=${month.key}`"
            class="text-brand-500 hover:text-brand-600 flex items-center gap-0.5">
            Arbeitszeit <ArrowRight class="w-3 h-3" />
          </RouterLink>
        </div>
      </div>

      <div v-for="a in month.items" :key="a.id"
        class="bg-white rounded-xl shadow-sm border border-gray-100 p-3 mb-2 flex items-center gap-3 transition-colors"
        :class="hasActual(a) ? 'hover:border-gray-200' : 'border-amber-200 hover:border-amber-300'">
        <div class="shrink-0 w-12 text-center cursor-pointer"
          @click="$router.push(`/assignments/${a.id}`)">
          <div class="text-lg font-bold text-gray-700 leading-none">{{ day(a.date) }}</div>
          <div class="text-[10px] text-gray-400 uppercase">{{ weekday(a.date) }}</div>
        </div>
        <div class="flex-1 min-w-0 cursor-pointer"
          @click="$router.push(`/assignments/${a.id}`)">
          <div class="font-medium text-gray-800 truncate">
            {{ a.kita?.name || a.group_name || '–' }}
            <span v-if="a.provider?.name" class="text-xs px-1.5 py-0.5 rounded-full text-white ml-1"
              :style="{ backgroundColor: a.provider.color_hex }">{{ a.provider.name }}</span>
          </div>
          <div class="text-sm text-gray-500 flex items-center gap-2 flex-wrap">
            <span>Soll {{ a.start_time || '–' }}–{{ a.end_time || '–' }}</span>
            <span v-if="hasActual(a)"
              :class="differs(a) ? 'text-amber-600' : 'text-emerald-600'">
              · Ist {{ a.actual_start_time || '–' }}–{{ a.actual_end_time || '–' }}
            </span>
            <span v-else class="text-gray-400 italic">· Arbeitszeit fehlt</span>
            <span v-if="hasActual(a)" class="text-gray-400">
              · Netto {{ netHours(a) }} h
              <span v-if="breakMin(a) > 0"
                :class="breakWarn(a) ? 'text-red-600 font-medium' : ''"
                :title="breakWarn(a) ? 'Pause unter Mindestmass' : ''">
                · Pause {{ breakHm(a) }}
              </span>
            </span>
          </div>
        </div>
        <button @click="editAssignment = a"
          :title="hasActual(a) ? 'Arbeitszeit bearbeiten' : 'Arbeitszeit erfassen'"
          :class="['shrink-0 p-2 rounded-lg transition-colors',
            hasActual(a)
              ? 'text-gray-400 hover:bg-gray-100 hover:text-gray-600'
              : 'text-amber-600 bg-amber-50 hover:bg-amber-100']">
          <Clock class="w-4 h-4" />
        </button>
      </div>
    </div>

    <AssignmentForm v-if="editAssignment" :assignment="editAssignment"
      @close="editAssignment = null" @saved="onSaved" @deleted="onSaved" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { ArrowLeft, ArrowRight, Clock, Search } from 'lucide-vue-next'
import dayjs from 'dayjs'
import 'dayjs/locale/de'
import { assignmentsApi } from '../api'
import {
  diffMinutes, formatHours, formatHm, netWorkMinutes, breakMinutes,
  grossWorkMinutes, requiredBreakMinutes,
} from '../utils/time'
import AssignmentForm from '../components/AssignmentForm.vue'

dayjs.locale('de')

const assignments = ref([])
const editAssignment = ref(null)
const today = dayjs().format('YYYY-MM-DD')
const providerFilter = ref('')
const kitaFilter = ref('')
const query = ref('')

const day = (d) => dayjs(d).format('D')
const weekday = (d) => dayjs(d).format('dd')
const hasActual = (a) => a.actual_start_time || a.actual_end_time
const netMin = (a) => netWorkMinutes(a.actual_start_time, a.actual_break_start, a.actual_break_end, a.actual_end_time)
const netHours = (a) => formatHours(netMin(a))
const breakMin = (a) => breakMinutes(a.actual_break_start, a.actual_break_end)
const breakHm = (a) => formatHm(breakMin(a))
const breakWarn = (a) => {
  const req = requiredBreakMinutes(
    grossWorkMinutes(a.actual_start_time, a.actual_end_time),
    a.provider?.min_break_minutes || 0,
  )
  return req > 0 && breakMin(a) < req
}
const differs = (a) =>
  hasActual(a) && netMin(a) !== diffMinutes(a.start_time, a.end_time)

const pastAssignments = computed(() =>
  assignments.value
    .filter(a => a.date < today && a.status !== 'free')
    .sort((x, y) => y.date.localeCompare(x.date))
)

const providersInHistory = computed(() => {
  const seen = new Map()
  for (const a of pastAssignments.value) {
    if (a.provider?.name && a.provider_id && !seen.has(a.provider_id)) {
      seen.set(a.provider_id, { id: a.provider_id, name: a.provider.name })
    }
  }
  return [...seen.values()].sort((a, b) => a.name.localeCompare(b.name))
})

const kitasInHistory = computed(() => {
  const seen = new Map()
  for (const a of pastAssignments.value) {
    if (a.kita?.name && a.kita_id && !seen.has(a.kita_id)) {
      seen.set(a.kita_id, { id: a.kita_id, name: a.kita.name })
    }
  }
  return [...seen.values()].sort((a, b) => a.name.localeCompare(b.name))
})

const months = computed(() => {
  const q = query.value.trim().toLowerCase()
  const past = pastAssignments.value.filter(a => {
    if (providerFilter.value && a.provider_id !== providerFilter.value) return false
    if (kitaFilter.value && a.kita_id !== kitaFilter.value) return false
    if (q) {
      const hay = [a.kita?.name, a.group_name, a.notes].filter(Boolean).join(' ').toLowerCase()
      if (!hay.includes(q)) return false
    }
    return true
  })

  const groups = {}
  for (const a of past) {
    const key = a.date.slice(0, 7)
    if (!groups[key]) groups[key] = []
    groups[key].push(a)
  }

  return Object.keys(groups).sort().reverse().map(key => {
    const items = groups[key]
    const totalMin = items.reduce((s, a) => {
      if (hasActual(a)) return s + netMin(a)
      return s + diffMinutes(a.start_time, a.end_time)
    }, 0)
    const plannedMin = items.reduce(
      (s, a) => s + diffMinutes(a.start_time, a.end_time), 0,
    )
    return {
      key,
      label: dayjs(key + '-01').format('MMMM YYYY'),
      items,
      count: items.length,
      totalHours: formatHours(totalMin),
      plannedHours: formatHours(plannedMin),
    }
  })
})

const load = async () => {
  assignments.value = await assignmentsApi.list(
    dayjs().subtract(2, 'year').format('YYYY-MM-DD'),
    today,
  )
}

const onSaved = () => {
  editAssignment.value = null
  load()
}

onMounted(load)
</script>
