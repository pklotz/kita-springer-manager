<template>
  <div class="relative">
    <input
      type="text"
      :value="modelValue"
      @input="onInput"
      @focus="showSuggestions = true"
      placeholder="Haltestelle suchen…"
      class="w-full rounded-lg border border-gray-200 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500"
    />
    <ul v-if="showSuggestions && suggestions.length"
      class="absolute z-10 w-full bg-white border border-gray-200 rounded-lg shadow-lg mt-1 max-h-48 overflow-y-auto">
      <li v-for="s in suggestions" :key="s.id"
        @mousedown.prevent="select(s)"
        class="px-3 py-2 text-sm hover:bg-gray-50 cursor-pointer">
        {{ s.name }}
      </li>
    </ul>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { transitApi } from '../api'

const props = defineProps({ modelValue: String })
const emit = defineEmits(['update:modelValue'])

const suggestions = ref([])
const showSuggestions = ref(false)
let debounceTimer = null

const onInput = (e) => {
  emit('update:modelValue', e.target.value)
  clearTimeout(debounceTimer)
  if (e.target.value.length < 2) { suggestions.value = []; return }
  debounceTimer = setTimeout(async () => {
    const result = await transitApi.stops(e.target.value)
    suggestions.value = (result.stations || []).filter(s => s.name).slice(0, 8)
  }, 300)
}

const select = (s) => {
  emit('update:modelValue', s.name)
  suggestions.value = []
  showSuggestions.value = false
}
</script>
