<template>
  <div class="ambient-page">
    <v-row>
      <v-col cols="12">
        <div class="d-flex align-center mb-4">
          <v-icon icon="mdi-broadcast" class="mr-2" color="secondary" size="24" />
          <h1 class="text-h6 font-weight-medium mb-0">Ambient Session</h1>
        </div>
      </v-col>
    </v-row>

    <v-row>
      <!-- Control Panel -->
      <v-col cols="12" md="4">
        <v-card class="glass-card">
          <v-card-text>
            <!-- Connection Status -->
            <div class="mb-4">
              <span class="status-indicator" :class="connectionStatusClass">
                {{ connectionStatus }}
              </span>
            </div>

            <!-- Language Selection -->
            <v-select
              v-model="selectedLanguage"
              :items="languages"
              item-title="name"
              item-value="code"
              label="Language"
              :disabled="isStreaming"
              prepend-inner-icon="mdi-translate"
              class="mb-4"
              data-tour="language-select"
            />

            <!-- Microphone Selection -->
            <v-select
              v-model="selectedMicrophone"
              :items="microphones"
              item-title="label"
              item-value="deviceId"
              label="Microphone"
              :disabled="isStreaming"
              prepend-inner-icon="mdi-microphone"
              class="mb-4"
              data-tour="microphone-select"
            />

            <DemoUsageBanner
              feature="ambient"
              label="Ambient Recording"
              description="Records your voice live and transcribes it automatically."
            />

            <!-- Recording Controls -->
            <div class="d-flex flex-column ga-3">
              <v-btn
                v-if="!isStreaming"
                block
                size="large"
                color="primary"
                :loading="isConnecting"
                @click="handleStartRecording"
                data-tour="start-recording"
              >
                <v-icon icon="mdi-record" class="mr-2" />
                Start New Recording
              </v-btn>

              <v-btn
                v-else
                block
                size="large"
                color="error"
                class="recording-btn"
                @click="stopSession"
              >
                <div class="d-flex align-center">
                  <div class="waveform-container-inline mr-3">
                    <span v-for="i in 5" :key="i" class="waveform-bar-small" />
                  </div>
                  <span>Stop Recording</span>
                </div>
              </v-btn>
            </div>
          </v-card-text>
        </v-card>

        <!-- Document Generation (after session ends) -->
        <v-card v-if="sessionEnded && interactionId && transcript.length > 0" class="glass-card mt-4">
          <v-card-title class="text-h6">
            <v-icon icon="mdi-file-document-edit-outline" class="mr-2" />
            Generate Report
          </v-card-title>
          <v-card-text>
            <p class="text-body-2 text-medium-emphasis mb-3">
              Select a template to generate a report from the transcript:
            </p>
            <DemoUsageBanner
              feature="document_generation"
              label="Document Generation"
              description="Turn a transcript into a structured clinical document."
            />
            <ReportGenerator
              :templates="availableTemplates"
              v-model:selected-template="selectedTemplate"
              v-model:selected-verbosity="selectedVerbosity"
              v-model:patient-name="patientName"
              v-model:recipient-doctor="recipientDoctor"
              v-model:sender-doctor="senderDoctor"
              v-model:sender-title="senderTitle"
              v-model:soap-overrides="soapSectionOverrides"
              :is-generating="isGeneratingDocument"
              :show-verbosity-options="isSuperUser"
              @generate="generateDocument"
            />
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Transcript & Events -->
      <v-col cols="12" md="8">
        <v-expansion-panels v-model="expandedPanels" multiple class="mb-4">
          <!-- Live Transcript -->
          <v-expansion-panel value="transcript" data-tour="live-transcript">
            <v-expansion-panel-title>
              <div class="d-flex align-center justify-space-between w-100 pr-2">
                <span class="d-flex align-center">
                  <v-icon icon="mdi-text" class="mr-2" />
                  <span class="text-h6">Live Transcript</span>
                </span>
                <v-chip v-if="transcript.length" size="small" color="primary">
                  {{ transcript.length }} segments
                </v-chip>
              </div>
            </v-expansion-panel-title>
            <v-expansion-panel-text>
              <TranscriptDisplay
                :segments="transcript"
                :is-streaming="isStreaming"
                empty-title="No transcript yet"
                empty-subtitle="Start a session to begin transcribing"
              />
            </v-expansion-panel-text>
          </v-expansion-panel>

          <!-- Generated Report -->
          <v-expansion-panel v-if="generatedDocument" value="report">
            <v-expansion-panel-title>
              <div class="d-flex align-center">
                <v-icon icon="mdi-file-document-check" class="mr-2" color="success" />
                <span class="text-h6">Generated Report</span>
              </div>
            </v-expansion-panel-title>
            <v-expansion-panel-text>
              <GeneratedReportCard
                ref="reportCardRef"
                v-model="editableDocumentContent"
                :is-processing="false"
                :is-saving="isSaving"
                :show-save-button="true"
                :hide-title="true"
                :document-title="sessionTitle || 'Ambient Session'"
                :template-name="selectedTemplateName"
                @save="saveCurrentSession"
              />
            </v-expansion-panel-text>
          </v-expansion-panel>
        </v-expansion-panels>

        <!-- AI Assistance (after session ends, bottom of the live-transcript section) -->
        <v-btn
          v-if="sessionEnded && transcript.length > 0 && !showAiPanel"
          block
          size="large"
          color="secondary"
          variant="tonal"
          class="mb-4"
          prepend-icon="mdi-robot-outline"
          @click="showAiPanel = true"
        >
          AI Assistance
        </v-btn>

        <AiAssistantPanel
          v-if="showAiPanel"
          :transcript="fullTranscriptText"
          :language="selectedLanguage"
          allow-report
          :report-meta="aiReportMeta"
          class="mb-4"
          @summarized="handleAiSummarized"
        />

        <!-- Medical Facts & Events - Only visible to SuperUsers (outside expansion panels) -->
        <v-card v-if="canViewClinicalFacts" class="glass-card">
          <v-card-title class="d-flex justify-space-between align-center">
            <span>
              <v-icon icon="mdi-medical-bag" class="mr-2" />
              Clinical Facts
              <v-chip size="x-small" color="warning" variant="tonal" class="ml-2">Super User Only</v-chip>
            </span>
            <div class="d-flex align-center ga-2">
              <!-- Facts Status Indicator -->
              <v-chip 
                v-if="isStreaming" 
                size="small" 
                :color="factsStatusColor"
                variant="tonal"
              >
                <v-progress-circular 
                  v-if="factsStatus === 'processing'" 
                  indeterminate 
                  size="12" 
                  width="2" 
                  class="mr-1"
                />
                <v-icon 
                  v-else-if="factsStatus === 'ready'" 
                  icon="mdi-check-circle" 
                  size="14" 
                  class="mr-1"
                />
                <v-icon 
                  v-else 
                  icon="mdi-clock-outline" 
                  size="14" 
                  class="mr-1"
                />
                {{ factsStatusText }}
              </v-chip>
              <v-chip v-if="facts.length || medicalEvents.length" size="small" color="secondary">
                {{ facts.length || medicalEvents.length }} detected
              </v-chip>
            </div>
          </v-card-title>

          <v-card-text>
            <!-- Facts Processing Alert -->
            <v-alert
              v-if="isStreaming && factsStatus === 'waiting'"
              type="info"
              variant="tonal"
              density="compact"
              class="mb-3"
            >
              <template v-slot:prepend>
                <v-icon icon="mdi-information" />
              </template>
              <div class="text-body-2">
                <strong>Tip:</strong> Mention specific clinical details like medications (e.g., "Metformin 500mg"), 
                vital signs (e.g., "blood pressure 140 over 90"), allergies, and diagnoses for better fact extraction.
              </div>
            </v-alert>

            <template v-if="facts.length || medicalEvents.length">
              <v-row>
                <v-col
                  v-for="(fact, index) in (facts.length ? facts : medicalEvents)"
                  :key="index"
                  cols="12"
                  sm="6"
                >
                  <div
                    class="medical-event-card pa-3 rounded-lg bg-surface-variant"
                    :class="fact.category?.toLowerCase()"
                  >
                    <div class="d-flex align-center mb-2">
                      <v-icon
                        :icon="getEventIcon(fact.category)"
                        :color="getEventColor(fact.category)"
                        size="20"
                        class="mr-2"
                      />
                      <span class="text-caption font-weight-medium text-uppercase">
                        {{ fact.category || 'Event' }}
                      </span>
                    </div>
                    <p class="text-body-2 mb-0">{{ fact.value }}</p>
                  </div>
                </v-col>
              </v-row>
            </template>
            <template v-else>
              <div class="text-center py-8">
                <v-icon
                  icon="mdi-medical-bag"
                  size="48"
                  color="medium-emphasis"
                  class="mb-2"
                />
                <p class="text-body-2 text-medium-emphasis">
                  {{ isStreaming ? 'Listening for clinical facts...' : 'Clinical facts will appear here as they\'re detected' }}
                </p>
              </div>
            </template>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

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

    <!-- Start New Recording Confirmation Dialog -->
    <v-dialog v-model="showResetDialog" max-width="450">
      <v-card class="glass-card">
        <v-card-title class="text-h6">
          <v-icon icon="mdi-record" class="mr-2" color="warning" />
          Start New Recording?
        </v-card-title>
        <v-card-text>
          <p class="text-body-1">
            Starting a new recording will clear the current transcript and any generated reports.
          </p>
          <v-alert type="warning" variant="tonal" density="compact" class="mt-3">
            Make sure to save your session first if you want to keep it.
          </v-alert>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="showResetDialog = false">Cancel</v-btn>
          <v-btn color="primary" @click="confirmStartNewRecording">
            <v-icon icon="mdi-record" class="mr-1" />
            Start New Recording
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Unsaved Changes Dialog -->
    <v-dialog v-model="showUnsavedDialog" max-width="450" persistent>
      <v-card class="glass-card">
        <v-card-title class="d-flex align-center">
          <v-icon icon="mdi-alert-circle" color="warning" class="mr-2" />
          Unsaved Changes
        </v-card-title>
        <v-card-text>
          <p>You have unsaved changes that will be lost if you leave this page.</p>
          <p class="text-medium-emphasis mb-0">Would you like to stay and save your work, or leave without saving?</p>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="confirmLeave">Leave Without Saving</v-btn>
          <v-btn color="primary" @click="cancelLeave">
            <v-icon icon="mdi-content-save" class="mr-1" />
            Stay on Page
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Snackbars -->
    <v-snackbar v-model="showError" color="error" timeout="5000">
      {{ errorMessage }}
      <template v-slot:actions>
        <v-btn variant="text" @click="showError = false">Close</v-btn>
      </template>
    </v-snackbar>

    <v-snackbar v-model="showSuccess" color="success" timeout="3000">
      {{ successMessage }}
    </v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import TranscriptDisplay from '@/components/TranscriptDisplay.vue'
