<template>
  <div class="min-h-screen flex flex-col">
    <header class="bg-brand-500 text-white shadow-md">
      <div class="max-w-2xl mx-auto px-4 py-3 flex items-center justify-between">
        <span class="font-semibold text-lg">Kita Springer</span>
        <nav class="flex gap-4 text-sm">
          <RouterLink to="/" class="hover:text-blue-200 transition-colors" active-class="font-bold underline">Kalender</RouterLink>
          <RouterLink to="/worktime" class="hover:text-blue-200 transition-colors" active-class="font-bold underline">Arbeitszeit</RouterLink>
          <RouterLink to="/providers" class="hover:text-blue-200 transition-colors" active-class="font-bold underline">Kitas</RouterLink>
          <RouterLink to="/settings" class="hover:text-blue-200 transition-colors" active-class="font-bold underline">Einstellungen</RouterLink>
        </nav>
      </div>
    </header>

    <main class="flex-1 max-w-2xl w-full mx-auto px-4 py-6">
      <RouterView />
    </main>

    <ToastContainer />
  </div>
</template>

<script setup>
import { RouterLink, RouterView } from 'vue-router'
import ToastContainer from './components/ToastContainer.vue'
</script>
