<template>
  <div class="fixed inset-0 bg-black/40 flex items-end sm:items-center justify-center z-50 p-4"
    @click.self="$emit('close')">
    <div class="bg-white rounded-2xl w-full max-w-md p-6 shadow-xl max-h-[90vh] overflow-y-auto">
      <h3 v-if="title" class="font-semibold text-lg mb-4">{{ title }}</h3>
      <slot />
    </div>
  </div>
</template>

<script setup>
defineProps({ title: { type: String, default: '' } })
defineEmits(['close'])
</script>
