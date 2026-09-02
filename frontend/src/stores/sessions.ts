import { ref, computed } from 'vue'
import { currentUser } from './auth'
import api from '@/services/api'
import type { AiExtraction } from '@/composables/useAiAssistant'

export interface SavedDocument {
  id: string
  templateKey: string
  templateName: string
  sections: Array<{
    key?: string
    name: string
    text?: string
    content?: string
  }>
  htmlContent?: string  // Store the formatted/edited HTML content
  // Letter template details (for GP Letter and similar templates)
  letterDetails?: {
    patientName?: string
    recipientDoctor?: string
    senderDoctor?: string
    senderTitle?: string
  }
  createdAt: string
  updatedAt: string
}

export interface SavedSession {
  id: string
  userId: string
  type: 'ambient' | 'file-transcription' | 'dictation' | 'embedded-assistant'
  title: string
  transcript: string
  document?: SavedDocument
  extraction?: AiExtraction
  interactionId?: string
  createdAt: string
  updatedAt: string
}

// Reactive state
const allSessions = ref<SavedSession[]>([])
const isLoading = ref(false)
const lastError = ref<string | null>(null)

// Get sessions for current user only (from backend, already filtered)
// Sorted by createdAt descending (newest first) so sessions don't rearrange on edit
export const userSessions = computed(() => {
  return allSessions.value
    .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
})

// Generate unique ID
const generateId = () => {
  return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
}

// Load sessions from backend
export async function loadSessions(): Promise<void> {
  if (!currentUser.value?.id) {
    allSessions.value = []
    return
  }

  isLoading.value = true
  lastError.value = null

  try {
    const response = await api.get('/sessions')
    allSessions.value = response.data || []
  } catch (error: unknown) {
    console.error('Failed to load sessions:', error)
    lastError.value = 'Failed to load sessions'
    allSessions.value = []
  } finally {
    isLoading.value = false
  }
}

// Save a new session or update existing one
export async function saveSession(
  sessionData: Omit<SavedSession, 'id' | 'userId' | 'createdAt' | 'updatedAt'> & { id?: string }
): Promise<SavedSession> {
  const userId = currentUser.value?.id
  if (!userId) {
    throw new Error('User must be logged in to save sessions')
  }

  const sessionId = sessionData.id || generateId()
  
  const payload = {
    id: sessionId,
    type: sessionData.type,
    title: sessionData.title,
    transcript: sessionData.transcript,
    document: sessionData.document,
    extraction: sessionData.extraction,
    interactionId: sessionData.interactionId
  }

  try {
    const response = await api.post('/sessions', payload)
    const savedSession = response.data
    
    // Update local state
    const existingIndex = allSessions.value.findIndex(s => s.id === savedSession.id)
    if (existingIndex !== -1) {
      allSessions.value[existingIndex] = savedSession
    } else {
      allSessions.value.push(savedSession)
    }
    
    return savedSession
  } catch (error) {
    console.error('Failed to save session:', error)
    throw new Error('Failed to save session to server')
  }
}

// Delete a session
export async function deleteSession(sessionId: string): Promise<boolean> {
  const userId = currentUser.value?.id
  if (!userId) return false

  try {
    await api.delete(`/sessions/${sessionId}`)
    allSessions.value = allSessions.value.filter(s => s.id !== sessionId)
    return true
  } catch (error) {
    console.error('Failed to delete session:', error)
    return false
  }
}

// Get a specific session
export function getSession(sessionId: string): SavedSession | undefined {
  return allSessions.value.find(s => s.id === sessionId)
}

// Update session document
export async function updateSessionDocument(sessionId: string, document: SavedDocument): Promise<boolean> {
  const userId = currentUser.value?.id
  if (!userId) return false

  try {
    const response = await api.put(`/sessions/${sessionId}/document`, { document })
    const updatedSession = response.data
    
    // Update local state
    const session = allSessions.value.find(s => s.id === sessionId)
    if (session) {
      session.document = updatedSession.document
      session.updatedAt = updatedSession.updatedAt
    }
    
    return true
  } catch (error) {
    console.error('Failed to update session document:', error)
    return false
  }
}

// Get session count for current user
export const sessionCount = computed(() => userSessions.value.length)

// Loading state
export const sessionsLoading = computed(() => isLoading.value)

// Clear all sessions for current user (local only - for logout)
export function clearUserSessions(): void {
  allSessions.value = []
}
