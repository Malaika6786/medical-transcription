import { ref, onUnmounted } from 'vue'

/**
 * Dictation Composable
 * Reference: https://docs.corti.ai/quickstart/dictation
 * 
 * This provides real-time stateless dictation using the /transcribe WebSocket endpoint.
 * Unlike Ambient AI, this is simpler and doesn't require creating interactions.
 */

interface TranscriptSegment {
  text: string
  rawTranscriptText?: string
  start: number
  end: number
  isFinal: boolean
  timestamp?: number
}

interface CommandEvent {
  id: string
  variables?: Record<string, string>
  timestamp?: number
  executed?: boolean
}

// Command callback type for handling command events
type CommandCallback = (command: CommandEvent) => void

interface ServerMessage {
  type: string
  data?: any
  error?: string
  timestamp: number
}

export function useDictation() {
  const isConnecting = ref(false)
  const isStreaming = ref(false)
  const connectionStatus = ref<'disconnected' | 'connecting' | 'connected' | 'ready'>('disconnected')
  const transcript = ref<TranscriptSegment[]>([])
  const commands = ref<CommandEvent[]>([])
  const error = ref<string>('')
  const credits = ref<number>(0)

  let websocket: WebSocket | null = null
  let pingInterval: ReturnType<typeof setInterval> | null = null
  let commandCallbacks: CommandCallback[] = []

  /**
   * Start a new dictation session
   * Reference: https://docs.corti.ai/quickstart/dictation#initiate-real-time-bi-directional-communication
   */
  const startSession = async (language: string = 'en'): Promise<void> => {
    isConnecting.value = true
    connectionStatus.value = 'connecting'
    error.value = ''
    transcript.value = []
    commands.value = []
    credits.value = 0

    try {
      // Connect directly to WebSocket (no interaction needed for dictation)
      const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const wsUrl = `${wsProtocol}//${window.location.host}/api/dictation/ws?language=${encodeURIComponent(language)}`

      console.log('Connecting to Dictation WebSocket:', wsUrl)
      await connectWebSocket(wsUrl)
    } catch (err) {
      connectionStatus.value = 'disconnected'
      error.value = err instanceof Error ? err.message : 'Failed to start dictation session'
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
        console.log('Dictation WebSocket connected')
        connectionStatus.value = 'connected'
        startPingInterval()
        resolve()
      }

      websocket.onclose = (event) => {
        console.log('Dictation WebSocket closed:', event.code, event.reason)
        connectionStatus.value = 'disconnected'
        isStreaming.value = false
        stopPingInterval()
      }

      websocket.onerror = (event) => {
        console.error('Dictation WebSocket error:', event)
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
   * Reference: https://docs.corti.ai/quickstart/dictation#handle-responses
   */
  const handleMessage = (data: string) => {
    try {
      const message: ServerMessage = JSON.parse(data)
      console.log('Dictation message:', message.type, message.data)

      switch (message.type) {
        case 'connected':
          console.log('Connection confirmed:', message.data)
          break

        case 'status':
          handleStatusMessage(message.data)
          break

        case 'transcript':
          handleTranscriptEvent(message.data)
          break

        case 'command':
          handleCommandEvent(message.data)
          break

        case 'usage':
          if (message.data?.credits !== undefined) {
            credits.value = message.data.credits
            console.log('Usage credits:', credits.value)
          }
          break

        case 'pong':
          // Heartbeat response
          break

        case 'error':
          error.value = message.error || message.data?.error || 'Unknown error'
          console.error('Dictation error:', error.value)
          break

        default:
          console.log('Unknown message type:', message.type, message.data)
      }
    } catch (err) {
      console.error('Failed to parse message:', err, data)
    }
  }

  /**
   * Handle status messages
   */
  const handleStatusMessage = (data: any) => {
    if (!data) return

    const statusType = data.type || data.status

    switch (statusType) {
      case 'CONFIG_ACCEPTED':
        console.log('Config accepted - ready to stream')
        connectionStatus.value = 'ready'
        isStreaming.value = true
        break
      case 'flushed':
        console.log('Buffer flushed')
        break
      case 'ended':
        console.log('Session ended by server')
        connectionStatus.value = 'disconnected'
        isStreaming.value = false
        break
    }
  }

  /**
   * Handle transcript events
   * Reference: https://docs.corti.ai/quickstart/dictation#handle-responses
   */
  const handleTranscriptEvent = (data: any) => {
    if (!data) return

    const segment: TranscriptSegment = {
      text: data.text || '',
      rawTranscriptText: data.rawTranscriptText,
      start: data.start || 0,
      end: data.end || 0,
      isFinal: data.isFinal || false,
      timestamp: Date.now(),
    }

    if (!segment.text) return

    console.log('Transcript:', segment.isFinal ? 'FINAL' : 'partial', segment.text)

    if (!segment.isFinal) {
      // Update partial transcript
      const existingIndex = transcript.value.findIndex(s => !s.isFinal)
      if (existingIndex >= 0) {
        transcript.value[existingIndex] = segment
      } else {
        transcript.value.push(segment)
      }
    } else {
      // Remove partial and add final
      const partialIndex = transcript.value.findIndex(s => !s.isFinal)
      if (partialIndex >= 0) {
        transcript.value.splice(partialIndex, 1)
      }
      transcript.value.push(segment)
    }
  }

  /**
   * Handle command events
   * Reference: https://docs.corti.ai/stt/commands
   */
  const handleCommandEvent = (data: any) => {
    if (!data) return

    const command: CommandEvent = {
      id: data.id || 'unknown',
      variables: data.variables,
      timestamp: Date.now(),
      executed: false,
    }

    console.log('Command detected:', command.id, command.variables)
    commands.value.push(command)

    // Execute registered callbacks
    commandCallbacks.forEach(callback => {
      try {
        callback(command)
        command.executed = true
      } catch (err) {
        console.error('Command callback error:', err)
      }
    })

    // Handle built-in commands
    executeBuiltInCommand(command)
  }

  /**
   * Execute built-in commands
   */
  const executeBuiltInCommand = (command: CommandEvent) => {
    switch (command.id) {
      case 'clear_all':
        // Clear all transcripts
        transcript.value = []
        console.log('Executed: clear_all')
        break

      case 'stop_dictation':
        // Stop the dictation session
        stopSession()
        console.log('Executed: stop_dictation')
        break

      case 'undo':
        // Remove last transcript segment
        if (transcript.value.length > 0) {
          transcript.value.pop()
          console.log('Executed: undo')
        }
        break

      case 'new_line':
        // Add a line break to transcript
        transcript.value.push({
          text: '\n',
          start: 0,
          end: 0,
          isFinal: true,
          timestamp: Date.now(),
        })
        console.log('Executed: new_line')
        break

      case 'new_paragraph':
        // Add a paragraph break to transcript
        transcript.value.push({
          text: '\n\n',
          start: 0,
          end: 0,
          isFinal: true,
          timestamp: Date.now(),
        })
        console.log('Executed: new_paragraph')
        break

      case 'delete_range':
        handleDeleteCommand(command.variables?.delete_range)
        break

      case 'select_range':
        // Selection requires UI context - emit event for parent to handle
        console.log('Select command:', command.variables?.select_range)
        break

      case 'go_to_section':
        // Navigation requires UI context - emit event for parent to handle
        console.log('Navigate command:', command.variables?.section_key)
        break

      case 'insert_template':
        // Template insertion requires UI context - emit event for parent to handle
        console.log('Insert template command:', command.variables?.template_name)
        break

      default:
        console.log('Unhandled command:', command.id)
    }
  }

  /**
   * Handle delete command
   */
  const handleDeleteCommand = (range: string | undefined) => {
    if (!range) return

    const normalizedRange = range.toLowerCase()

    switch (normalizedRange) {
      case 'everything':
      case 'all':
        transcript.value = []
        console.log('Executed: delete everything')
        break

      case 'the last word':
      case 'last word':
        // Remove last word from last transcript
        if (transcript.value.length > 0) {
          const lastSegment = transcript.value[transcript.value.length - 1]
          const words = lastSegment.text.trim().split(/\s+/)
          if (words.length > 1) {
            words.pop()
            lastSegment.text = words.join(' ') + ' '
          } else {
            transcript.value.pop()
          }
          console.log('Executed: delete last word')
        }
        break

      case 'the last sentence':
      case 'last sentence':
        // Remove last sentence (segment)
        if (transcript.value.length > 0) {
          transcript.value.pop()
          console.log('Executed: delete last sentence')
        }
        break

      case 'that':
      case 'this':
        // Delete last segment
        if (transcript.value.length > 0) {
          transcript.value.pop()
          console.log('Executed: delete that')
        }
        break

      default:
        console.log('Unknown delete range:', range)
    }
  }

  /**
   * Register a callback for command events
   */
  const onCommand = (callback: CommandCallback) => {
    commandCallbacks.push(callback)
    // Return unsubscribe function
    return () => {
      const index = commandCallbacks.indexOf(callback)
      if (index > -1) {
        commandCallbacks.splice(index, 1)
      }
    }
  }

  /**
   * Send audio data to the WebSocket
   * Reference: https://docs.corti.ai/quickstart/dictation#stream-audio-and-receive-transcripts
   */
  const sendAudioData = (audioData: ArrayBuffer): void => {
    if (websocket?.readyState === WebSocket.OPEN) {
      websocket.send(audioData)
    }
  }

  /**
   * Flush the audio buffer
   * Reference: https://docs.corti.ai/quickstart/dictation#force-results-to-be-returned-from-server
   */
  const flush = (): void => {
    if (websocket?.readyState === WebSocket.OPEN) {
      websocket.send(JSON.stringify({ type: 'flush' }))
      console.log('Flush sent')
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
   * Start ping interval
   */
  const startPingInterval = () => {
    stopPingInterval()
    pingInterval = setInterval(() => {
      sendControlMessage('ping')
    }, 30000)
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
   * Stop the dictation session
   * Reference: https://docs.corti.ai/quickstart/dictation#sending-the-end-message
   */
  const stopSession = (): void => {
    if (websocket) {
      // Send end message before closing
      sendControlMessage('end')
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
    transcript.value = []
    commands.value = []
    error.value = ''
    credits.value = 0
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
    transcript,
    commands,
    error,
    credits,
    // Methods
    startSession,
    stopSession,
    sendAudioData,
    flush,
    sendControlMessage,
    getFullTranscript,
    reset,
    onCommand,
  }
}