import ReportGenerator from '@/components/ReportGenerator.vue'
import GeneratedReportCard from '@/components/GeneratedReportCard.vue'
import AiAssistantPanel from '@/components/AiAssistantPanel.vue'
import DemoUsageBanner from '@/components/DemoUsageBanner.vue'
import { useAmbientSession } from '@/composables/useAmbientSession'
import { useAudioCapture } from '@/composables/useAudioCapture'
import { canViewClinicalFacts, currentUser, isSuperUser } from '@/stores/auth'
import { saveSession, getSession } from '@/stores/sessions'
import type { AiExtraction } from '@/composables/useAiAssistant'
import api from '@/services/api'

// Document interfaces
interface DocumentSection {
  key?: string
  name: string
  text?: string
  content?: string
}

interface DocumentResult {
  id: string
  name: string
  status: string
  sections: DocumentSection[]
}

interface DocumentTemplate {
  key: string
  name: string
  description: string
  category?: string
  isCustom?: boolean
}

const {
  isConnecting,
  isStreaming,
  connectionStatus,
  interactionId,
  transcript,
  medicalEvents,
  facts,
  error,
  startSession: startAmbientSession,
  stopSession: stopAmbientSession,
  sendAudioData,
} = useAmbientSession()

const {
  microphones,
  selectedMicrophone,
  initAudioCapture,
  startCapture,
  stopCapture,
} = useAudioCapture()

