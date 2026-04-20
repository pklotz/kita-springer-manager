import { reactive } from 'vue'

let nextId = 1
const state = reactive({ items: [] })

export function useToasts() {
  return state
}

export function showToast(message, type = 'error', timeoutMs = 4000) {
  const id = nextId++
  state.items.push({ id, message, type })
  setTimeout(() => {
    const idx = state.items.findIndex(t => t.id === id)
    if (idx >= 0) state.items.splice(idx, 1)
  }, timeoutMs)
}
