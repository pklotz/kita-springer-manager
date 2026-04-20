<template>
  <div>
    <CalendarGrid
      :month="currentMonth"
      :assignments="assignments"
      :closures="closures"
      :providers="providers"
      @prev="prevMonth"
      @next="nextMonth"
      @open-assignment="a => $router.push(`/assignments/${a.id}`)" />

    <div class="flex items-center justify-between mb-3 flex-wrap gap-2">
      <h3 class="font-semibold text-gray-700">Nächste Einsätze</h3>
      <div class="flex gap-2 flex-wrap">
        <RouterLink to="/history"
          class="flex items-center gap-1 text-sm bg-gray-100 text-gray-600 px-3 py-1.5 rounded-lg hover:bg-gray-200 transition-colors">
          <History class="w-4 h-4" /> Historie
        </RouterLink>
        <button @click="showClosureForm = true"
          class="flex items-center gap-1 text-sm bg-gray-100 text-gray-600 px-3 py-1.5 rounded-lg hover:bg-gray-200 transition-colors">
          <CalendarOff class="w-4 h-4" /> Abwesenheit
        </button>
        <button @click="showRecurringForm = true"
          class="flex items-center gap-1 text-sm bg-purple-50 text-purple-700 px-3 py-1.5 rounded-lg hover:bg-purple-100 transition-colors">
          <Repeat class="w-4 h-4" /> Fixe Einsätze
        </button>
        <button @click="editAssignment = null; showForm = true"
          class="flex items-center gap-1 text-sm bg-brand-500 text-white px-3 py-1.5 rounded-lg hover:bg-brand-600 transition-colors">
          <Plus class="w-4 h-4" /> Einsatz
        </button>
      </div>
    </div>

    <UpcomingList :assignments="upcomingAssignments" :providers="providers"
      @edit="a => { editAssignment = a; showForm = true }"
      @open-detail="a => $router.push(`/assignments/${a.id}`)"
      @bulk-delete="bulkDelete" />

    <VacationList :vacations="upcomingVacations" @remove="removeClosure" />

    <AssignmentForm v-if="showForm" :assignment="editAssignment" @close="showForm = false" @saved="onSaved" />
    <ClosureForm v-if="showClosureForm" @close="showClosureForm = false" @saved="onClosureSaved" />
    <RecurringForm v-if="showRecurringForm" @close="showRecurringForm = false" @saved="load" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { Plus, CalendarOff, Repeat, History } from 'lucide-vue-next'
import { RouterLink } from 'vue-router'
import dayjs from 'dayjs'
import { assignmentsApi, providersApi, closuresApi } from '../api'
import AssignmentForm from '../components/AssignmentForm.vue'
import ClosureForm from '../components/ClosureForm.vue'
import RecurringForm from '../components/RecurringForm.vue'
import CalendarGrid from '../components/CalendarGrid.vue'
import UpcomingList from '../components/UpcomingList.vue'
import VacationList from '../components/VacationList.vue'

const assignments = ref([])
const providers = ref([])
const closures = ref([])
const showForm = ref(false)
const showClosureForm = ref(false)
const showRecurringForm = ref(false)
const editAssignment = ref(null)
const currentMonth = ref(dayjs().startOf('month'))

const prevMonth = () => { currentMonth.value = currentMonth.value.subtract(1, 'month') }
const nextMonth = () => { currentMonth.value = currentMonth.value.add(1, 'month') }

const upcomingAssignments = computed(() =>
  assignments.value.filter(a => a.date >= dayjs().format('YYYY-MM-DD'))
)

const bulkDelete = async (ids) => {
  await assignmentsApi.bulkDelete(ids)
  load()
}

const upcomingVacations = computed(() =>
  closures.value
    .filter(c => (c.type === 'springerin' || c.type === 'provider' || c.type === 'kita') &&
      c.date >= dayjs().format('YYYY-MM-DD'))
    .slice(0, 20)
)

const removeClosure = async (c) => {
  if (confirm('Abwesenheit löschen?')) {
    await closuresApi.delete(c.id)
    loadClosures()
  }
}

const loadClosures = async () => {
  const from = currentMonth.value.subtract(1, 'month').format('YYYY-MM-DD')
  const to = dayjs().add(6, 'month').format('YYYY-MM-DD')
  closures.value = await closuresApi.list({ from, to })
}

const load = async () => {
  [assignments.value, providers.value] = await Promise.all([
    assignmentsApi.list(),
    providersApi.list(),
  ])
  await loadClosures()
}

const onSaved = () => { showForm.value = false; editAssignment.value = null; load() }
const onClosureSaved = () => { showClosureForm.value = false; loadClosures() }

onMounted(load)
</script>