const selectedLanguage = ref('en-GB')
const sessionStartTime = ref<Date | null>(null)
const showError = ref(false)
const showSuccess = ref(false)
const errorMessage = ref('')
const successMessage = ref('')

// Expansion panels state - both expanded by default
const expandedPanels = ref<string[]>(['transcript', 'report'])

// AI Assistance panel state
const showAiPanel = ref(false)
// Latest extraction from the AI panel — persisted on the session record and
// included in manual saves so they don't wipe it.
const latestExtraction = ref<AiExtraction | null>(null)

// Full transcript with speaker labels — the only handoff to the AI module.
const fullTranscriptText = computed(() =>
  transcript.value
    .filter(s => s.isFinal)
    .map(s => (s.speaker ? `${s.speaker.replace('speaker_', 'Speaker ')}: ${s.text}` : s.text))
    .join('\n')
)

// Session save state
const currentSessionId = ref<string | null>(null)
const isSaving = ref(false)
const showSaveDialog = ref(false)
const sessionTitle = ref('')
const showResetDialog = ref(false)
const lastSavedContent = ref<string>('')
const showUnsavedDialog = ref(false)
const pendingNavigation = ref<(() => void) | null>(null)

// Document generation state
const sessionEnded = ref(false)
const selectedTemplate = ref<string>('corti-soap')
const selectedVerbosity = ref<string>('concise')
const availableTemplates = ref<DocumentTemplate[]>([])
const generatedDocument = ref<DocumentResult | null>(null)
const isGeneratingDocument = ref(false)
const transcriptId = ref<string>('')

