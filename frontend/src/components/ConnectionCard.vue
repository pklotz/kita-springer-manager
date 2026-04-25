<template>
  <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-4">
    <!-- Summary header -->
    <div class="flex items-center justify-between mb-3 pb-3 border-b border-gray-50">
      <div>
        <div class="text-2xl font-bold text-gray-800 leading-none">{{ departureTime }}</div>
        <div class="text-xs text-gray-500 mt-1">{{ fromStation }}</div>
      </div>
      <div class="flex-1 flex flex-col items-center px-3">
        <div class="text-xs text-gray-500 font-medium">{{ duration }}</div>
        <div class="flex gap-1 mt-1 flex-wrap justify-center">
          <span v-for="(leg, i) in transportLegs" :key="i"
            :class="['text-[11px] px-1.5 py-0.5 rounded font-bold', badgeClass(leg.category)]">
            {{ leg.label }}
          </span>
          <span v-if="walkCount" class="text-[11px] px-1.5 py-0.5 rounded bg-gray-100 text-gray-500 font-medium">
            🚶 {{ walkCount }}×
          </span>
        </div>
      </div>
      <div class="text-right">
        <div class="text-2xl font-bold text-gray-800 leading-none">{{ arrivalTime }}</div>
        <div class="text-xs text-gray-500 mt-1">{{ toStation }}</div>
      </div>
    </div>

    <!-- Per-section timeline -->
    <div class="space-y-3">
      <div v-for="(s, i) in connection.sections" :key="i">
        <!-- Walk -->
        <div v-if="s.walk" class="flex items-center gap-3 text-xs text-gray-500 italic pl-2">
          <span class="w-12 text-right">🚶</span>
          <span>Fussweg {{ s.walk.duration }} Min.</span>
        </div>

        <!-- Journey (Bus / Tram / Train) -->
        <div v-else-if="s.journey" class="flex gap-3">
          <!-- Line badge column -->
          <div class="flex flex-col items-center shrink-0 w-14">
            <span :class="['text-sm px-2 py-1 rounded font-bold leading-none', badgeClass(s.journey.category)]">
              {{ lineLabel(s.journey) }}
            </span>
          </div>

          <!-- Stops column -->
          <div class="flex-1 min-w-0">
            <div v-if="s.journey.to" class="text-[11px] text-gray-500 uppercase tracking-wide mb-1">
              Richtung {{ s.journey.to }}
            </div>

            <div class="relative pl-4">
              <!-- Vertical line -->
              <div class="absolute left-[5px] top-2 bottom-2 w-px bg-gray-200" />

              <!-- Departure -->
              <div class="flex items-baseline gap-2 mb-1 relative">
                <span class="absolute -left-4 top-1.5 w-2.5 h-2.5 rounded-full bg-brand-500 border-2 border-white ring-1 ring-brand-500" />
                <span class="font-mono text-sm font-semibold text-gray-800 shrink-0 w-10">{{ fmt(s.departure?.departure) }}</span>
                <span class="text-sm text-gray-700 truncate">{{ s.departure?.station?.name }}</span>
                <span v-if="s.departure?.platform" class="text-[11px] text-gray-400 shrink-0 ml-auto">
                  Gl./Kante {{ s.departure.platform }}
                </span>
              </div>

              <!-- Arrival -->
              <div class="flex items-baseline gap-2 relative">
                <span class="absolute -left-4 top-1.5 w-2.5 h-2.5 rounded-full bg-white border-2 border-gray-400" />
                <span class="font-mono text-sm font-semibold text-gray-800 shrink-0 w-10">{{ fmt(s.arrival?.arrival) }}</span>
                <span class="text-sm text-gray-700 truncate">{{ s.arrival?.station?.name }}</span>
                <span v-if="s.arrival?.platform" class="text-[11px] text-gray-400 shrink-0 ml-auto">
                  Gl./Kante {{ s.arrival.platform }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import dayjs from 'dayjs'

const props = defineProps({ connection: Object })

const fmt = (ts) => ts ? dayjs(ts).format('HH:mm') : '–'

const departureTime = computed(() => fmt(props.connection.from?.departure))
const arrivalTime = computed(() => fmt(props.connection.to?.arrival))
const fromStation = computed(() => props.connection.from?.station?.name || '')
const toStation = computed(() => props.connection.to?.station?.name || '')

const duration = computed(() => {
  const d = props.connection.duration
  if (!d) return ''
  const match = d.match(/(\d+):(\d+):\d+$/)
  if (!match) return d
  const [, h, m] = match
  return h !== '00' ? `${parseInt(h)}h ${parseInt(m)}m` : `${parseInt(m)} Min.`
})

// Category codes from transport.opendata.ch: B (Bus), T (Tram), S (S-Bahn),
// IC/IR/RE/R (trains), FUN (funicular), etc.
const badgeClass = (cat) => {
  const c = (cat || '').toUpperCase()
  if (c === 'T') return 'bg-red-100 text-red-700'
  if (c === 'B') return 'bg-amber-100 text-amber-800'
  if (c === 'S') return 'bg-blue-100 text-blue-700'
  if (['IC', 'ICE', 'IR', 'RE', 'R', 'EC'].includes(c)) return 'bg-indigo-100 text-indigo-700'
  return 'bg-gray-100 text-gray-700'
}

const lineLabel = (j) => {
  if (!j) return '–'
  const cat = (j.category || '').toUpperCase()
  const num = j.number || ''
  if (cat === 'T') return `Tram ${num}`.trim()
  if (cat === 'B') return `Bus ${num}`.trim()
  if (cat === 'S') return `S${num}`.trim()
  return j.name || `${cat} ${num}`.trim()
}

const transportLegs = computed(() =>
  (props.connection.sections || [])
    .filter(s => s.journey)
    .map(s => ({ category: s.journey.category, label: lineLabel(s.journey) }))
)

const walkCount = computed(() =>
  (props.connection.sections || []).filter(s => s.walk).length
)
</script>
