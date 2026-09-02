<template>
  <div class="embedded-assistant-page">
    <!-- Header -->
    <v-row>
      <v-col cols="12">
        <div class="d-flex align-center justify-space-between mb-4">
          <div class="d-flex align-center">
            <v-icon icon="mdi-robot-outline" class="mr-2" color="primary" size="24" />
            <h1 class="text-h6 font-weight-medium mb-0">Embedded Assistant</h1>
            <v-chip
              v-if="isAuthenticated"
              class="ml-3"
              color="success"
              size="small"
              variant="tonal"
            >
              <v-icon start icon="mdi-circle" size="8" />
              Connected
            </v-chip>
            <v-chip
              v-else-if="isReady"
              class="ml-3"
              color="warning"
              size="small"
              variant="tonal"
            >
              Authenticating...
            </v-chip>
            <v-chip
              v-else
              class="ml-3"
              color="grey"
              size="small"
              variant="tonal"
            >
              Loading...
            </v-chip>
          </div>

          <!-- Controls -->
          <div class="d-flex align-center ga-2">
            <v-btn
              v-if="!currentInteractionId && isAuthenticated"
              color="primary"
              variant="tonal"
              prepend-icon="mdi-plus"
              @click="encounterDialog = true"
            >
              New Encounter
            </v-btn>
            <v-btn
              v-if="currentInteractionId && !isRecording"
              color="success"
              variant="tonal"
              prepend-icon="mdi-microphone"
              :loading="recordingLoading"
              @click="handleStartRecording"
            >
              Start Recording
            </v-btn>
            <v-btn
              v-if="isRecording"
              color="error"
              variant="tonal"
              prepend-icon="mdi-stop"
              :loading="recordingLoading"
              @click="handleStopRecording"
            >
              Stop Recording
            </v-btn>
            <v-btn
              v-if="capturedSessionData && !sessionSaved"
              color="primary"
              variant="tonal"
              prepend-icon="mdi-content-save"
              @click="saveCurrentSession"
            >
              Save Session
            </v-btn>
            <v-chip
              v-if="sessionSaved"
              color="success"
              size="small"
              variant="tonal"
              prepend-icon="mdi-check"
            >
              Saved
            </v-chip>
          </div>
        </div>
      </v-col>
    </v-row>

    <!-- Error alert -->
    <v-row v-if="error">
      <v-col cols="12">
        <v-alert type="error" variant="tonal" closable @click:close="error = null">
          {{ error }}
        </v-alert>
      </v-col>
    </v-row>

    <!-- Embedded Assistant Web Component -->
    <v-row>
      <v-col cols="12">
        <v-card class="glass-card pa-0 overflow-hidden" style="height: calc(100vh - 220px); position: relative;">
          <corti-embedded
            ref="cortiElement"
            :baseurl="BASE_URL"
            visibility="visible"
            style="width: 100%; height: 100%; display: block;"
          />
        </v-card>
      </v-col>
    </v-row>

    <!-- New Encounter Dialog -->
    <v-dialog v-model="encounterDialog" max-width="480" persistent>
      <v-card class="glass-card">
        <v-card-title class="d-flex align-center pa-4">
          <v-icon icon="mdi-stethoscope" class="mr-2" color="primary" />
          New Encounter
        </v-card-title>
        <v-divider />
        <v-card-text class="pa-4">
          <v-select
            v-model="encounterType"
            :items="encounterTypes"
            item-title="label"
            item-value="value"
            label="Encounter Type"
            variant="outlined"
            class="mb-3"
          />
          <v-text-field
            v-model="encounterTitle"
            label="Title (optional)"
            variant="outlined"
            placeholder="e.g. GP Consultation"
          />
        </v-card-text>
        <v-divider />
        <v-card-actions class="pa-4">
          <v-spacer />
          <v-btn variant="text" @click="encounterDialog = false">Cancel</v-btn>
          <v-btn
            color="primary"
            variant="tonal"
            :loading="encounterLoading"
            :disabled="!encounterType"
            @click="handleCreateEncounter"
          >
            Start Encounter
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
    <!-- Save Session Dialog -->
    <v-dialog v-model="showSaveDialog" max-width="500">
      <v-card class="glass-card">
        <v-card-title class="text-h6">
          <v-icon icon="mdi-content-save" class="mr-2" color="primary" />
          Save Session
        </v-card-title>
        <v-card-text>
          <p class="text-body-2 text-medium-emphasis mb-4">
            Save this session for future review.
          </p>
          <v-text-field
            v-model="sessionTitle"
            label="Session Title"
            placeholder="Enter a title for this session"
            prepend-inner-icon="mdi-format-title"
            autofocus
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="showSaveDialog = false">Cancel</v-btn>
          <v-btn color="primary" :loading="isSaving" @click="confirmSaveSession">
            <v-icon icon="mdi-content-save" class="mr-1" />
            Save
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Save success snackbar -->
    <v-snackbar v-model="showSaveSnackbar" color="success" timeout="3000">
      Session saved successfully.
    </v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useCortiEmbedded } from '@/composables/useCortiEmbedded'