// Edit/Preview mode for generated document
const isEditingDocument = ref(false)
const editableDocumentContent = ref('')

// Report card ref
const reportCardRef = ref<InstanceType<typeof GeneratedReportCard> | null>(null)

// GP Letter specific options
const patientName = ref('')
const recipientDoctor = ref('')
const senderDoctor = ref('')
const senderTitle = ref('General Practitioner')

// SOAP Section Overrides - Editable by user
// Default values based on verbosity level
const soapSectionOverrides = ref({
  subjective: {
    writingStyle: '',
    formatRule: '',
    additionalInstructions: ''
  },
  objective: {
    writingStyle: '',
    formatRule: '',
    additionalInstructions: ''
  },
  assessment: {
    writingStyle: '',
    formatRule: '',
    additionalInstructions: ''
  },
  plan: {
    writingStyle: '',
    formatRule: '',
    additionalInstructions: ''
  }
})

// Default override presets based on verbosity
const verbosityPresets = {
  concise: {
    subjective: {
      writingStyle: 'Concise summary in third person, focus on key clinical findings only',
      formatRule: '',
      additionalInstructions: ''
    },
    objective: {
      writingStyle: 'Brief objective findings, essential data only',
      formatRule: '',
      additionalInstructions: ''
    },
    assessment: {
      writingStyle: 'Concise clinical assessment with primary diagnosis',
      formatRule: '',
      additionalInstructions: ''
    },
    plan: {
      writingStyle: 'Brief management plan with key action items',
      formatRule: '',
      additionalInstructions: ''
    }
  },
  standard: {
    subjective: {
      writingStyle: 'Clear narrative in third person, include relevant history and context. Use bullet points for each key finding.',
      formatRule: 'Use bullet points for multiple findings',
      additionalInstructions: 'Include relevant history and context. Use bullet points for multiple findings.'
    },
    objective: {
      writingStyle: 'Complete examination findings with relevant diagnostic results. Use bullet points.',
      formatRule: 'Use bullet points for findings',
      additionalInstructions: 'Include all significant findings. Use bullet points if multiple findings.'
    },
    assessment: {
      writingStyle: 'Clinical assessment with reasoning and differential considerations',
      formatRule: '',
      additionalInstructions: 'Include primary diagnosis and key reasoning'
    },
    plan: {
      writingStyle: 'Detailed management plan with specific instructions. Use bullet points for each action item.',
      formatRule: 'Use bullet points for action items',
      additionalInstructions: 'Include specific action items with details. Use bullet points for each action.'
    }
  },
  detailed: {
    subjective: {
      writingStyle: 'Write in third person using bullet points. Each bullet should capture one distinct clinical finding.',
      formatRule: 'Use bullet points starting with "- " for all findings',
      additionalInstructions: `IMPORTANT: Format as bullet points, each starting with "- ". Include:
- Reason for visit/chief complaint
- Symptom characteristics (what, when, where, how long)
- Changes since last visit if follow-up
- Impact on activities or quality of life
- Associated symptoms or factors
- Patient's concerns or questions
Do NOT use paragraphs. Each distinct finding should be a separate bullet.`
    },
    objective: {
      writingStyle: 'Write using bullet points for each finding. Include examination findings, vital signs, and any diagnostic results.',
      formatRule: 'Use bullet points starting with "- " for all findings. Use subheadings where relevant.',
      additionalInstructions: `Format as bullet points starting with "- ". If there is past medical history, use a subheading "Past Medical History:" followed by bullet points. Include:
- Examination findings (what was observed/measured)
- Vital signs if mentioned
- Any test results or measurements
Keep each bullet focused on one finding.`
    },
    assessment: {
      writingStyle: 'State the diagnosis clearly. If multiple diagnoses, use bullet points.',
      formatRule: 'Use bullet points for multiple diagnoses',
      additionalInstructions: `State the diagnosis clearly. If multiple diagnoses, use bullet points. Include:
- Primary diagnosis/clinical impression
- Any secondary diagnoses
- Status of condition (new, ongoing, resolved)`
    },
    plan: {
      writingStyle: 'Write using bullet points for each action item. Each bullet should be ONE complete action or recommendation.',
      formatRule: 'Use bullet points starting with "- " for all action items',
      additionalInstructions: `IMPORTANT: Format as bullet points, each starting with "- ". Each bullet should be ONE complete action or recommendation. Include:
- Counselling and education provided to patient
- Treatment recommendations with specific details
- Medications with dose, frequency, and duration if mentioned
- Procedures or referrals recommended
- Follow-up plans and timeline
- Safety netting advice (when to return, warning signs)
- Lifestyle recommendations
Be specific about what was discussed or recommended.`
    }
  }
}

