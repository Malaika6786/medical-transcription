import { ref, computed } from 'vue'
import axios from 'axios'
import { loadSessions, clearUserSessions } from './sessions'
import { handleSessionExpired } from '@/services/api'

export interface User {
  id: string
  username: string
  email: string
  name: string
  roles: string[]
  permissions: string[]        // effective permissions, embedded in JWT and returned by /api/auth/me
  granted_permissions?: string[]
  denied_permissions?: string[]
  isActive: boolean
  // "pending" | "approved" | "rejected". While not "approved", permissions
  // above is already empty (server-side, see backend auth.User.IsApproved)
  // — this field is what the UI uses to decide which screen to show.
  status: string
  requestedRole?: string
  createdAt: string
  lastLogin?: string
}

const TOKEN_KEY = 'xstek_auth_token'
const USER_KEY  = 'xstek_auth_user'

// Reactive state
const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))
const user  = ref<User | null>(null)
const isLoading  = ref(false)
const error      = ref<string | null>(null)
const authLoading = ref(false)

// Load user from localStorage on init
const storedUser = localStorage.getItem(USER_KEY)
if (storedUser) {
  try {
    user.value = JSON.parse(storedUser)
  } catch {
    localStorage.removeItem(USER_KEY)
  }
}

// Core computed properties
export const isAuthenticated = computed(() => !!token.value && !!user.value)
export const currentUser  = computed(() => user.value)
export const authToken    = computed(() => token.value)
export const authError    = computed(() => error.value)
export { authLoading }

// --- Permission system ---

/**
 * hasPermission checks whether the current user's effective permission list
 * includes the given permission string.
 * This is the single source of truth for all access control in the frontend.
 */
export function hasPermission(perm: string): boolean {
  return user.value?.permissions?.includes(perm) ?? false
}

// Convenience computed aliases — kept for backward-compat with existing v-if bindings.
export const isSuperUser  = computed(() => hasPermission('users.manage'))
export const isApproved   = computed(() => user.value?.status === 'approved')
export const isDemoAccount = computed(() => user.value?.roles?.includes('user') ?? false)

// defaultLandingPath is the single source of truth for "where does an
// approved user go after login" — a superuser's main job is running the
// app (Command Center), everyone else's is the clinical work (My
// Dashboard). Used by router.beforeEach and every page that redirects an
// already-authenticated user (Login/Signup/PendingApproval), so all of
// them agree.
export function defaultLandingPath(): string {
  return hasPermission('users.manage') ? '/command-center' : '/my-dashboard'
}

export const canAccessAmbient           = computed(() => hasPermission('ambient.access'))
export const canAccessFileTranscription = computed(() => hasPermission('file_transcription.access'))
export const canAccessDictation         = computed(() => hasPermission('dictation.access'))
export const canViewClinicalFacts       = computed(() => hasPermission('clinical_facts.view'))
export const canManageUsers             = computed(() => hasPermission('users.manage'))
export const canViewDocumentation       = computed(() => hasPermission('documentation.view'))
export const canAccessEmbeddedAssistant = computed(() => hasPermission('embedded_assistant.access'))
export const canManageTemplates         = computed(() => hasPermission('templates.manage'))
export const canViewCortiSections       = computed(() => hasPermission('corti_sections.view'))

// Configure axios defaults if a token is already stored.
if (token.value) {
  axios.defaults.headers.common['Authorization'] = `Bearer ${token.value}`
  loadSessions()
}

// Actions

// login accepts either the account's username or its email.
export async function login(loginId: string, password: string): Promise<boolean> {
  isLoading.value = true
  error.value = null

  try {
    const response = await axios.post('/api/auth/login', { login: loginId, password })
    return applyAuthResponse(response.data)
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Login failed. Please try again.'
    return false
  } finally {
    isLoading.value = false
  }
}

export interface SignupPayload {
  username: string
  email: string
  password: string
  name: string
  role: string
}

export async function signup(payload: SignupPayload): Promise<boolean> {
  isLoading.value = true
  error.value = null

  try {
    const response = await axios.post('/api/auth/signup', payload)
    return applyAuthResponse(response.data)
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Signup failed. Please try again.'
    return false
  } finally {
    isLoading.value = false
  }
}

// applyAuthResponse stores the token/user from a successful login or signup
// response and loads the new session's saved sessions.
async function applyAuthResponse(data: any): Promise<boolean> {
  if (data.success) {
    token.value = data.token
    user.value  = data.user

    localStorage.setItem(TOKEN_KEY, data.token)
    localStorage.setItem(USER_KEY,  JSON.stringify(data.user))

    axios.defaults.headers.common['Authorization'] = `Bearer ${data.token}`

    await loadSessions()
    return true
  }
  error.value = data.error || 'Request failed'
  return false
}

export function logout(redirectToLogin: boolean = true) {
  token.value = null
  user.value  = null
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
  delete axios.defaults.headers.common['Authorization']
  clearUserSessions()

  if (redirectToLogin) {
    window.location.href = '/login?session_expired=true'
  }
}

export async function refreshUser(): Promise<void> {
  if (!token.value) return
  try {
    const response = await axios.get('/api/auth/me')
    if (response.data.success) {
      user.value = response.data.user
      localStorage.setItem(USER_KEY, JSON.stringify(response.data.user))
    }
  } catch {
    logout()
  }
}

export function clearError() {
  error.value = null
}

export function getAuthHeaders(): Record<string, string> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token.value) {
    headers['Authorization'] = `Bearer ${token.value}`
  }
  return headers
}

// DemoLimitError carries the 403 demoLimitReached response body so callers
// can show its message directly rather than a generic failure.
export class DemoLimitError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'DemoLimitError'
  }
}

export function handleFetchResponse(response: Response): Response {
  if (response.status === 401) {
    handleSessionExpired('Your session has expired. Please log in again.')
    throw new Error('Session expired')
  }
  return response
}

// Like handleFetchResponse, but also surfaces a 403 demoLimitReached body as
// a DemoLimitError with the backend's own message, instead of letting
// callers fall through to a generic error later. Call sites that already
// use handleFetchResponse should call this first (await response.clone().json()
// isn't needed — this reads the body once and callers use the return value).
export async function checkDemoLimit(response: Response): Promise<Response> {
  if (response.status === 403) {
    const clone = response.clone()
    try {
      const data = await clone.json()
      if (data?.demoLimitReached) {
        throw new DemoLimitError(data.error || "You've used all 3 free trials for this feature.")
      }
    } catch (e) {
      if (e instanceof DemoLimitError) throw e
      // Not JSON / not our shape — fall through, let the normal error path handle it.
    }
  }
  return handleFetchResponse(response)
}
