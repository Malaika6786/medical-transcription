import { ref, onUnmounted } from 'vue'

interface MicrophoneDevice {
  deviceId: string
  label: string
}

// Audio configuration for Corti API
// Reference: https://docs.corti.ai/api-reference/stream#sending-audio-data
// "For bandwidth and efficiency reasons, utilizing the webm/opus encoding is recommended"

export function useAudioCapture() {
  const microphones = ref<MicrophoneDevice[]>([])
  const selectedMicrophone = ref<string>('')
  const isCapturing = ref(false)
  const error = ref<string>('')

  let mediaStream: MediaStream | null = null
  let mediaRecorder: MediaRecorder | null = null
  let audioCallback: ((data: ArrayBuffer) => void) | null = null

  /**
   * List available microphones
   */
  const listMicrophones = async (): Promise<MicrophoneDevice[]> => {
    try {
      // Request permission first
      await navigator.mediaDevices.getUserMedia({ audio: true })
      
      const devices = await navigator.mediaDevices.enumerateDevices()
      const audioInputs = devices
        .filter(d => d.kind === 'audioinput')
        .map(d => ({
          deviceId: d.deviceId,
          label: d.label || `Microphone ${d.deviceId.slice(0, 8)}`,
        }))

      microphones.value = audioInputs
      
      if (audioInputs.length && !selectedMicrophone.value) {
        selectedMicrophone.value = audioInputs[0].deviceId
      }

      return audioInputs
    } catch (err) {
      error.value = 'Failed to access microphones'
      console.error('Microphone access error:', err)
      return []
    }
  }

  /**
   * Get supported MIME type for MediaRecorder
   * Corti recommends webm/opus, fallback to other formats
   */
  const getSupportedMimeType = (): string => {
    const mimeTypes = [
      'audio/webm;codecs=opus',
      'audio/webm',
      'audio/ogg;codecs=opus',
      'audio/ogg',
      'audio/mp4',
      'audio/mpeg',
    ]

    for (const mimeType of mimeTypes) {
      if (MediaRecorder.isTypeSupported(mimeType)) {
        console.log('Using MIME type:', mimeType)
        return mimeType
      }
    }

    // Default fallback
    console.warn('No preferred MIME type supported, using default')
    return ''
  }

  /**
   * Initialize audio capture with selected microphone
   */
  const initAudioCapture = async (deviceId?: string): Promise<void> => {
    try {
      // Close any existing capture
      await stopCapture()

      const constraints: MediaStreamConstraints = {
        audio: {
          deviceId: deviceId ? { exact: deviceId } : undefined,
          echoCancellation: { ideal: true },
          noiseSuppression: { ideal: true },
          autoGainControl: { ideal: true },
        },
      }

      mediaStream = await navigator.mediaDevices.getUserMedia(constraints)

      console.log('Audio capture initialized:', {
        deviceId: deviceId || 'default',
        tracks: mediaStream.getAudioTracks().map(t => t.label),
      })
    } catch (err) {
      error.value = 'Failed to initialize audio capture'
      console.error('Audio init error:', err)
      throw err
    }
  }

  /**
   * Start capturing audio using MediaRecorder (webm/opus format)
   * Reference: https://docs.corti.ai/api-reference/stream#sending-audio-data
   * "We recommend sending audio in chunks of 500ms"
   */
  const startCapture = async (
    onAudioData: (data: ArrayBuffer) => void
  ): Promise<void> => {
    if (!mediaStream) {
      throw new Error('Audio capture not initialized')
    }

    audioCallback = onAudioData
    isCapturing.value = true

    const mimeType = getSupportedMimeType()
    
    const options: MediaRecorderOptions = {
      mimeType: mimeType || undefined,
      audioBitsPerSecond: 128000, // 128 kbps for good quality
    }

    mediaRecorder = new MediaRecorder(mediaStream, options)

    mediaRecorder.ondataavailable = async (event) => {
      if (event.data.size > 0 && audioCallback) {
        const arrayBuffer = await event.data.arrayBuffer()
        console.log(`Sending audio chunk: ${arrayBuffer.byteLength} bytes`)
        audioCallback(arrayBuffer)
      }
    }

    mediaRecorder.onerror = (event) => {
      console.error('MediaRecorder error:', event)
      error.value = 'Audio recording error'
    }

    mediaRecorder.onstop = () => {
      console.log('MediaRecorder stopped')
      isCapturing.value = false
    }

    // Start recording with 500ms chunks (as recommended by Corti)
    // "We recommend sending audio in chunks of 500ms"
    mediaRecorder.start(500)

    console.log('MediaRecorder started:', {
      mimeType: mediaRecorder.mimeType,
      state: mediaRecorder.state,
      chunkInterval: '500ms',
    })
  }

  /**
   * Stop audio capture
   */
  const stopCapture = async (): Promise<void> => {
    isCapturing.value = false
    audioCallback = null

    if (mediaRecorder && mediaRecorder.state !== 'inactive') {
      mediaRecorder.stop()
      mediaRecorder = null
    }

    if (mediaStream) {
      mediaStream.getTracks().forEach(track => track.stop())
      mediaStream = null
    }

    console.log('Audio capture stopped')
  }

  /**
   * Request data flush from MediaRecorder
   * Call this before sending a "flush" message to Corti
   * Reference: https://docs.corti.ai/api-reference/stream#flush-the-audio-buffer
   */
  const requestData = (): void => {
    if (mediaRecorder && mediaRecorder.state === 'recording') {
      mediaRecorder.requestData()
    }
  }

  // Cleanup on unmount
  onUnmounted(() => {
    stopCapture()
  })

  return {
    // State
    microphones,
    selectedMicrophone,
    isCapturing,
    error,
    // Methods
    listMicrophones,
    initAudioCapture,
    startCapture,
    stopCapture,
    requestData,
  }
}