// Apply verbosity preset to section overrides
const applyVerbosityPreset = () => {
  const preset = verbosityPresets[selectedVerbosity.value as keyof typeof verbosityPresets]
  if (preset) {
    soapSectionOverrides.value = JSON.parse(JSON.stringify(preset))
  }
}

// Watch for verbosity changes and apply preset
watch(selectedVerbosity, () => {
  applyVerbosityPreset()
}, { immediate: true })

const languages = [
  { name: 'English (US)', code: 'en-US' },
  { name: 'English (UK)', code: 'en-GB' },
  { name: 'Spanish', code: 'es-ES' },
  { name: 'German', code: 'de-DE' },
  { name: 'French', code: 'fr-FR' },
]

const connectionStatusClass = computed(() => {
  switch (connectionStatus.value) {
    case 'connected':
      return 'connected'
    case 'connecting':
      return 'recording'
    case 'disconnected':
    default:
      return 'disconnected'
  }
})

const selectedTemplateName = computed(() => {
  const template = availableTemplates.value.find(t => t.key === selectedTemplate.value)
  return template?.name || 'Document'
})

// Facts status tracking
const factsStatus = computed(() => {
  if (!isStreaming.value) return 'idle'
  if (facts.value.length > 0 || medicalEvents.value.length > 0) return 'ready'
  if (transcript.value.length > 3) return 'processing' // After some transcript segments, we expect facts
  return 'waiting'
})

const factsStatusText = computed(() => {
  switch (factsStatus.value) {
    case 'ready': return 'Facts detected'
    case 'processing': return 'Extracting facts...'
    case 'waiting': return 'Waiting for clinical content'
    default: return ''
  }
})

const factsStatusColor = computed(() => {
  switch (factsStatus.value) {
    case 'ready': return 'success'
    case 'processing': return 'warning'
    case 'waiting': return 'info'
    default: return 'grey'
  }
})

const startSession = async () => {
  try {
    // Reset state for new recording
    sessionEnded.value = false
    generatedDocument.value = null
    transcriptId.value = ''
    showAiPanel.value = false
    latestExtraction.value = null
    
    // Reset session ID so a new save creates a new session, not overwrite
    currentSessionId.value = null
    sessionTitle.value = ''

    // Start ambient session (creates WebSocket connection)
    await startAmbientSession(selectedLanguage.value)
    
    // Initialize and start audio capture
    await initAudioCapture(selectedMicrophone.value)
    await startCapture((audioData: ArrayBuffer) => {
      sendAudioData(audioData)
    })
    
    sessionStartTime.value = new Date()
  } catch (err) {
    showError.value = true
    errorMessage.value = error.value || 'Failed to start session'
  }
}

const stopSession = () => {
  stopCapture()
  stopAmbientSession()
  sessionStartTime.value = null
  sessionEnded.value = true
}

// Save current session
const saveCurrentSession = () => {
  if (!transcript.value.length) {
    showError.value = true
    errorMessage.value = 'No transcript to save'
    return
  }
  
  // Generate default title
  const date = new Date()
  sessionTitle.value = `Ambient Session - ${date.toLocaleDateString()} ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`
  showSaveDialog.value = true
}

// Confirm save session
const confirmSaveSession = async () => {
  if (!sessionTitle.value.trim()) {
    showError.value = true
    errorMessage.value = 'Please enter a session title'
    return
  }
  
  await performSave()
}

