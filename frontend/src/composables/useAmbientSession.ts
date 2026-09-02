import { ref, onUnmounted } from 'vue'
import { getAuthHeaders, handleFetchResponse } from '@/stores/auth'

interface TranscriptSegment {
  id?: string
  text: string
  speaker?: string
  isFinal: boolean
  timestamp?: number
  startTime?: number
  endTime?: number
}

interface MedicalEvent {
  type: string
  category?: string
  value: string
  confidence?: number
}

interface Fact {
  id?: string
  category: string
  value: string
  text?: string
  confidence?: number
  source?: string
}

interface ServerMessage {
  type: string
  data?: any
  error?: string
  timestamp: number
}

interface StartSessionResponse {
  success: boolean
  interaction_id: string
  websocket_url: string
  status: string
  message?: string
}

const API_BASE = '/api'

export function useAmbientSession() {
  const isConnecting = ref(false)
  const isStreaming = ref(false)
  const connectionStatus = ref<'disconnected' | 'connecting' | 'connected'>('disconnected')
  const interactionId = ref<string>('')
  const transcript = ref<TranscriptSegment[]>([])
  const medicalEvents = ref<MedicalEvent[]>([])
  const facts = ref<Fact[]>([])
  const error = ref<string>('')

  let websocket: WebSocket | null = null
  let pingInterval: ReturnType<typeof setInterval> | null = null

  /**
   * Start a new ambient session
   */
  const startSession = async (language: string = 'en-US'): Promise<void> => {
    isConnecting.value = true
    connectionStatus.value = 'connecting'
    error.value = ''
    transcript.value = []
    medicalEvents.value = []
    facts.value = []

    try {
      // Create interaction via REST API
      const response = await fetch(`${API_BASE}/ambient/start`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({ language }),
      })

      // Handle session expiry (401)
      handleFetchResponse(response)

      const data: StartSessionResponse = await response.json()

      if (!response.ok || !data.success) {
        throw new Error(data.message || 'Failed to start session')
      }

      interactionId.value = data.interaction_id

      // Connect to WebSocket with language parameter
      const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const wsUrl = `${wsProtocol}//${window.location.host}/api/ambient/ws/${data.interaction_id}?language=${encodeURIComponent(language)}`

      console.log('Connecting to WebSocket:', wsUrl)
      await connectWebSocket(wsUrl)
    } catch (err) {
      connectionStatus.value = 'disconnected'
      error.value = err instanceof Error ? err.message : 'Failed to start session'
      throw err
    } finally {
      isConnecting.value = false
    }
  }

  /**
   * Connect to the WebSocket endpoint
   */
  const connectWebSocket = (url: string): Promise<void> => {
    return new Promise((resolve, reject) => {
      websocket = new WebSocket(url)

      websocket.onopen = () => {
        console.log('WebSocket connected')
        connectionStatus.value = 'connected'
        isStreaming.value = true
        startPingInterval()
        resolve()
      }

      websocket.onclose = (event) => {
        console.log('WebSocket closed:', event.code, event.reason)
        connectionStatus.value = 'disconnected'
        isStreaming.value = false
        stopPingInterval()
      }

      websocket.onerror = (event) => {
        console.error('WebSocket error:', event)
        error.value = 'WebSocket connection error'
        reject(new Error('WebSocket connection failed'))
      }

      websocket.onmessage = (event) => {
        handleMessage(event.data)
      }
    })
  }

  /**
   * Handle incoming WebSocket messages
   */
  const handleMessage = (data: string) => {
    try {
      const message: ServerMessage = JSON.parse(data)
      console.log('Received message:', message.type, message.data)

      switch (message.type) {
        case 'connected':
          console.log('Connection confirmed:', message.data)
          break

        case 'transcript':
          handleTranscriptEvent(message.data)
          break

        case 'fact':
          handleFactEvent(message.data)
          break

        case 'medical_event':
          handleMedicalEvent(message.data)
          break

        case 'pong':
          // Heartbeat response
          break

        case 'error':
          error.value = message.error || message.data?.error || 'Unknown error'
          console.error('Server error:', error.value)
          break

        case 'status':
          console.log('Status update:', message.data)
          break

        default:
          console.log('Unknown message type:', message.type, message.data)
      }
    } catch (err) {
      console.error('Failed to parse message:', err, data)
    }
  }

  /**
   * Handle transcript events from the stream
   * Reference: https://docs.corti.ai/api-reference/stream#transcripts-data-streams
   * Format: { type: "transcript", data: [{ id, transcript, final, speakerId, ... }] }
   */
  const handleTranscriptEvent = (data: any) => {
    if (!data) return

    // Corti sends transcripts as an array in data.data or data itself
    const transcriptsArray = data.data || (Array.isArray(data) ? data : [data])

    for (const item of transcriptsArray) {
      const segment: TranscriptSegment = {
        id: item.id || `seg-${Date.now()}`,
        text: item.transcript || item.text || '',
        speaker: item.speakerId !== undefined ? `speaker_${item.speakerId}` : item.speaker,
        isFinal: item.final || item.isFinal || item.is_final || false,
        timestamp: Date.now(),
        startTime: item.time?.start || item.startTime,
        endTime: item.time?.end || item.endTime,
      }

      if (!segment.text) continue

      console.log('Transcript segment:', segment.isFinal ? 'FINAL' : 'partial', segment.text.substring(0, 50))

      // If it's a partial result, update or add
      if (!segment.isFinal) {
        // Find existing partial and update, or add new
        const existingIndex = transcript.value.findIndex(s => !s.isFinal)
        if (existingIndex >= 0) {
          transcript.value[existingIndex] = segment
        } else {
          transcript.value.push(segment)
        }
      } else {
        // Remove any partial and add the final
        const partialIndex = transcript.value.findIndex(s => !s.isFinal)
        if (partialIndex >= 0) {
          transcript.value.splice(partialIndex, 1)
        }
        transcript.value.push(segment)
      }
    }
  }

  /**
   * Handle fact events from the stream
   * Reference: https://docs.corti.ai/api-reference/stream#facts-data-streams
   * Format: { type: "facts", fact: [{ id, text, group, groupId, ... }] }
   */
  const handleFactEvent = (data: any) => {
    if (!data) return

    console.log('Processing fact event:', data)

    // Extract facts array from various possible formats
    let factsArray: any[] = []
    
    // Try data.fact first (standard Corti format)
    if (data.fact && Array.isArray(data.fact)) {
      factsArray = data.fact
    }
    // Try data.facts as alternative
    else if (data.facts && Array.isArray(data.facts)) {
      factsArray = data.facts
    }
    // Try data.data if nested
    else if (data.data && Array.isArray(data.data)) {
      factsArray = data.data
    }
    // If data itself is an array
    else if (Array.isArray(data)) {
      factsArray = data
    }
    // If data is a single fact object
    else if (data.text || data.value) {
      factsArray = [data]
    }

    console.log(`Found ${factsArray.length} facts to process`)

    for (const factData of factsArray) {
      // Skip discarded facts early
      if (factData.isDiscarded) {
        console.log('Skipping discarded fact:', factData.id)
        continue
      }

      // Extract value with fallbacks
      const factValue = factData.text || factData.value || ''
      if (!factValue) {
        console.log('Skipping fact with no value')
        continue
      }

      const fact: Fact = {
        id: factData.id || `fact-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
        // Corti uses "group" instead of "category"
        category: factData.group || factData.category || 'general',
        value: factValue,
        text: factData.text,
        confidence: factData.confidence,
        source: factData.source,
      }

      // Avoid duplicates by ID or value+category
      const exists = facts.value.some(
        f => (f.id === fact.id) || (f.value === fact.value && f.category === fact.category)
      )

      if (!exists) {
        console.log('New fact detected:', fact.category, fact.value)
        facts.value.push(fact)

        // Also add as medical event for backward compatibility
        const event: MedicalEvent = {
          type: 'fact',
          category: fact.category,
          value: fact.value,
          confidence: fact.confidence,
        }
        medicalEvents.value.push(event)
      } else {
        console.log('Duplicate fact skipped:', fact.id, fact.value.substring(0, 30))
      }
    }

    console.log(`Total facts now: ${facts.value.length}`)
  }

  /**
   * Handle medical event detection
   */
  const handleMedicalEvent = (data: any) => {
    if (!data) return

    const event: MedicalEvent = {
      type: data.type || 'unknown',
      category: data.category,
      value: data.value || data.text || '',
      confidence: data.confidence,
    }

    // Avoid duplicates
    const exists = medicalEvents.value.some(
      e => e.value === event.value && e.category === event.category
    )

    if (!exists && event.value) {
      console.log('New medical event:', event.category, event.value)
      medicalEvents.value.push(event)
    }
  }

  /**
   * Send audio data to the WebSocket
   */
  const sendAudioData = (audioData: ArrayBuffer): void => {
    if (websocket?.readyState === WebSocket.OPEN) {
      websocket.send(audioData)
    }
  }

  /**
   * Send a control message
   */
  const sendControlMessage = (type: string, data?: any): void => {
    if (websocket?.readyState === WebSocket.OPEN) {
      websocket.send(JSON.stringify({ type, data }))
    }
  }

  /**
   * Start ping interval to keep connection alive
   */
  const startPingInterval = () => {
    stopPingInterval()
    pingInterval = setInterval(() => {
      sendControlMessage('ping')
    }, 30000) // Ping every 30 seconds
  }

  /**
   * Stop ping interval
   */
  const stopPingInterval = () => {
    if (pingInterval) {
      clearInterval(pingInterval)
      pingInterval = null
    }
  }

  /**
   * Stop the ambient session
   */
  const stopSession = (): void => {
    if (websocket) {
      sendControlMessage('stop')
      websocket.close()
      websocket = null
    }
    stopPingInterval()
    isStreaming.value = false
    connectionStatus.value = 'disconnected'
  }

  /**
   * Get full transcript text
   */
  const getFullTranscript = (): string => {
    return transcript.value
      .filter(s => s.isFinal)
      .map(s => s.text)
      .join(' ')
  }

  /**
   * Reset the session state
   */
  const reset = (): void => {
    stopSession()
    interactionId.value = ''
    transcript.value = []
    medicalEvents.value = []
    facts.value = []
    error.value = ''
  }

  // Cleanup on unmount
  onUnmounted(() => {
    stopSession()
  })

  return {
    // State
    isConnecting,
    isStreaming,
    connectionStatus,
    interactionId,
    transcript,
    medicalEvents,
    facts,
    error,
    // Methods
    startSession,
    stopSession,
    sendAudioData,
    sendControlMessage,
    getFullTranscript,
    reset,
  }
}
