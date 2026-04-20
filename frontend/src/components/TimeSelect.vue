<template>
  <div class="flex gap-1">
    <select :value="hour" @change="emitChange($event.target.value, minute)"
      class="flex-1 rounded-lg border border-gray-200 px-2 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500">
      <option value="">–</option>
      <option v-for="h in hours" :key="h" :value="h">{{ h }}</option>
    </select>
    <select :value="minute" @change="emitChange(hour, $event.target.value)"
      class="w-20 rounded-lg border border-gray-200 px-2 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500">
      <option v-for="m in minutes" :key="m" :value="m">:{{ m }}</option>
    </select>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])

const hours = Array.from({ length: 24 }, (_, i) => String(i).padStart(2, '0'))
const minutes = ['00', '15', '30', '45']

const roundMinute = (m) => {
  const n = parseInt(m, 10)
  if (isNaN(n)) return '00'
  return minutes.reduce((prev, cur) => (n >= parseInt(cur, 10) ? cur : prev), '00')
}

const parsed = computed(() => {
  if (!props.modelValue) return { hour: '', minute: '00' }
  const [h, m] = props.modelValue.split(':')
  return { hour: h.padStart(2, '0'), minute: roundMinute(m || '00') }
})

const hour = computed(() => parsed.value.hour)
const minute = computed(() => parsed.value.minute)

const emitChange = (h, m) => {
  emit('update:modelValue', h ? `${h}:${m}` : '')
}
</script>
