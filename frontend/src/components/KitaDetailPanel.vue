<template>
  <Transition name="panel">
    <div v-if="kita" class="fixed inset-0 z-40 flex" @click.self="$emit('close')">
      <div class="flex-1" @click="$emit('close')" />

      <div class="w-80 max-w-[90vw] bg-white shadow-2xl h-full overflow-y-auto flex flex-col" @click.stop>
        <!-- Header -->
        <div class="flex items-start justify-between p-5 border-b border-gray-100 sticky top-0 bg-white z-10">
          <div>
            <h3 class="font-bold text-lg leading-tight text-gray-900">{{ kita.name }}</h3>
            <span v-if="provider" class="inline-block mt-1 text-xs px-2 py-0.5 rounded-full text-white"
              :style="{ backgroundColor: provider.color_hex }">
              {{ provider.name }}
            </span>
          </div>
          <button @click="$emit('close')"
            class="p-1.5 rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors shrink-0 ml-2">
            <X class="w-5 h-5" />
          </button>
        </div>

        <!-- Body -->
        <div class="p-5 space-y-5 flex-1">
          <img v-if="kita.photo_url" :src="kita.photo_url" :alt="kita.name"
            class="w-full h-40 object-cover rounded-xl" />

          <div v-if="kita.address">
            <div class="flex items-center gap-1.5 text-xs font-semibold text-gray-400 uppercase tracking-wide mb-1.5">
              <MapPin class="w-3.5 h-3.5" /> Adresse
            </div>
            <p class="text-sm text-gray-700">{{ kita.address }}</p>
            <a :href="mapsUrl(kita.address)" target="_blank" rel="noopener"
              class="inline-flex items-center gap-1.5 mt-2 text-xs text-brand-500 hover:text-brand-600 transition-colors">
              <ExternalLink class="w-3.5 h-3.5" /> In Google Maps öffnen
            </a>
          </div>

          <div v-if="kita.leitung_name">
            <div class="flex items-center gap-1.5 text-xs font-semibold text-gray-400 uppercase tracking-wide mb-1.5">
              <Users class="w-3.5 h-3.5" /> Leitung
            </div>
            <p class="text-sm text-gray-700">{{ kita.leitung_name }}</p>
          </div>

          <div v-if="kita.phone || kita.email">
            <div class="flex items-center gap-1.5 text-xs font-semibold text-gray-400 uppercase tracking-wide mb-1.5">
              <Phone class="w-3.5 h-3.5" /> Kontakt
            </div>
            <a v-if="kita.phone" :href="`tel:${kita.phone}`"
              class="flex items-center gap-2 text-sm text-gray-700 hover:text-brand-500 transition-colors py-1">
              <Phone class="w-4 h-4 text-gray-400" />
              {{ kita.phone }}
            </a>
            <a v-if="kita.email" :href="`mailto:${kita.email}`"
              class="flex items-center gap-2 text-sm text-gray-700 hover:text-brand-500 transition-colors py-1 break-all">
              <Mail class="w-4 h-4 text-gray-400 shrink-0" />
              {{ kita.email }}
            </a>
          </div>

          <div v-if="stops.length">
            <div class="flex items-center gap-1.5 text-xs font-semibold text-gray-400 uppercase tracking-wide mb-1.5">
              <Bus class="w-3.5 h-3.5" /> ÖV-Haltestelle{{ stops.length > 1 ? 'n' : '' }}
            </div>
            <ul class="text-sm text-gray-700 space-y-0.5">
              <li v-for="s in stops" :key="s">{{ s }}</li>
            </ul>
          </div>

          <div v-if="kita.groups?.length">
            <div class="flex items-center gap-1.5 text-xs font-semibold text-gray-400 uppercase tracking-wide mb-2">
              <Users class="w-3.5 h-3.5" /> Gruppen
            </div>
            <div class="flex flex-wrap gap-1.5">
              <span v-for="g in kita.groups" :key="g"
                class="text-xs bg-gray-100 text-gray-600 px-2.5 py-1 rounded-full">{{ g }}</span>
            </div>
          </div>

          <div v-if="kita.notes">
            <div class="flex items-center gap-1.5 text-xs font-semibold text-gray-400 uppercase tracking-wide mb-1.5">
              <FileText class="w-3.5 h-3.5" /> Notizen
            </div>
            <p class="text-sm text-gray-600 whitespace-pre-line">{{ kita.notes }}</p>
          </div>
        </div>

        <!-- Optional footer (e.g. edit/delete actions) -->
        <div v-if="$slots.footer" class="p-4 border-t border-gray-100 flex gap-2">
          <slot name="footer" />
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { computed } from 'vue'
import { MapPin, Phone, Mail, Bus, Users, FileText, X, ExternalLink } from 'lucide-vue-next'

const props = defineProps({
  kita: Object,
  provider: Object,
})

defineEmits(['close'])

const stops = computed(() => {
  const s = (props.kita?.stops || []).filter(Boolean)
  if (s.length) return s
  return props.kita?.stop_name ? [props.kita.stop_name] : []
})

const mapsUrl = (address) =>
  `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(address)}`
</script>

<style scoped>
.panel-enter-active,
.panel-leave-active {
  transition: opacity 0.2s ease;
}
.panel-enter-active > div:last-child,
.panel-leave-active > div:last-child {
  transition: transform 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}
.panel-enter-from,
.panel-leave-to {
  opacity: 0;
}
.panel-enter-from > div:last-child,
.panel-leave-to > div:last-child {
  transform: translateX(100%);
}
</style>
