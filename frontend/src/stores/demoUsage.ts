import { ref, computed } from 'vue'
import api from '@/services/api'

// Mirrors the backend's pgstore.demo_usage table and
// middleware.RequireDemoAllowance's feature strings exactly — these are the
// six features actually rate-limited for a demo ("user" role) account.
export const DEMO_TRIAL_LIMIT = 3

export interface DemoFeatureDef {
  key: string
  label: string
  icon: string
  color: string // CSS color value, not a Vuetify color name
}

export const DEMO_FEATURES: DemoFeatureDef[] = [
  { key: 'ambient', label: 'Ambient AI', icon: 'mdi-broadcast', color: '#2dd4f0' },
  { key: 'file_transcription', label: 'File transcription', icon: 'mdi-file-upload', color: '#5b9df5' },
  { key: 'dictation', label: 'Dictation', icon: 'mdi-microphone-message', color: '#a855f7' },
  { key: 'ai_summary', label: 'AI summary', icon: 'mdi-robot-outline', color: '#34e5a8' },
  { key: 'document_generation', label: 'Document generation', icon: 'mdi-file-document-outline', color: '#fb5aa0' },
  { key: 'search', label: 'Semantic search', icon: 'mdi-text-search', color: '#fbbf24' },
]

const usage = ref<Record<string, number>>({})
const isLoading = ref(false)
const loaded = ref(false)

export const demoUsage = computed(() => usage.value)
export const demoUsageLoading = computed(() => isLoading.value)

export function usageFor(feature: string): number {
  return usage.value[feature] ?? 0
}
export function remainingFor(feature: string): number {
  return Math.max(0, DEMO_TRIAL_LIMIT - usageFor(feature))
}
export function isMaxedOut(feature: string): boolean {
  return usageFor(feature) >= DEMO_TRIAL_LIMIT
}

/** GET /api/users/me/demo-usage — the caller's own lifetime per-feature counts. */
export async function loadDemoUsage(): Promise<void> {
  isLoading.value = true
  try {
    const res = await api.get('/users/me/demo-usage')
    usage.value = res.data.usage || {}
    loaded.value = true
  } catch (err) {
    console.error('Failed to load demo usage:', err)
  } finally {
    isLoading.value = false
  }
}

export const demoUsageLoaded = computed(() => loaded.value)
