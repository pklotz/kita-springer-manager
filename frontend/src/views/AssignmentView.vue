<template>
  <div v-if="assignment">
    <div class="flex items-center gap-3 mb-6">
      <button @click="$router.back()" class="p-2 rounded-lg hover:bg-gray-200 transition-colors">
        <ArrowLeft class="w-5 h-5" />
      </button>
      <div class="flex-1 min-w-0">
        <h2 class="text-xl font-semibold truncate">{{ assignment.kita?.name }}</h2>
        <p class="text-gray-500 text-sm">{{ formatDate(assignment.date) }}</p>
      </div>
      <button @click="showEditForm = true"
        class="p-2 rounded-lg hover:bg-gray-200 text-gray-500 hover:text-gray-700 transition-colors">
        <Pencil class="w-5 h-5" />
      </button>
    </div>

    <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-4 mb-4">
      <!-- Planned time -->
      <div class="flex items-center gap-2 text-gray-700 mb-2">
        <Clock class="w-4 h-4 text-gray-400" />
        <span class="text-sm">
          <span class="text-xs text-gray-400 mr-1">Soll:</span>
          <span v-if="assignment.start_time">
            {{ assignment.start_time }}<span v-if="assignment.end_time"> – {{ assignment.end_time }}</span>
            <span v-if="plannedHours" class="text-gray-400 ml-2">({{ plannedHours }} h)</span>
          </span>
          <span v-else class="text-gray-400">Keine Zeit angegeben</span>
        </span>
      </div>

      <!-- Actual time -->
      <div v-if="hasActual" class="flex items-center gap-2 mb-2"
        :class="actualDiffers ? 'text-amber-700' : 'text-gray-700'">
        <CheckCircle2 class="w-4 h-4" :class="actualDiffers ? 'text-amber-500' : 'text-emerald-500'" />
        <span class="text-sm">
          <span class="text-xs mr-1" :class="actualDiffers ? 'text-amber-600' : 'text-gray-400'">Ist:</span>
          {{ assignment.actual_start_time || '–' }} – {{ assignment.actual_end_time || '–' }}
          <span v-if="actualHours" class="ml-2" :class="actualDiffers ? 'text-amber-600' : 'text-gray-400'">
            ({{ actualHours }} h<span v-if="hourDelta"> · {{ hourDelta }}</span>)
          </span>
        </span>
      </div>
      <div v-else-if="isPastOrToday" class="flex items-center gap-2 text-gray-400 text-sm mb-2">
        <CheckCircle2 class="w-4 h-4" />
        <button @click="showEditForm = true" class="underline hover:text-gray-600">Arbeitszeit erfassen</button>
      </div>

      <div class="flex items-start gap-2 text-gray-700 mb-2">
        <MapPin class="w-4 h-4 text-gray-400 mt-0.5" />
        <div>
          <div>{{ assignment.kita?.address || '–' }}</div>
          <div class="text-sm text-gray-500">
            {{ kitaStops.length > 1 ? 'Haltestellen' : 'Haltestelle' }}: {{ kitaStops.join(' · ') || '–' }}
          </div>
        </div>
      </div>
      <div v-if="assignment.notes" class="flex items-start gap-2 text-gray-700 mt-3 pt-3 border-t">
        <FileText class="w-4 h-4 text-gray-400 mt-0.5" />
        <span class="text-sm">{{ assignment.notes }}</span>
      </div>
    </div>

    <!-- Transit connections only for today and future -->
    <template v-if="isFuture">
      <div class="mb-2 flex items-center justify-between">
        <h3 class="font-semibold text-gray-700">Verbindungen (Ankunft {{ assignment.start_time }})</h3>
      </div>

      <div class="flex gap-2 mb-4">
        <input type="time" v-model="customTime"
          class="flex-1 rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500" />
        <select v-model="isArrival"
          class="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500">
          <option :value="true">Ankunft bis</option>
          <option :value="false">Abfahrt ab</option>
        </select>
        <button @click="loadConnections"
          class="bg-brand-500 text-white px-4 py-2 rounded-lg text-sm hover:bg-brand-600 transition-colors">
          Suchen
        </button>
      </div>

      <div v-if="loading" class="text-center text-gray-400 py-8">Verbindungen werden geladen…</div>
      <div v-else-if="error" class="bg-red-50 text-red-600 rounded-lg p-4 text-sm">{{ error }}</div>
      <div v-else-if="connections.length === 0" class="text-center text-gray-400 py-8">Keine Verbindungen gefunden</div>

      <div v-if="walkToFirstStop > 0" class="flex items-center gap-2 text-sm text-gray-500 mb-2 pl-1">
        <Footprints class="w-4 h-4" />
        <span>~{{ walkToFirstStop }} Min Fussweg zur ersten Haltestelle</span>
      </div>

      <ConnectionCard v-for="(c, i) in connections" :key="i" :connection="c" class="mb-3" />
    </template>
    <div v-else class="text-center text-gray-400 text-sm py-4 italic">
      Archivierter Einsatz – keine Verbindungsinformationen
    </div>

    <AssignmentForm v-if="showEditForm" :assignment="assignment"
      @close="showEditForm = false" @saved="onSaved" @deleted="onDeleted" />
  </div>

  <div v-else class="text-center text-gray-400 py-16">Wird geladen…</div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Clock, MapPin, FileText, CheckCircle2, Pencil, Footprints } from 'lucide-vue-next'