import { saveSession, loadSessions } from '@/stores/sessions'

const cortiElement = ref<HTMLElement | null>(null)

const {
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
} = useCortiEmbedded()

const encounterDialog = ref(false)
const encounterLoading = ref(false)
const recordingLoading = ref(false)
const encounterType = ref('first_consultation')
const encounterTitle = ref('')

const encounterTypes = [
  { label: 'First Consultation', value: 'first_consultation' },
  { label: 'Ambulatory', value: 'ambulatory' },
  { label: 'Virtual', value: 'virtual' },
  { label: 'Emergency', value: 'emergency' },
  { label: 'Inpatient', value: 'inpatient_encounter' },
  { label: 'Home Health', value: 'home_health' },
]

// ─── Session saving ────────────────────────────────────────────────────────

const showSaveDialog = ref(false)
const showSaveSnackbar = ref(false)
const sessionTitle = ref('')
const isSaving = ref(false)
const sessionSaved = ref(false)

function saveCurrentSession() {
  if (!capturedSessionData.value) return
  sessionTitle.value = capturedSessionData.value.title
  showSaveDialog.value = true
}

async function confirmSaveSession() {
  showSaveDialog.value = false
  await performSave()
}

async function performSave() {
  if (!capturedSessionData.value) return
  isSaving.value = true
  try {
    await saveSession({
      type: 'embedded-assistant',
      title: sessionTitle.value || capturedSessionData.value.title,
      transcript: capturedSessionData.value.transcript,
      document: capturedSessionData.value.document,
      interactionId: capturedSessionData.value.interactionId,
    })
    sessionSaved.value = true
    showSaveSnackbar.value = true
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err)
    error.value = `Failed to save session: ${msg}`
  } finally {
    isSaving.value = false
  }
}

onMounted(async () => {
  await loadSessions()
  if (cortiElement.value) {
    await initialize(cortiElement.value)
  }
})

async function handleCreateEncounter() {
  encounterLoading.value = true
  try {
    await createInteraction(encounterType.value, encounterTitle.value || undefined)
    encounterDialog.value = false
    encounterTitle.value = ''
    sessionSaved.value = false
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err)
    console.error('Failed to create encounter:', err)
    error.value = `Failed to create encounter: ${msg}`
  } finally {
    encounterLoading.value = false
  }
}

async function handleStartRecording() {
  recordingLoading.value = true
  try {
    await startRecording()
  } finally {
    recordingLoading.value = false
  }
}

async function handleStopRecording() {
  recordingLoading.value = true
  try {
    await stopRecording()
  } finally {
    recordingLoading.value = false
  }
}
</script>

<style scoped>
.embedded-assistant-page {
  height: 100%;
}
</style>