// Actual save logic
const performSave = async () => {
  isSaving.value = true
  
  try {
    // Combine transcript into plain text (as per meeting requirement #7)
    const transcriptText = transcript.value
      .map(s => s.text)
      .join(' ')
    
    // Prepare document if generated - include the edited HTML content
    const documentData = generatedDocument.value ? {
      id: generatedDocument.value.id || '',
      templateKey: selectedTemplate.value,
      templateName: selectedTemplateName.value,
      sections: generatedDocument.value.sections || [],
      htmlContent: editableDocumentContent.value || undefined, // Save the edited HTML content
      // Store letter details for letter templates (for regeneration)
      letterDetails: (patientName.value || recipientDoctor.value || senderDoctor.value) ? {
        patientName: patientName.value || undefined,
        recipientDoctor: recipientDoctor.value || undefined,
        senderDoctor: senderDoctor.value || undefined,
        senderTitle: senderTitle.value || undefined
      } : undefined,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    } : undefined
    
    // Save session
    await saveSession({
      id: currentSessionId.value || undefined,
      type: 'ambient',
      title: sessionTitle.value.trim(),
      transcript: transcriptText,
      document: documentData,
      extraction: latestExtraction.value || undefined,
      interactionId: interactionId.value
    })
    
    showSuccess.value = true
    successMessage.value = 'Session saved successfully!'
    showSaveDialog.value = false
    
    // Reset the page after successful save
    resetPage()
  } catch (err: any) {
    showError.value = true
    errorMessage.value = err.message || 'Failed to save session'
  } finally {
    isSaving.value = false
  }
}

// Header metadata for the exported AI report. None of it is stored specially:
// the patient name is whatever was already typed for the letter template, and
// blank fields are omitted from the report header rather than left dangling.
const aiReportMeta = computed(() => ({
  title: sessionTitle.value.trim() || 'Ambient Consultation Report',
  patientName: patientName.value,
  clinician: currentUser.value?.name,
  consultationDate: sessionStartTime.value,
  sessionId: currentSessionId.value,
}))

// AI extraction landed — auto-save the session so the extraction (and
// transcript) survives a page reload. Silent on failure: the extraction
// stays available in memory and the manual save path still works.
const handleAiSummarized = async (extraction: AiExtraction) => {
  latestExtraction.value = extraction
  if (!transcript.value.length) return

  const transcriptText = transcript.value
    .map(s => s.text)
    .join(' ')

  const existing = currentSessionId.value ? getSession(currentSessionId.value) : undefined
  const date = new Date()
  const fallbackTitle = extraction.chiefComplaint
    ? `Ambient — ${extraction.chiefComplaint}`
    : `Ambient Session - ${date.toLocaleDateString()} ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`

  try {
    const saved = await saveSession({
      id: currentSessionId.value || undefined,
      type: 'ambient',
      title: existing?.title || sessionTitle.value.trim() || fallbackTitle,
      transcript: transcriptText,
      extraction,
      interactionId: interactionId.value
    })
    currentSessionId.value = saved.id
  } catch (err) {
    console.warn('AI extraction auto-save failed (record remains in memory):', err)
  }
}

// Reset page to initial state
const resetPage = () => {
  // Stop any ongoing capture/session
  stopCapture()
  stopAmbientSession()
  
  // Clear transcript and facts
  transcript.value = []
  facts.value = []
  medicalEvents.value = []
  
  // Reset document state
  sessionEnded.value = false
  generatedDocument.value = null
  showAiPanel.value = false
  latestExtraction.value = null
  currentSessionId.value = null
  sessionStartTime.value = null
  sessionTitle.value = ''
  editableDocumentContent.value = ''
  lastSavedContent.value = ''
  
  // Reset panels to default
  expandedPanels.value = ['transcript', 'report']
}

// Handle start recording button click
const handleStartRecording = () => {
  // If there's existing content, show confirmation dialog
  if (transcript.value.length > 0 || generatedDocument.value) {
    showResetDialog.value = true
  } else {
    // No existing content, start directly
    startSession()
  }
}


// Confirm and start new recording (clears existing content first)
const confirmStartNewRecording = async () => {
  // Reset the page
  resetPage()
  
  showResetDialog.value = false
  
  // Now start the new recording
  await startSession()
}

