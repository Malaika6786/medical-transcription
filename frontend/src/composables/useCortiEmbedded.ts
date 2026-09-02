import { ref, onUnmounted } from 'vue'
import '@corti/embedded-web'
import type { CortiEmbeddedAPI } from '@corti/embedded-web'
import api from '@/services/api'
import type { SavedDocument } from '@/stores/sessions'

const BASE_URL = 'https://assistant.us.corti.app'
const READY_TIMEOUT_MS = 30_000

export function useCortiEmbedded() {
  const isReady = ref(false)
  const isAuthenticated = ref(false)
  const isRecording = ref(false)
  const currentInteractionId = ref<string | null>(null)
  const error = ref<string | null>(null)

  interface CapturedSessionData {
    interactionId: string
    title: string
    encounterType: string
    transcript: string
    document: SavedDocument
  }
  const capturedSessionData = ref<CapturedSessionData | null>(null)

  let cortiApi: CortiEmbeddedAPI | null = null
  let element: HTMLElement | null = null
  let currentEncounterType = ''

  // ─── Token fetch ───────────────────────────────────────────────────────────

  async function fetchTokens() {
    const resp = await api.get('/embedded/token')
    return resp.data as {
      access_token: string
      refresh_token: string
      id_token: string
      token_type: string
    }
  }

  // ─── Init ──────────────────────────────────────────────────────────────────

  async function initialize(el: HTMLElement) {
    element = el
    // The custom element itself exposes the CortiEmbeddedAPI methods directly
    cortiApi = el as unknown as CortiEmbeddedAPI

    el.addEventListener('error', handleError as EventListener)
    el.addEventListener('event', handleEmbeddedEvent as EventListener)

    // Wait for the web component to signal it is ready
    await new Promise<void>((resolve, reject) => {
      const timeout = window.setTimeout(() => {
        reject(new Error(`Embedded Assistant did not initialize within ${READY_TIMEOUT_MS / 1000}s. Check browser console/network for ${BASE_URL}.`))
      }, READY_TIMEOUT_MS)

      el.addEventListener('embedded.ready', () => {
        window.clearTimeout(timeout)
        console.info('[EmbeddedAssistant] embedded.ready received')
        resolve()
      }, { once: true })
    })

    isReady.value = true

    // Use the canonical "event" stream — the iframe sends camelCase names
    // (recordingStarted, documentGenerated) which the bundle re-dispatches
    // under the raw name AND under the "event" stream. Listening here handles
    // both the current camelCase names and future kebab-case names.
    await authenticate()
    await configure()
    await cortiApi.navigate('/')
  }

  async function authenticate() {
    if (!cortiApi) return
    try {
      const tokens = await fetchTokens()
      await cortiApi.auth({
        access_token: tokens.access_token,
        refresh_token: tokens.refresh_token,
        id_token: tokens.id_token,
        token_type: 'Bearer',
      })
      isAuthenticated.value = true
      error.value = null
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Authentication failed'
      console.error('[EmbeddedAssistant] auth() failed:', err)
      error.value = msg
      throw err
    }
  }

  async function configure() {
    if (!cortiApi) return
    await cortiApi.configureApp({
      ui: {
        navigation: false,
        documentFeedback: false,
        aiChat: true,
        interactionTitle: true,
      },
      appearance: { primaryColor: '#1565C0' },
      locale: { interfaceLanguage: 'en', dictationLanguage: 'en' },
    })
  }

  // ─── Canonical event stream handler ───────────────────────────────────────

  function handleEmbeddedEvent(e: Event) {
    const { name } = (e as CustomEvent<{ name: string }>).detail ?? {}
    console.info('[EmbeddedAssistant] event', (e as CustomEvent).detail)
    if (!name) return
    if (name === 'recordingStarted' || name === 'recording-started') {
      isRecording.value = true
    } else if (name === 'recordingStopped' || name === 'recording-stopped') {
      isRecording.value = false
    } else if (
      name === 'documentGenerated' || name === 'document-generated' ||
      name === 'documentUpdated' || name === 'document-updated'
    ) {
      handleDocumentGenerated()
    }
  }

  // ─── Error recovery ────────────────────────────────────────────────────────

  function handleError(event: CustomEvent) {
    const { code, message } = event.detail ?? {}
    console.error('[EmbeddedAssistant] error event', event.detail)
    if (code === 'UNAUTHORIZED') {
      // Silent re-auth — re-fetch tokens and re-authenticate
      authenticate().catch(() => {
        error.value = 'Session expired. Please reload the page.'
      })
      return
    }
    error.value = message ?? 'Embedded Assistant error'
  }

  async function handleDocumentGenerated() {
    if (!cortiApi) return
    try {
      const status = await cortiApi.getStatus()
      const interaction = status.interaction
      if (!interaction) return

      const transcript = interaction.transcripts
        .flatMap((t) => t.utterances)
        .filter((u) => u.isFinal)
        .map((u) => u.text)
        .join(' ')

      const doc = interaction.documents[0]
      if (!doc) return

      const now = new Date().toISOString()
      capturedSessionData.value = {
        interactionId: interaction.id,
        title: interaction.title ?? `Embedded Session ${new Date().toLocaleDateString()}`,
        encounterType: currentEncounterType,
        transcript,
        document: {
          id: doc.id,
          templateKey: doc.templateRef ?? doc.customTemplateId ?? '',
          templateName: doc.name,
          sections: doc.sections.map((s) => ({
            key: s.key,
            name: s.name,
            text: s.text,
            content: s.htmlText ?? s.plainText,
          })),
          createdAt: now,
          updatedAt: now,
        },
      }
    } catch (err) {
      console.error('[EmbeddedAssistant] Failed to capture session data:', err)
    }
  }

  function clearSessionData() {
    capturedSessionData.value = null
  }

  // ─── Public API wrappers ───────────────────────────────────────────────────

  async function createInteraction(encounterType: string, encounterTitle?: string) {
    if (!cortiApi) throw new Error('Embedded Assistant not ready')

    currentEncounterType = encounterType
    capturedSessionData.value = null

    const payload: Parameters<CortiEmbeddedAPI['createInteraction']>[0] = {
      assignedUserId: null,
      encounter: {
        identifier: `encounter-${Date.now()}`,
        status: 'planned',
        type: encounterType as Parameters<CortiEmbeddedAPI['createInteraction']>[0]['encounter']['type'],
        period: { startedAt: new Date().toISOString() },
      },
    }
    if (encounterTitle) {
      payload.encounter.title = encounterTitle
    }
    const interaction = await cortiApi.createInteraction(payload)

    currentInteractionId.value = interaction.id
    await cortiApi.navigate(`/session/${interaction.id}`)
    return interaction
  }

  async function startRecording() {
    if (!cortiApi) return
    await cortiApi.startRecording()
  }

  async function stopRecording() {
    if (!cortiApi) return
    await cortiApi.stopRecording()
  }

  async function getStatus() {
    if (!cortiApi) return null
    return cortiApi.getStatus()
  }

  // ─── Cleanup ───────────────────────────────────────────────────────────────

  onUnmounted(() => {
    if (element) {
      element.removeEventListener('error', handleError as EventListener)
      element.removeEventListener('event', handleEmbeddedEvent as EventListener)
    }
    isReady.value = false
    isAuthenticated.value = false
    isRecording.value = false
    currentInteractionId.value = null
    capturedSessionData.value = null
    cortiApi = null
    element = null
  })

  return {
    BASE_URL,
    isReady,
    isAuthenticated,
    isRecording,
    currentInteractionId,
    capturedSessionData,
    error,
    initialize,
    createInteraction,
    startRecording,
    stopRecording,
    getStatus,
    clearSessionData,
  }
}
