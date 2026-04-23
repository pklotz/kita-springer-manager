<template>
  <table class="w-full text-sm">
    <thead class="bg-gray-50 text-gray-500 text-xs uppercase">
      <tr>
        <th class="text-left px-3 py-2 font-medium">Datum</th>
        <th class="text-left px-3 py-2 font-medium">Kita</th>
        <th class="text-left px-3 py-2 font-medium whitespace-nowrap">Vormittag</th>
        <th class="text-left px-3 py-2 font-medium whitespace-nowrap">Nachmittag</th>
        <th class="text-right px-3 py-2 font-medium whitespace-nowrap">Pause</th>
        <th class="text-right px-3 py-2 font-medium whitespace-nowrap">Arbeitszeit</th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="a in rows" :key="a.id"
        class="border-t border-gray-100 hover:bg-gray-50 cursor-pointer"
        @click="$router.push(`/assignments/${a.id}`)">
        <td class="px-3 py-2 whitespace-nowrap">
          <div class="font-medium text-gray-800">{{ dayLabel(a.date) }}</div>
          <div class="text-[10px] text-gray-400 uppercase">{{ weekday(a.date) }}</div>
        </td>
        <td class="px-3 py-2 text-gray-700">
          <div class="font-medium truncate max-w-[10rem]">{{ a.kita?.name || '–' }}</div>
        </td>
        <td class="px-3 py-2 text-gray-600 whitespace-nowrap">{{ morningRange(a) || '–' }}</td>
        <td class="px-3 py-2 text-gray-600 whitespace-nowrap">{{ afternoonRange(a) || '–' }}</td>
        <td class="px-3 py-2 text-right whitespace-nowrap"
          :class="breakWarn(a) ? 'text-red-600 font-medium' : 'text-gray-600'"
          :title="breakWarn(a) ? breakWarnTitle(a) : ''">
          {{ breakLabel(a) }}
        </td>
        <td class="px-3 py-2 text-right font-medium text-gray-800 whitespace-nowrap">
          {{ formatHm(netMin(a)) }}
        </td>
      </tr>
    </tbody>
    <tfoot class="bg-gray-50 text-gray-700 border-t-2 border-gray-200">
      <tr>
        <td class="px-3 py-2 font-medium" colspan="2">{{ totals.count }} Einsätze</td>
        <td class="px-3 py-2" colspan="2">
          <span v-if="totals.breaches" class="text-xs text-red-600">
            {{ totals.breaches }} × Pause zu kurz
          </span>
        </td>
        <td class="px-3 py-2 text-right font-medium whitespace-nowrap">{{ formatHm(totals.breakMin) }}</td>
        <td class="px-3 py-2 text-right font-semibold whitespace-nowrap">{{ formatHm(totals.netMin) }}</td>
      </tr>
    </tfoot>
  </table>
</template>

<script setup>
import dayjs from 'dayjs'
import 'dayjs/locale/de'
import {
  netWorkMinutes, breakMinutes, grossWorkMinutes, requiredBreakMinutes,
  legalMinBreakMinutes, formatHm,
} from '../utils/time'

dayjs.locale('de')

defineProps({
  rows: { type: Array, required: true },
  totals: { type: Object, required: true },
})

const dayLabel = (d) => dayjs(d).format('D.M.')
const weekday = (d) => dayjs(d).format('dd')

const morningRange = (a) => {
  if (!a.actual_start_time) return ''
  const end = a.actual_break_start || (a.actual_break_end ? '' : a.actual_end_time)
  return end ? `${a.actual_start_time}–${end}` : a.actual_start_time
}
const afternoonRange = (a) => {
  if (!a.actual_break_end) return ''
  return `${a.actual_break_end}–${a.actual_end_time || '–'}`
}
const netMin = (a) => netWorkMinutes(a.actual_start_time, a.actual_break_start, a.actual_break_end, a.actual_end_time)
const breakMin = (a) => breakMinutes(a.actual_break_start, a.actual_break_end)
const breakLabel = (a) => breakMin(a) > 0 ? formatHm(breakMin(a)) : '–'
const breakWarn = (a) => {
  const req = requiredBreakMinutes(
    grossWorkMinutes(a.actual_start_time, a.actual_end_time),
    a.provider?.min_break_minutes || 0,
  )
  return req > 0 && breakMin(a) < req
}
const breakWarnTitle = (a) => {
  const gross = grossWorkMinutes(a.actual_start_time, a.actual_end_time)
  const legal = legalMinBreakMinutes(gross)
  const prov = a.provider?.min_break_minutes || 0
  const parts = []
  if (legal > 0) parts.push(`${legal} min laut ArG Art. 15`)
  if (prov > 0) parts.push(`${prov} min Trägervorgabe`)
  return `Mindestpause: ${parts.join(', ')}`
}
</script>