// Automatically generate SOAP note when session ends
// Fetch available templates
const fetchTemplates = async () => {
  try {
    const response = await api.get('/templates')
    if (response.data.success) {
      availableTemplates.value = response.data.templates
      if (availableTemplates.value.length > 0) {
        selectedTemplate.value = availableTemplates.value[0].key
      }
    }
  } catch (err) {
    console.error('Failed to fetch templates:', err)
  }
}

// Generate document from session
const generateDocument = async () => {
  if (!interactionId.value || !selectedTemplate.value) {
    showError.value = true
    errorMessage.value = 'Missing interaction ID or template'
    return
  }

  isGeneratingDocument.value = true
  generatedDocument.value = null
  // Reset editing state for new document
  isEditingDocument.value = false
  editableDocumentContent.value = ''

  try {
    // Build request body with verbosity option
    const requestBody: { 
      templateKey: string; 
      transcriptId?: string; 
      verbosity?: string; 
      sectionOverrides?: typeof soapSectionOverrides.value;
      patientName?: string; 
      recipientDoctor?: string;
      senderDoctor?: string;
      senderTitle?: string;
    } = {
      templateKey: selectedTemplate.value,
      transcriptId: transcriptId.value || undefined,
    }
    
    // For GP/letter templates, include patient and doctor names
    const selectedTemplateObj = availableTemplates.value.find(t => t.key === selectedTemplate.value)
    const isLetterTemplate = selectedTemplate.value === 'gp-letter-with-summary' || selectedTemplateObj?.category === 'letter'
    if (isLetterTemplate) {
      if (patientName.value) requestBody.patientName = patientName.value
      if (recipientDoctor.value) requestBody.recipientDoctor = recipientDoctor.value
      if (senderDoctor.value) requestBody.senderDoctor = senderDoctor.value
      if (senderTitle.value) requestBody.senderTitle = senderTitle.value
    } else if (selectedTemplate.value === 'corti-soap') {
      // For SOAP Note, include verbosity and section overrides
      requestBody.verbosity = selectedVerbosity.value
      requestBody.sectionOverrides = soapSectionOverrides.value
    }
    // For other templates, just send the templateKey (no verbosity options)

    const response = await api.post(`/transcribe/${interactionId.value}/document`, requestBody)

    const docId = response.data.document_id
    if (docId) {
      // Poll for document completion
      await pollDocument(interactionId.value, docId)
      showSuccess.value = true
      successMessage.value = 'Document generated successfully!'
    } else {
      throw new Error('Document ID not received')
    }
  } catch (err: any) {
    console.error('Failed to generate document:', err)
    showError.value = true
    errorMessage.value = err.response?.data?.error || 'Failed to generate document'
  } finally {
    isGeneratingDocument.value = false
  }
}

// Poll for document completion
const pollDocument = async (intId: string, docId: string) => {
  let status = 'processing'
  let attempts = 0
  const maxAttempts = 30

  while ((status === 'processing' || status === 'pending') && attempts < maxAttempts) {
    attempts++
    try {
      const response = await api.get(`/transcribe/${intId}/document/${docId}`)
      status = response.data.document?.status || 'processing'
      
      if (status === 'completed') {
        generatedDocument.value = response.data.document
        break
      } else if (status === 'failed') {
        throw new Error('Document generation failed')
      }
    } catch (err) {
      console.error('Failed to poll document status:', err)
      break
    }
    await new Promise(resolve => setTimeout(resolve, 2000))
  }
}

// Convert sections to HTML for the rich text editor
const sectionsToHtml = (sections: Array<{ name: string; text?: string; content?: string }>) => {
  return sections
    .map(s => {
      const content = (s.text || s.content || '').replace(/\n/g, '<br>')
      return `<h2>${s.name}</h2><p>${content}</p>`
    })
    .join('')
}

// Watch for new document generation and populate editor content
watch(generatedDocument, (newDoc) => {
  if (newDoc?.sections) {
    editableDocumentContent.value = sectionsToHtml(newDoc.sections)
    // Auto-collapse transcript and expand report when document is generated
    expandedPanels.value = ['report']
  }
}, { immediate: true })

const getEventIcon = (category?: string): string => {
  const icons: Record<string, string> = {
    diagnosis: 'mdi-clipboard-pulse',
    medication: 'mdi-pill',
    symptom: 'mdi-alert-circle',
    procedure: 'mdi-medical-bag',
    allergy: 'mdi-alert-octagon',
    vital: 'mdi-heart-pulse',
  }
  return icons[category?.toLowerCase() || ''] || 'mdi-information'
}

