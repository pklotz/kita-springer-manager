<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <button @click="$emit('prev')" class="p-2 rounded-lg hover:bg-gray-200 transition-colors">
        <ChevronLeft class="w-5 h-5" />
      </button>
      <h2 class="text-xl font-semibold">{{ monthLabel }}</h2>
      <button @click="$emit('next')" class="p-2 rounded-lg hover:bg-gray-200 transition-colors">
        <ChevronRight class="w-5 h-5" />
      </button>
    </div>

    <div class="grid grid-cols-7 gap-1 mb-1">
      <div v-for="d in weekdays" :key="d" class="text-center text-xs font-medium text-gray-400 py-1">{{ d }}</div>
    </div>
    <div class="grid grid-cols-7 gap-1 mb-5">
      <div v-for="cell in cells" :key="cell.key"
        :class="['min-h-[64px] rounded-lg p-1.5 text-sm transition-colors',
          cell.isCurrentMonth ? 'bg-white shadow-sm border border-gray-100' : 'bg-gray-50/50',
          cell.isToday ? 'ring-2 ring-brand-500' : '',
          cell.holiday ? 'bg-gray-100 opacity-60' : '',
          cell.closure && !cell.holiday ? 'bg-emerald-50' : '']">
        <div :class="['text-xs mb-1 font-medium', cell.isCurrentMonth ? 'text-gray-600' : 'text-gray-300']">
          {{ cell.day }}
        </div>
        <div v-if="cell.holiday" class="text-[10px] text-gray-500 font-medium leading-tight mb-0.5 truncate uppercase tracking-tight">
          {{ cell.holiday }}
        </div>
        <div v-if="cell.closure && !cell.holiday"
          class="w-full text-[10px] bg-emerald-100 text-emerald-700 rounded px-1 py-0.5 mb-1 font-semibold truncate uppercase tracking-tight">
          {{ closureLabel(cell.closure) }}
        </div>
        <template v-for="a in cell.assignments" :key="a.id">
          <div v-if="a.status === 'free'"
            class="w-full text-left text-xs bg-blue-100 text-blue-700 rounded px-1 py-0.5 mb-0.5 font-medium truncate">
            {{ a.notes || 'Frei' }}
          </div>
          <button v-else
            @click="$emit('open-assignment', a)"
            :style="{ backgroundColor: providerColor(a), color: '#fff' }"
            class="w-full text-left text-xs rounded px-1 py-0.5 mb-0.5 truncate hover:opacity-90 transition-opacity font-medium shadow-sm">
            {{ assignmentLabel(a) }}
          </button>
        </template>
      </div>
    </div>

    <div class="flex flex-wrap gap-4 mb-8 p-4 bg-white rounded-xl border border-gray-100 shadow-sm">
      <div v-for="p in providers" :key="p.id" class="flex items-center gap-2 text-xs font-medium text-gray-600">
        <span class="w-3.5 h-3.5 rounded-full shadow-inner" :style="{ backgroundColor: p.color_hex }" />
        {{ p.name }}
      </div>
      <div class="flex items-center gap-2 text-xs font-medium text-gray-600">
        <span class="w-3.5 h-3.5 rounded-full bg-blue-100 border border-blue-200" />
        Frei / Schule
      </div>
      <div class="flex items-center gap-2 text-xs font-medium text-gray-600">
        <span class="w-3.5 h-3.5 rounded-sm bg-emerald-100 border border-emerald-200" />
        Schliesstage / Urlaub
      </div>
      <div class="flex items-center gap-2 text-xs font-medium text-gray-600">
        <span class="w-3.5 h-3.5 rounded-sm bg-gray-100 border border-gray-200" />
        Feiertag
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'
import dayjs from 'dayjs'
import isoWeek from 'dayjs/plugin/isoWeek'
import 'dayjs/locale/de'
import { closureLabel } from '../utils/closures'

dayjs.extend(isoWeek)
dayjs.locale('de')

const props = defineProps({
  month: { type: Object, required: true },
  assignments: { type: Array, default: () => [] },
  closures: { type: Array, default: () => [] },
  providers: { type: Array, default: () => [] },
})
defineEmits(['prev', 'next', 'open-assignment'])

const weekdays = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So']
const monthLabel = computed(() => props.month.format('MMMM YYYY'))

const assignmentsByDate = computed(() => {
  const map = {}
  for (const a of props.assignments) {
    if (!map[a.date]) map[a.date] = []
    map[a.date].push(a)
  }
  return map
})

const holidayByDate = computed(() => {
  const map = {}
  for (const c of props.closures) {
    if (c.type === 'holiday') map[c.date] = c.note
  }
  return map
})

const closureByDate = computed(() => {
  const map = {}
  for (const c of props.closures) {
    if (c.type !== 'holiday') map[c.date] = c
  }
  return map
})

const cells = computed(() => {
  const start = props.month.startOf('month')
  const end = props.month.endOf('month')
  const firstCell = start.startOf('isoWeek')
  const lastCell = end.endOf('isoWeek')
  const out = []
  let d = firstCell
  while (d.isBefore(lastCell) || d.isSame(lastCell, 'day')) {
    const key = d.format('YYYY-MM-DD')
    out.push({
      key, day: d.date(),
      isCurrentMonth: d.month() === props.month.month(),
      isToday: d.isSame(dayjs(), 'day'),
      assignments: assignmentsByDate.value[key] || [],
      holiday: holidayByDate.value[key] || null,
      closure: closureByDate.value[key] || null,
    })
    d = d.add(1, 'day')
  }
  return out
})

const providerColor = (a) => a.provider?.color_hex || '#6366f1'
const assignmentLabel = (a) => a.group_name || a.kita?.name || '–'
</script>
