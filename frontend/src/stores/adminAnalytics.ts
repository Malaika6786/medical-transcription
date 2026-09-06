import { ref, computed } from 'vue'
import api from '@/services/api'

export interface AnalyticsOverview {
  totalUsers: number
  usersByRole: Record<string, number> // doctor, superuser, admin, user (demo)
  totalSessions: number
  sessionsByType: Record<string, number>
  sessionsLast24h: number
  sessionsLast7d: number
  sessionsPrev7d: number
  sessionsLast30d: number
  pendingApprovals: number
  demoAccountsCount: number
  demoAccountsAtLimit: number
}

export interface DayUsage {
  date: string // YYYY-MM-DD
  ambient: number
  file: number
  dictation: number
}

export interface DemoFunnelUser {
  userId: string
  username: string
  requestedRole: string
  approvalStatus: string
  createdAt: string
  featureUsage: Record<string, number> // ambient, file_transcription, dictation, ai_summary, document_generation, search
}

export interface RecentSessionEntry {
  id: string
  userId: string
  userName: string
  username: string
  isDemo: boolean
  type: string
  title: string
  transcript: string
  document: boolean
  createdAt: string
}

const overview = ref<AnalyticsOverview | null>(null)
const timeseries = ref<DayUsage[]>([])
const demoFunnel = ref<DemoFunnelUser[]>([])
const recentSessions = ref<RecentSessionEntry[]>([])
const isLoading = ref(false)
const lastError = ref<string | null>(null)

export const analyticsOverview = computed(() => overview.value)
export const usageTimeseries = computed(() => timeseries.value)
export const demoFunnelUsers = computed(() => demoFunnel.value)
export const recentSessionsAllUsers = computed(() => recentSessions.value)
export const analyticsLoading = computed(() => isLoading.value)
export const analyticsError = computed(() => lastError.value)

// DEMO_TRIAL_LIMIT mirrors the backend's middleware.demoTrialLimit — 3 free
// uses per feature for a demo ("user" role) account.
export const DEMO_TRIAL_LIMIT = 3

// loadAnalytics fetches everything the Command Center needs in parallel.
export async function loadAnalytics(days = 30): Promise<void> {
  isLoading.value = true
  lastError.value = null
  try {
    const [overviewRes, seriesRes, funnelRes, recentRes] = await Promise.all([
      api.get('/admin/analytics/overview'),
      api.get('/admin/analytics/usage-timeseries', { params: { days } }),
      api.get('/admin/analytics/demo-funnel'),
      api.get('/admin/analytics/recent-sessions', { params: { limit: 30 } })
    ])
    overview.value = overviewRes.data.overview
    timeseries.value = seriesRes.data.timeseries || []
    demoFunnel.value = funnelRes.data.demoUsers || []
    recentSessions.value = recentRes.data.sessions || []
  } catch (err: unknown) {
    console.error('Failed to load Command Center analytics:', err)
    lastError.value = 'Failed to load analytics'
  } finally {
    isLoading.value = false
  }
}
