import { ref } from 'vue'
import { getAuthHeaders, checkDemoLimit, handleFetchResponse } from '@/stores/auth'

// Helper to get auth headers without Content-Type for FormData
const getAuthHeadersForFormData = (): Record<string, string> => {
  const headers = getAuthHeaders()
  // Remove Content-Type for FormData (browser will set it automatically with boundary)
  delete headers['Content-Type']
  return headers
}

interface TranscriptResult {
  text: string
  status: string
  segments?: TranscriptSegment[]
  duration?: number
  language?: string
}

interface TranscriptSegment {
  id: string
  text: string
  startTime: number
  endTime: number
  speaker?: string
  confidence?: number
}

interface UploadResponse {
  success: boolean
  interaction_id: string
  recording_id?: string
  transcript_id?: string
  status: string
  message?: string
  transcript?: TranscriptResult  // present when Corti returns transcript synchronously
}

interface TranscriptResponse {
  success: boolean
  status: string
  transcript?: TranscriptResult
  is_complete: boolean
  error?: string
}

const API_BASE = '/api'

export function useTranscription() {
  const isUploading = ref(false)
  const isPolling = ref(false)
  const transcriptionStatus = ref<string>('')
  const interactionId = ref<string>('')
  const transcript = ref<TranscriptResult | null>(null)
  const error = ref<string>('')

  /**
   * Upload audio file and initiate transcription
   * @param file - The audio file to transcribe
   * @param language - Language code (default: 'en-US')
   * @param enableDiarization - Whether to enable speaker diarization (default: false)
   */
  const uploadAndTranscribe = async (
    file: File,
    language: string = 'en-US',
    enableDiarization: boolean = false
  ): Promise<void> => {
    isUploading.value = true
    error.value = ''
    transcript.value = null
    interactionId.value = ''
    transcriptionStatus.value = ''

    try {
      const formData = new FormData()
      formData.append('audio', file)
      formData.append('language', language)
      formData.append('diarization', enableDiarization ? 'true' : 'false')

      const response = await fetch(`${API_BASE}/transcribe/upload`, {
        method: 'POST',
        headers: getAuthHeadersForFormData(),
        body: formData,
      })

      // Handle session expiry (401) and demo-trial-limit (403)
      await checkDemoLimit(response)

      const data: UploadResponse = await response.json()

      if (!response.ok || !data.success) {
        throw new Error(data.message || 'Upload failed')
      }

      interactionId.value = data.interaction_id
      transcriptionStatus.value = data.status

      // Corti returned the full transcript synchronously — store it so the caller
      // can skip polling (avoids the 403-prone GetTranscriptByID call on EU region)
      if (data.transcript && data.transcript.text) {
        transcript.value = data.transcript
        transcriptionStatus.value = 'completed'
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Upload failed'
      throw err
    } finally {
      isUploading.value = false
    }
  }

  /**
   * Poll for transcript completion
   */
  const pollTranscript = async (
    id: string,
    maxAttempts: number = 60,
    intervalMs: number = 2000
  ): Promise<TranscriptResult | null> => {
    isPolling.value = true
    error.value = ''

    let attempts = 0
    let consecutiveErrors = 0
    const maxConsecutiveErrors = 3

    while (attempts < maxAttempts) {
      try {
        const response = await fetch(`${API_BASE}/transcribe/${id}`, {
          headers: getAuthHeaders(),
        })

        // Handle session expiry (401)
        handleFetchResponse(response)

        const data: TranscriptResponse = await response.json()

        if (!response.ok) {
          // Treat a backend error as transient — retry up to maxConsecutiveErrors times
          // before giving up, so a single 404/503 from Corti doesn't abort the whole poll
          consecutiveErrors++
          console.warn(`Transcript poll attempt ${attempts + 1} got ${response.status}:`, data)
          if (consecutiveErrors >= maxConsecutiveErrors) {
            throw new Error(data.error || 'Failed to get transcript')
          }
          await new Promise(resolve => setTimeout(resolve, intervalMs))
          attempts++
          continue
        }

        consecutiveErrors = 0
        transcriptionStatus.value = data.status

        if (data.is_complete && data.transcript) {
          transcript.value = data.transcript
          isPolling.value = false
          return data.transcript
        }

        if (data.status === 'failed') {
          throw new Error('Transcription failed')
        }

        // Wait before next poll
        await new Promise(resolve => setTimeout(resolve, intervalMs))
        attempts++
      } catch (err) {
        // Don't treat session expiry as polling error - it will redirect
        if (err instanceof Error && err.message === 'Session expired') {
          throw err
        }
        error.value = err instanceof Error ? err.message : 'Polling failed'
        isPolling.value = false
        throw err
      }
    }

    isPolling.value = false
    error.value = 'Transcription timed out'
    return null
  }

  /**
   * Get transcript by interaction ID (single request)
   */
  const getTranscript = async (id: string): Promise<TranscriptResult | null> => {
    error.value = ''

    try {
      const response = await fetch(`${API_BASE}/transcribe/${id}`, {
        headers: getAuthHeaders(),
      })
      
      // Handle session expiry (401)
      handleFetchResponse(response)
      
      const data: TranscriptResponse = await response.json()

      if (!response.ok) {
        throw new Error(data.error || 'Failed to get transcript')
      }

      transcriptionStatus.value = data.status

      if (data.transcript) {
        transcript.value = data.transcript
        return data.transcript
      }

      return null
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to get transcript'
      throw err
    }
  }

  /**
   * Reset the transcription state
   */
  const reset = () => {
    isUploading.value = false
    isPolling.value = false
    transcriptionStatus.value = ''
    interactionId.value = ''
    transcript.value = null
    error.value = ''
  }

  return {
    // State
    isUploading,
    isPolling,
    transcriptionStatus,
    interactionId,
    transcript,
    error,
    // Methods
    uploadAndTranscribe,
    pollTranscript,
    getTranscript,
    reset,
  }
}

