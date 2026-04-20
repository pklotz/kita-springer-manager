<template>
  <div class="fixed bottom-4 right-4 z-[100] flex flex-col gap-2 max-w-sm">
    <transition-group name="toast">
      <div v-for="t in toasts.items" :key="t.id"
        :class="['px-4 py-3 rounded-lg shadow-lg text-sm font-medium border',
          t.type === 'error' ? 'bg-red-50 border-red-200 text-red-700'
          : t.type === 'success' ? 'bg-green-50 border-green-200 text-green-700'
          : 'bg-gray-50 border-gray-200 text-gray-700']">
        {{ t.message }}
      </div>
    </transition-group>
  </div>
</template>

<script setup>
import { useToasts } from '../toast'
const toasts = useToasts()
</script>

<style>
.toast-enter-active, .toast-leave-active { transition: all 0.2s ease; }
.toast-enter-from { opacity: 0; transform: translateY(8px); }
.toast-leave-to { opacity: 0; transform: translateX(16px); }
</style>
