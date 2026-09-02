import { ref } from 'vue'
import { getAuthHeaders, handleFetchResponse } from '@/stores/auth'

/** Structured clinical extraction returned by /api/ai/summarize (team schema). */
export interface AiExtraction {
  summary: string
  chiefComplaint: string
  history: string[]
  allergies: string[]
  medications: string[]
  symptoms: string[]
  diagnosis: string[]
  differentialDiagnosis: string[]
  treatment: string[]
  labTests: string[]
  procedures: string[]
  followUp: string[]
  riskFactors: string[]
  medicalTerms: string[]
  actionItems: string[]
}

export interface AiChatMessage {
  role: 'user' | 'assistant'
  content: string
}

const API_BASE = '/api'

const normalizeExtraction = (raw: Partial<AiExtraction> | null | undefined): AiExtraction => ({
  summary: raw?.summary || '',
  chiefComplaint: raw?.chiefComplaint || '',
  history: raw?.history || [],
  allergies: raw?.allergies || [],
  medications: raw?.medications || [],
  symptoms: raw?.symptoms || [],
  diagnosis: raw?.diagnosis || [],
  differentialDiagnosis: raw?.differentialDiagnosis || [],
  treatment: raw?.treatment || [],
  labTests: raw?.labTests || [],
  procedures: raw?.procedures || [],
  followUp: raw?.followUp || [],
  riskFactors: raw?.riskFactors || [],
  medicalTerms: raw?.medicalTerms || [],
  actionItems: raw?.actionItems || [],
})

/**
 * useAiAssistant — API calls + chat state for the AI Assistance module.
 * Stateless backend: this composable stores the extraction record after
 * summarize() and sends it (instead of the far larger transcript) as the
 * grounding for each chat question.
 */
export function useAiAssistant() {
  const isSummarizing = ref(false)
  const isAsking = ref(false)
  const extraction = ref<AiExtraction | null>(null)
  const chatMessages = ref<AiChatMessage[]>([])
  const error = ref<string>('')

  /**
   * Generate the structured clinical extraction for a completed transcript.
   */
  const summarize = async (transcript: string, language: string = 'en'): Promise<void> => {
    isSummarizing.value = true
    error.value = ''
    extraction.value = null
    chatMessages.value = []

    try {
      const response = await fetch(`${API_BASE}/ai/summarize`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({ transcript, language }),
      })

      handleFetchResponse(response)

      const data = await response.json()
      if (!response.ok || !data.success) {
        throw new Error(data.error || 'Failed to generate summary')
      }

      extraction.value = normalizeExtraction(data.extraction)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to generate summary'
      throw err
    } finally {
      isSummarizing.value = false
    }
  }

  /**
   * Ask a follow-up question grounded in the stored extraction record. Prior
   * turns are sent as history so the conversation stays coherent.
   */
  const ask = async (question: string): Promise<void> => {
    const trimmed = question.trim()
    if (!trimmed || isAsking.value) return
    if (!extraction.value) {
      error.value = 'No extracted record — generate a summary first'
      return
    }

    // History = everything before this question.
    const history = chatMessages.value.map(m => ({ role: m.role, content: m.content }))

    chatMessages.value.push({ role: 'user', content: trimmed })
    isAsking.value = true
    error.value = ''

    try {
      const response = await fetch(`${API_BASE}/ai/chat`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({ extraction: extraction.value, history, question: trimmed }),
      })

      handleFetchResponse(response)

      const data = await response.json()
      if (!response.ok || !data.success) {
        throw new Error(data.error || 'Failed to get an answer')
      }

      chatMessages.value.push({ role: 'assistant', content: data.answer })
    } catch (err) {
      // Drop the unanswered question so a retry doesn't duplicate it.
      chatMessages.value.pop()
      error.value = err instanceof Error ? err.message : 'Failed to get an answer'
      throw err
    } finally {
      isAsking.value = false
    }
  }

  /**
   * Restore a previously saved extraction (e.g. from a saved session) without
   * calling the API, so chat can resume grounded in the stored record.
   */
  const restore = (record: AiExtraction): void => {
    extraction.value = normalizeExtraction(record)
    chatMessages.value = []
    error.value = ''
  }

  /**
   * Reset all AI assistant state (e.g. when a new recording starts).
   */
  const reset = (): void => {
    extraction.value = null
    chatMessages.value = []
    error.value = ''
    isSummarizing.value = false
    isAsking.value = false
  }

  return {
    // State
    isSummarizing,
    isAsking,
    extraction,
    chatMessages,
    error,
    // Methods
    summarize,
    ask,
    restore,
    reset,
  }
}