import dayjs from 'dayjs'
import 'dayjs/locale/de'
import { assignmentsApi, transitApi } from '../api'
import ConnectionCard from '../components/ConnectionCard.vue'
import AssignmentForm from '../components/AssignmentForm.vue'
import { diffMinutes, formatHours } from '../utils/time'

dayjs.locale('de')

const route = useRoute()
const router = useRouter()
const assignment = ref(null)
const connections = ref([])
const walkToFirstStop = ref(0)
const loading = ref(false)
const error = ref(null)
const customTime = ref('')
const isArrival = ref(true)
const showEditForm = ref(false)

const formatDate = (d) => dayjs(d).format('dddd, D. MMMM YYYY')

const today = dayjs().format('YYYY-MM-DD')
const isFuture = computed(() => assignment.value && assignment.value.date >= today)
const isPastOrToday = computed(() => assignment.value && assignment.value.date <= today)

const kitaStops = computed(() => {
  const k = assignment.value?.kita
  const stops = (k?.stops || []).filter(Boolean)
  if (stops.length) return stops
  return k?.stop_name ? [k.stop_name] : []
})

const hasActual = computed(() =>
  assignment.value && (assignment.value.actual_start_time || assignment.value.actual_end_time)
)

const plannedMinutes = computed(() =>
  assignment.value ? diffMinutes(assignment.value.start_time, assignment.value.end_time) : 0
)
const actualMinutes = computed(() =>
  assignment.value ? diffMinutes(assignment.value.actual_start_time, assignment.value.actual_end_time) : 0
)
const plannedHours = computed(() => formatHours(plannedMinutes.value))
const actualHours = computed(() => formatHours(actualMinutes.value))
const actualDiffers = computed(() => hasActual.value && actualMinutes.value !== plannedMinutes.value)
const hourDelta = computed(() => {
  if (!actualDiffers.value) return ''
  const d = actualMinutes.value - plannedMinutes.value
  const sign = d > 0 ? '+' : '−'
  return `${sign}${formatHours(Math.abs(d))} h`
})

const loadConnections = async () => {
  if (!assignment.value || !isFuture.value) return
  loading.value = true
  error.value = null
  try {
    const result = await transitApi.connections({
      assignment_id: assignment.value.id,
      time: customTime.value || assignment.value.start_time,
      is_arrival: isArrival.value ? '1' : '0',
    })
    connections.value = result.connections || []
    walkToFirstStop.value = result.walk_to_first_stop_minutes || 0
  } catch (e) {
    error.value = e.response?.data?.error || 'Verbindungen konnten nicht geladen werden'
  } finally {
    loading.value = false
  }
}

const loadAssignment = async () => {
  assignment.value = await assignmentsApi.get(route.params.id)
  customTime.value = assignment.value.start_time || ''
}

const onSaved = async () => {
  showEditForm.value = false
  await loadAssignment()
}

const onDeleted = () => {
  showEditForm.value = false
  // Don't refetch — the assignment is gone. Pop back to wherever the user
  // came from (calendar or history).
  router.back()
}

onMounted(async () => {
  await loadAssignment()
  if (isFuture.value) await loadConnections()
})
</script>