const getEventColor = (category?: string): string => {
  const colors: Record<string, string> = {
    diagnosis: '#FF6B6B',
    medication: '#4ECDC4',
    symptom: '#FFE66D',
    procedure: '#95E1D3',
    allergy: '#FF8E8E',
    vital: '#FF6B6B',
  }
  return colors[category?.toLowerCase() || ''] || '#7C5CFF'
}

// Check if there are unsaved changes
const hasUnsavedChanges = computed(() => {
  // No changes if nothing has been recorded or generated
  if (transcript.value.length === 0 && !generatedDocument.value) {
    return false
  }
  
  // If currently recording, there are unsaved changes
  if (isStreaming.value) {
    return true
  }
  
  // Check if content has changed since last save
  const currentContent = JSON.stringify({
    transcript: transcript.value,
    document: editableDocumentContent.value || generatedDocument.value
  })
  
  return currentContent !== lastSavedContent.value
})

// Handle browser refresh/close
const handleBeforeUnload = (e: BeforeUnloadEvent) => {
  if (hasUnsavedChanges.value) {
    e.preventDefault()
    e.returnValue = 'You have unsaved changes. Are you sure you want to leave?'
    return e.returnValue
  }
}

// Handle route navigation
onBeforeRouteLeave((_to, _from, next) => {
  if (hasUnsavedChanges.value) {
    showUnsavedDialog.value = true
    pendingNavigation.value = () => next()
  } else {
    next()
  }
})

// Confirm leaving without saving
const confirmLeave = () => {
  showUnsavedDialog.value = false
  if (pendingNavigation.value) {
    // Remove beforeunload listener temporarily
    window.removeEventListener('beforeunload', handleBeforeUnload)
    pendingNavigation.value()
    pendingNavigation.value = null
  }
}

// Cancel leaving
const cancelLeave = () => {
  showUnsavedDialog.value = false
  pendingNavigation.value = null
}

// Update lastSavedContent when session is saved
watch(() => currentSessionId.value, () => {
  if (currentSessionId.value) {
    lastSavedContent.value = JSON.stringify({
      transcript: transcript.value,
      document: editableDocumentContent.value || generatedDocument.value
    })
  }
})

onMounted(async () => {
  // Add beforeunload listener
  window.addEventListener('beforeunload', handleBeforeUnload)
  
  // Fetch available templates
  await fetchTemplates()

  // Request microphone permissions and list devices
  try {
    await navigator.mediaDevices.getUserMedia({ audio: true })
    const devices = await navigator.mediaDevices.enumerateDevices()
    const audioInputs = devices
      .filter(d => d.kind === 'audioinput')
      .map(d => ({
        deviceId: d.deviceId,
        label: d.label || `Microphone ${d.deviceId.slice(0, 5)}`,
      }))
    
    if (audioInputs.length) {
      microphones.value = audioInputs
      selectedMicrophone.value = audioInputs[0].deviceId
    }
  } catch (err) {
    console.error('Failed to get microphones:', err)
  }
})

onUnmounted(() => {
  // Remove beforeunload listener
  window.removeEventListener('beforeunload', handleBeforeUnload)
  
  if (isStreaming.value) {
    stopSession()
  }
})
</script>

<style scoped>
.ambient-page {
  min-height: calc(100vh - 120px);
}

.waveform-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 48px;
}

/* Inline waveform for recording button */
.waveform-container-inline {
  display: flex;
  align-items: center;
  gap: 3px;
  height: 20px;
}

.waveform-bar-small {
  width: 3px;
  height: 100%;
  background: currentColor;
  border-radius: 2px;
  animation: waveform-small 0.8s ease-in-out infinite;
}

.waveform-bar-small:nth-child(1) { animation-delay: 0s; }
.waveform-bar-small:nth-child(2) { animation-delay: 0.1s; }
.waveform-bar-small:nth-child(3) { animation-delay: 0.2s; }
.waveform-bar-small:nth-child(4) { animation-delay: 0.3s; }
.waveform-bar-small:nth-child(5) { animation-delay: 0.4s; }

@keyframes waveform-small {
  0%, 100% { transform: scaleY(0.4); }
  50% { transform: scaleY(1); }
}

.recording-btn {
  position: relative;
}

/* Mobile responsiveness */
@media (max-width: 600px) {
  .ambient-page h1 {
    font-size: 1.5rem !important;
  }
}
</style>

