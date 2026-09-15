import { reactive } from 'vue'

// Shared, app-wide toast/snackbar state. Module-level (not a Pinia store)
// on purpose — this mirrors the same pattern stores/auth.ts already uses
// for shared reactive state, so there's exactly one queue instead of every
// page reimplementing its own showSuccess/showError refs + duplicate
// <v-snackbar> markup (previously duplicated across 15+ files).
export type ToastColor = 'success' | 'error' | 'warning' | 'info'

export interface ToastMessage {
  id: number
  text: string
  color: ToastColor
  timeout: number
}

let nextId = 1

const state = reactive<{ queue: ToastMessage[] }>({ queue: [] })

function push(text: string, color: ToastColor, timeout: number) {
  state.queue.push({ id: nextId++, text, color, timeout })
}

function dismiss(id: number) {
  const i = state.queue.findIndex(m => m.id === id)
  if (i !== -1) state.queue.splice(i, 1)
}

// The composable — call from any component's <script setup>.
export function useToast() {
  return {
    success: (text: string, timeout = 3000) => push(text, 'success', timeout),
    error: (text: string, timeout = 5000) => push(text, 'error', timeout),
    warning: (text: string, timeout = 4000) => push(text, 'warning', timeout),
    info: (text: string, timeout = 3000) => push(text, 'info', timeout),
  }
}

// Internal — only ToastHost.vue (mounted once in App.vue) should use these.
export function useToastQueue() {
  return { queue: state.queue, dismiss }
}
