<template>
  <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-4 mb-4">
    <h3 class="text-sm font-semibold text-gray-700 mb-3">Stunden pro Tag</h3>

    <div ref="container" class="relative">
      <!-- Chart with Y-axis -->
      <div class="flex">
        <!-- Y-axis labels -->
        <div class="relative shrink-0 pr-1.5 text-[9px] text-gray-400 tabular-nums text-right"
          :style="{ height: chartHeight + 'px', width: '24px' }">
          <span v-for="t in yTicks" :key="t.label"
            class="absolute right-1.5 -translate-y-1/2 leading-none"
            :style="{ top: t.topPct + '%' }">
            {{ t.label }}
          </span>
        </div>

        <!-- Chart area with horizontal grid lines -->
        <div class="flex-1 relative">
          <!-- Grid lines aligned with y-ticks -->
          <div v-for="t in yTicks" :key="t.label"
            class="absolute inset-x-0 border-t border-dashed border-gray-100"
            :style="{ top: t.topPct + '%' }" />
          <div class="relative flex items-stretch border-l border-b border-gray-200"
            :style="{ height: chartHeight + 'px' }">
            <div v-for="(d, i) in days" :key="d.date"
              class="relative flex-1 cursor-default border-r border-gray-100 last:border-r-0"
              :class="{ 'bg-gray-100/70': d.isWeekend }"
              @mouseenter="onEnter(i)"
              @mouseleave="onLeave"
              @click.stop="onClick(i)">
              <div class="absolute inset-x-[2px] bottom-0 flex flex-col-reverse"
                :style="{ height: barTotalPct(d) + '%' }">
                <div v-for="seg in d.segments" :key="seg.providerId"
                  class="w-full"
                  :style="{ flex: seg.netMin, backgroundColor: seg.color }" />
              </div>
            </div>
          </div>

          <!-- Day labels -->
          <div class="flex pt-1">
            <div v-for="d in days" :key="d.date"
              class="flex-1 text-center text-[9px] tabular-nums"
              :class="d.isWeekend ? 'text-gray-300' : 'text-gray-500'">
              {{ d.day }}
            </div>
          </div>
        </div>
      </div>

      <!-- Tooltip -->
      <div v-if="active && active.totalNetMin > 0"
        class="absolute z-20 -translate-x-1/2 -translate-y-full pointer-events-none"
        :style="{ left: tooltipLeft + 'px', top: '-6px' }">
        <div class="bg-white border border-gray-200 shadow-lg rounded-lg p-3 text-xs min-w-[200px] max-w-[280px]">
          <div class="font-semibold text-gray-800 mb-2">{{ active.label }}</div>
          <div v-for="seg in active.segments" :key="seg.providerId" class="mb-2 last:mb-0">
            <div class="flex items-center gap-1.5 mb-0.5">
              <span class="w-2.5 h-2.5 rounded-sm" :style="{ backgroundColor: seg.color }" />
              <span class="font-medium text-gray-700 truncate">{{ seg.providerName || '–' }}</span>
              <span class="ml-auto text-gray-600 font-medium tabular-nums">{{ formatStd(seg.netMin) }}</span>
            </div>
            <ul class="ml-4 text-gray-500 space-y-0.5">
              <li v-for="(k, idx) in seg.kitas" :key="idx" class="flex justify-between gap-2">
                <span class="truncate">{{ k.name }}<span v-if="k.group" class="text-gray-400"> ({{ k.group }})</span></span>
                <span class="tabular-nums shrink-0">{{ formatStd(k.netMin) }}</span>
              </li>
            </ul>
          </div>
          <div class="border-t border-gray-100 mt-2 pt-1.5 flex justify-between font-semibold text-gray-800">
            <span>Total</span>
            <span class="tabular-nums">{{ formatStd(active.totalNetMin) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import dayjs from 'dayjs'
import { netWorkMinutes } from '../utils/time'

const props = defineProps({
  items: { type: Array, default: () => [] },     // assignments of the selected month
  month: { type: String, required: true },       // YYYY-MM
})

const chartHeight = 110
const container = ref(null)
const hoveredIdx = ref(null)
const stickyIdx = ref(null)
const tooltipLeft = ref(0)

const activeIdx = computed(() => stickyIdx.value ?? hoveredIdx.value)

const days = computed(() => {
  const start = dayjs(props.month + '-01')
  const daysInMonth = start.daysInMonth()
  const out = []
  for (let i = 0; i < daysInMonth; i++) {
    const d = start.add(i, 'day')
    const iso = d.format('YYYY-MM-DD')
    const wd = d.day() // 0=Sun..6=Sat
    out.push({
      date: iso,
      day: d.date(),
      label: d.format('dd, D. MMMM'),
      isWeekend: wd === 0 || wd === 6,
      segments: [],
      totalNetMin: 0,
    })
  }
  // Group items by date → provider; collect kitas per provider segment.
  const byDate = new Map(out.map(d => [d.date, d]))
  for (const a of props.items) {
    const day = byDate.get(a.date)
    if (!day) continue
    const net = netWorkMinutes(a.actual_start_time, a.actual_break_start, a.actual_break_end, a.actual_end_time)
    if (net <= 0) continue
    const pid = a.provider?.id || ''
    let seg = day.segments.find(s => s.providerId === pid)
    if (!seg) {
      seg = {
        providerId: pid,
        providerName: a.provider?.name || '–',
        color: a.provider?.color_hex || '#9ca3af',
        netMin: 0,
        kitas: [],
      }
      day.segments.push(seg)
    }
    seg.netMin += net
    seg.kitas.push({
      name: a.kita?.name || '–',
      group: a.group_name || '',
      netMin: net,
    })
    day.totalNetMin += net
  }
  // Sort segments by provider name (deterministic stack order).
  for (const d of out) {
    d.segments.sort((a, b) => a.providerName.localeCompare(b.providerName))
  }
  return out
})

const maxNetMin = computed(() => Math.max(0, ...days.value.map(d => d.totalNetMin)))

// Round max up to a "nice" hour boundary so the y-axis has neat tick values.
const chartMaxMin = computed(() => {
  const m = maxNetMin.value
  if (m <= 0) return 60
  const h = m / 60
  const niceSteps = [1, 2, 3, 4, 5, 6, 8, 10, 12, 16, 20, 24]
  for (const s of niceSteps) {
    if (h <= s) return s * 60
  }
  return Math.ceil(h) * 60
})

// Y-axis ticks: 0 at bottom, chartMax at top, evenly spaced between.
const yTicks = computed(() => {
  const max = chartMaxMin.value
  if (max <= 0) return []
  const h = max / 60
  let step
  if (h <= 4) step = 1
  else if (h <= 8) step = 2
  else if (h <= 12) step = 3
  else step = 4
  const ticks = []
  for (let v = 0; v <= h + 0.0001; v += step) {
    ticks.push({
      label: v === 0 ? '0' : `${v} h`,
      topPct: ((h - v) / h) * 100,
    })
  }
  return ticks
})

const barTotalPct = (d) => {
  if (chartMaxMin.value === 0) return 0
  return (d.totalNetMin / chartMaxMin.value) * 100
}

const active = computed(() => {
  if (activeIdx.value == null) return null
  return days.value[activeIdx.value] || null
})

const formatStd = (min) => {
  const h = min / 60
  return `${Number.isInteger(h) ? h : h.toFixed(2)} h`
}

const positionTooltip = async () => {
  if (activeIdx.value == null || !container.value) return
  await nextTick()
  const cols = container.value.querySelectorAll('.flex.items-stretch > div')
  const col = cols[activeIdx.value]
  if (!col) return
  const colRect = col.getBoundingClientRect()
  const containerRect = container.value.getBoundingClientRect()
  tooltipLeft.value = colRect.left - containerRect.left + colRect.width / 2
}

watch(activeIdx, positionTooltip)
watch(() => props.items, positionTooltip)

const onEnter = (i) => {
  if (stickyIdx.value !== null) return
  hoveredIdx.value = i
}
const onLeave = () => {
  if (stickyIdx.value !== null) return
  hoveredIdx.value = null
}
const onClick = (i) => {
  stickyIdx.value = stickyIdx.value === i ? null : i
  hoveredIdx.value = stickyIdx.value
}

const onDocClick = (e) => {
  if (!container.value || !container.value.contains(e.target)) {
    stickyIdx.value = null
    hoveredIdx.value = null
  }
}

onMounted(() => document.addEventListener('click', onDocClick))
onBeforeUnmount(() => document.removeEventListener('click', onDocClick))
</script>
