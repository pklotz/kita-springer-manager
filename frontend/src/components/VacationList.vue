<template>
  <div v-if="vacations.length" class="mt-2">
    <h3 class="font-semibold text-gray-700 mb-3">Urlaub / Abwesenheiten</h3>
    <div v-for="c in vacations" :key="c.id"
      class="flex items-center justify-between bg-gray-50 border border-gray-200 rounded-xl px-4 py-3 mb-2">
      <div>
        <span class="font-medium text-gray-700">{{ closureLabel(c) }}</span>
        <span class="text-sm text-gray-500 ml-2">{{ formatDate(c.date) }}</span>
        <span v-if="c.note" class="text-sm text-gray-400 ml-2">· {{ c.note }}</span>
      </div>
      <button @click="$emit('remove', c)"
        class="p-1.5 rounded-lg hover:bg-red-50 text-gray-400 hover:text-red-500 transition-colors">
        <Trash2 class="w-4 h-4" />
      </button>
    </div>
  </div>
</template>

<script setup>
import { Trash2 } from 'lucide-vue-next'
import dayjs from 'dayjs'
import 'dayjs/locale/de'
import { closureLabel } from '../utils/closures'

dayjs.locale('de')

defineProps({
  vacations: { type: Array, default: () => [] },
})
defineEmits(['remove'])

const formatDate = (d) => dayjs(d).format('dddd, D. MMMM YYYY')
</script>
