<template>
  <div class="async-page">
    <v-row>
      <v-col cols="12">
        <div class="d-flex align-center mb-4">
          <v-icon icon="mdi-file-upload" class="mr-2" color="primary" size="24" />
          <h1 class="text-h6 font-weight-medium mb-0">File Transcription</h1>
        </div>
      </v-col>
    </v-row>

    <v-row>
      <!-- Upload Section -->
      <v-col cols="12" md="5">
        <v-card class="glass-card">
          <v-card-title class="text-h6">
            <v-icon icon="mdi-cloud-upload" class="mr-2" />
            Upload Audio
          </v-card-title>
          
          <v-card-text>
            <DemoUsageBanner
              feature="file_transcription"
              label="File Transcription"
              description="Upload an audio file and get a transcript."
            />

            <div data-tour="file-upload">
              <UploadAudio
                @file-selected="handleFileSelected"
                @upload-complete="handleUploadComplete"
                @error="handleError"
                :loading="isUploading"
              />
            </div>

            <v-select
              v-model="selectedLanguage"
              :items="languages"
              item-title="name"
              item-value="code"
              label="Language"
              class="mt-4"
              prepend-inner-icon="mdi-translate"
              data-tour="upload-language"
            />

            <!-- Speaker Diarization Toggle -->
            <v-card variant="outlined" class="mt-4 pa-3">
              <div class="d-flex align-center justify-space-between">
                <div>
                  <div class="d-flex align-center">
                    <v-icon icon="mdi-account-voice" class="mr-2" color="secondary" />
                    <span class="text-subtitle-2 font-weight-medium">Speaker Diarization</span>
                  </div>
                  <p class="text-caption text-medium-emphasis mt-1 mb-0">
                    Identify and label different speakers in the audio
                  </p>
                </div>
                <v-switch
                  v-model="enableDiarization"
                  color="secondary"
                  hide-details
                  density="compact"
                />
              </div>
              <v-alert
                v-if="enableDiarization"
                type="info"
                variant="tonal"
                density="compact"
                class="mt-3"
              >
                <span class="text-caption">
                  <strong>Tip:</strong> Works best with clear audio and distinct voices. 
                  The AI will label speakers as Speaker 0, Speaker 1, etc.
                </span>
              </v-alert>
            </v-card>

            <v-btn
              block
              size="large"
              color="primary"
              class="mt-4"
              :loading="isUploading"
              :disabled="!selectedFile"
              @click="uploadFile"
              data-tour="upload-button"
            >
              <v-icon icon="mdi-upload" class="mr-2" />
              {{ enableDiarization ? 'Start Transcription with Diarization' : 'Start Transcription' }}
            </v-btn>
          </v-card-text>
        </v-card>

        <!-- Status Card -->
        <v-card v-if="transcriptionStatus" class="glass-card mt-4">
          <v-card-title class="text-h6">
            <v-icon icon="mdi-information" class="mr-2" />
            Status
          </v-card-title>
          <v-card-text>
            <div class="d-flex align-center mb-2">
              <span class="status-indicator" :class="statusClass">
                {{ transcriptionStatus }}
              </span>
            </div>

            <!-- Show diarization badge if enabled -->
            <v-chip
              v-if="enableDiarization"
              size="small"
              color="secondary"
              variant="tonal"
              class="mb-2"
            >
              <v-icon icon="mdi-account-voice" size="14" class="mr-1" />
              Speaker Diarization Enabled
            </v-chip>
            
            <template v-if="interactionId">
              <v-text-field
                :model-value="interactionId"
                label="Interaction ID"
                readonly
                density="compact"
                class="mt-2"
                append-inner-icon="mdi-content-copy"
                @click:append-inner="copyInteractionId"
              />
            </template>

            <v-progress-linear
              v-if="isPolling"
              indeterminate
              :color="enableDiarization ? 'secondary' : 'primary'"
              class="mt-4"
            />
            <p v-if="isPolling && enableDiarization" class="text-caption text-medium-emphasis mt-2 mb-0">
              <v-icon icon="mdi-account-multiple" size="14" class="mr-1" />
              Identifying speakers in audio...
            </p>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Results Section -->
      <v-col cols="12" md="7">
        <template v-if="transcript">
          <v-expansion-panels v-model="expandedPanels" multiple>
            <!-- Transcript Panel -->
            <v-expansion-panel value="transcript">
              <v-expansion-panel-title>
                <div class="d-flex align-center justify-space-between w-100 pr-2">
                  <span class="d-flex align-center">
                    <v-icon icon="mdi-text" class="mr-2" />
                    <span class="text-h6">Transcript</span>
                  </span>
                  <div class="d-flex ga-1" @click.stop>
                  </div>
                </div>
              </v-expansion-panel-title>
              <v-expansion-panel-text>
                <TranscriptDisplay
                  :text="transcript.text"
                  :is-streaming="false"
                  empty-title="No transcript yet"
                  empty-subtitle="Upload an audio file to get started"
                />

                <!-- Document Generation Section -->
                <v-divider class="my-4" />
                <div class="document-generation-section">
                  <div class="d-flex align-center justify-space-between mb-3">
                    <div class="d-flex align-center">
                      <v-icon icon="mdi-file-document-edit" class="mr-2" color="primary" />
                      <span class="text-h6">Generate Report</span>
                    </div>
                  </div>

                  <DemoUsageBanner
                    feature="document_generation"
                    label="Document Generation"
                    description="Turn a transcript into a structured clinical document."
                  />

                  <ReportGenerator
                    :templates="documentTemplates"
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
                </div>
              </v-expansion-panel-text>
            </v-expansion-panel>

            <!-- Generated Report Panel -->
            <v-expansion-panel v-if="generatedDocument" value="report">
              <v-expansion-panel-title>
                <div class="d-flex align-center justify-space-between w-100 pr-2">
                  <span class="d-flex align-center">
                    <v-icon icon="mdi-file-document-check" class="mr-2" color="success" />
                    <span class="text-h6">Generated Report</span>
                  </span>
                  <div class="d-flex ga-1" @click.stop>
                    <v-menu>
                      <template v-slot:activator="{ props }">
                        <v-btn
                          variant="tonal"
                          size="small"
                          color="success"
                          v-bind="props"
                        >
                          <v-icon icon="mdi-download" class="mr-1" />
                          Export
                          <v-icon icon="mdi-chevron-down" size="small" class="ml-1" />
                        </v-btn>
                      </template>
                      <v-list density="compact">
                        <v-list-item @click="exportDocument('txt')">
                          <template v-slot:prepend>
                            <v-icon icon="mdi-file-document-outline" size="small" />
                          </template>
                          <v-list-item-title>Download as TXT</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="exportDocument('html')">
                          <template v-slot:prepend>
                            <v-icon icon="mdi-language-html5" size="small" />
                          </template>
                          <v-list-item-title>Download as HTML</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="printDocument">
                          <template v-slot:prepend>
                            <v-icon icon="mdi-printer" size="small" />
                          </template>
                          <v-list-item-title>Print / Save as PDF</v-list-item-title>
                        </v-list-item>
                      </v-list>
                    </v-menu>
                  </div>
                </div>
              </v-expansion-panel-title>
              <v-expansion-panel-text>
                <GeneratedReportCard
                  ref="reportCardRef"
                  v-model="editableDocumentContent"
                  :is-processing="generatedDocument.status === 'processing'"
                  :is-saving="isSaving"
                  :show-save-button="true"
                  :hide-title="true"
                  :document-title="sessionTitle || 'File Transcription'"
                  :template-name="selectedTemplateDisplayName"
                  @save="saveCurrentSession"
                />
              </v-expansion-panel-text>
            </v-expansion-panel>
          </v-expansion-panels>
        </template>

        <v-card v-else class="glass-card h-100">
          <v-card-text>
            <div class="text-center py-12">
              <v-icon
                icon="mdi-file-document-outline"
                size="80"
                color="medium-emphasis"
                class="mb-4"
              />
              <p class="text-h6 text-medium-emphasis">
                No transcript yet
              </p>
              <p class="text-body-2 text-medium-emphasis">
                Upload an audio file to get started
              </p>
            </div>
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

    <!-- Error Snackbar -->
    <v-snackbar
      v-model="showError"
      color="error"
      timeout="5000"
    >
      {{ errorMessage }}
      <template v-slot:actions>
        <v-btn variant="text" @click="showError = false">Close</v-btn>
      </template>
    </v-snackbar>

    <!-- Success Snackbar -->
    <v-snackbar
      v-model="showSuccess"
      color="success"
      timeout="3000"
    >
      {{ successMessage }}
    </v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import UploadAudio from '@/components/UploadAudio.vue'
import TranscriptDisplay from '@/components/TranscriptDisplay.vue'
import ReportGenerator from '@/components/ReportGenerator.vue'
import GeneratedReportCard from '@/components/GeneratedReportCard.vue'
import DemoUsageBanner from '@/components/DemoUsageBanner.vue'
import { useTranscription } from '@/composables/useTranscription'
import { isSuperUser, getAuthHeaders, handleFetchResponse, checkDemoLimit } from '@/stores/auth'
import { saveSession } from '@/stores/sessions'

// Backend response format from Corti API
interface BackendTranscriptEntry {
  text: string
  start: number       // Start time in milliseconds
  end: number         // End time in milliseconds
  speakerId: number   // Speaker ID (-1 if no diarization)
  channel: number
  participant: number
}

interface TranscriptResult {
  text: string
  status: string
  segments?: BackendTranscriptEntry[]
}

interface DocumentSection {
  key?: string
  name: string
  text?: string
  content?: string
  sort?: number
}

interface GeneratedDocument {
  id?: string
  status: string
  templateKey?: string
  sections?: DocumentSection[]
}

const {
  uploadAndTranscribe,
  pollTranscript,
  isUploading,
  isPolling,
  transcriptionStatus,
  interactionId,
  transcript,
  error
} = useTranscription()

const selectedFile = ref<File | null>(null)
const selectedLanguage = ref('en-GB')
const enableDiarization = ref(false)
const showError = ref(false)
const showSuccess = ref(false)
const errorMessage = ref('')
const successMessage = ref('')

// Expansion panels state - both expanded by default
const expandedPanels = ref<string[]>(['transcript', 'report'])

// Document generation state
const selectedTemplate = ref('corti-soap')
const selectedTemplateDisplayName = computed(() =>
  documentTemplates.find(t => t.key === selectedTemplate.value)?.name || 'Clinical Document'
)
const selectedVerbosity = ref('concise')
const isGeneratingDocument = ref(false)
const generatedDocument = ref<GeneratedDocument | null>(null)

// Session save state
const showSaveDialog = ref(false)
const sessionTitle = ref('')
const isSaving = ref(false)
const currentSessionId = ref<string | null>(null)
const lastSavedContent = ref<string>('')
const showUnsavedDialog = ref(false)
const pendingNavigation = ref<(() => void) | null>(null)

const languages = [
  { name: 'English (US)', code: 'en-US' },
  { name: 'English (UK)', code: 'en-GB' },
  { name: 'Spanish', code: 'es-ES' },
  { name: 'German', code: 'de-DE' },
  { name: 'French', code: 'fr-FR' },
  { name: 'Italian', code: 'it-IT' },
  { name: 'Portuguese', code: 'pt-BR' },
  { name: 'Dutch', code: 'nl-NL' },
  { name: 'Danish', code: 'da-DK' },
  { name: 'Swedish', code: 'sv-SE' },
  { name: 'Norwegian', code: 'nb-NO' },
]

const documentTemplates = [
  { key: 'corti-soap', name: 'SOAP Note', description: 'Subjective, Objective, Assessment, Plan' },
  { key: 'gp-letter-with-summary', name: 'GP Letter with Summary', description: 'Formal GP referral letter with detailed clinical summary', isCustom: true, category: 'letter' },
  { key: 'corti-brief-clinical-note', name: 'Brief Clinical Note', description: 'Quick clinical summary' },
  { key: 'corti-h-and-p', name: 'History & Physical', description: 'Initial patient evaluation' },
  { key: 'corti-outpatient-visit-note', name: 'Outpatient Visit Note', description: 'Office visit documentation' },
  { key: 'corti-emergency-note', name: 'Emergency Note', description: 'Emergency department documentation' },
  { key: 'corti-nursing-note', name: 'Nursing Note', description: 'Nursing assessment and care' },
  { key: 'corti-emergency-response-note', name: 'Emergency Response Note', description: 'EMS/First responder documentation' },
  { key: 'corti-referral', name: 'Referral', description: 'Specialist referral letter' },
  { key: 'corti-patient-summary', name: 'Patient Summary', description: 'Patient overview' },
]

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

// Editable document content
const editableDocumentContent = ref('')

// Report card ref
const reportCardRef = ref<InstanceType<typeof GeneratedReportCard> | null>(null)

const statusClass = computed(() => {
  switch (transcriptionStatus.value) {
    case 'completed':
      return 'connected'
    case 'processing':
    case 'pending':
      return 'recording'
    case 'failed':
      return 'disconnected'
    default:
      return ''
  }
})

const handleFileSelected = (file: File) => {
  selectedFile.value = file
}

const handleUploadComplete = () => {
  showSuccess.value = true
  successMessage.value = 'Transcription complete!'
}

const handleError = (message: string) => {
  showError.value = true
  errorMessage.value = message
}

const uploadFile = async () => {
  if (!selectedFile.value) return
  await performUpload()
}

const performUpload = async () => {
  if (!selectedFile.value) return

  try {
    await uploadAndTranscribe(selectedFile.value, selectedLanguage.value, enableDiarization.value)

    if (interactionId.value) {
      // If the transcript was returned inline in the upload response, skip polling
      if (transcript.value) {
        showSuccess.value = true
        successMessage.value = enableDiarization.value
          ? 'Transcription with speaker diarization complete!'
          : 'Transcription complete!'
      } else {
        await pollTranscript(interactionId.value)
        showSuccess.value = true
        successMessage.value = enableDiarization.value
          ? 'Transcription with speaker diarization complete!'
          : 'Transcription complete!'
      }
    }
  } catch (err) {
    handleError(error.value || 'Upload failed')
  }
}


const copyInteractionId = async () => {
  if (interactionId.value) {
    await navigator.clipboard.writeText(interactionId.value)
    showSuccess.value = true
    successMessage.value = 'Interaction ID copied!'
  }
}

const generateDocument = async () => {
  if (!interactionId.value || !selectedTemplate.value) return

  isGeneratingDocument.value = true
  generatedDocument.value = null

  try {
    // Generate document with verbosity option
    const requestBody: { 
      templateKey: string; 
      verbosity?: string; 
      sectionOverrides?: typeof soapSectionOverrides.value;
      patientName?: string; 
      recipientDoctor?: string;
      senderDoctor?: string;
      senderTitle?: string;
    } = { 
      templateKey: selectedTemplate.value 
    }
    
    // For GP/letter templates, include patient and doctor names
    const selectedTemplateObj = documentTemplates.find(t => t.key === selectedTemplate.value)
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

    const generateResponse = await fetch(`/api/transcribe/${interactionId.value}/document`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify(requestBody)
    })

    // Handle session expiry (401) and demo-trial-limit (403)
    await checkDemoLimit(generateResponse)

    if (!generateResponse.ok) {
      throw new Error('Failed to generate document')
    }

    const generateResult = await generateResponse.json()
    
    if (generateResult.document_id) {
      // Poll for document completion
      let attempts = 0
      const maxAttempts = 30
      
      while (attempts < maxAttempts) {
        const docResponse = await fetch(
          `/api/transcribe/${interactionId.value}/document/${generateResult.document_id}`,
          { headers: getAuthHeaders() }
        )
        
        // Handle session expiry (401)
        handleFetchResponse(docResponse)
        
        if (docResponse.ok) {
          const docResult = await docResponse.json()
          
          if (docResult.document && docResult.document.sections) {
            generatedDocument.value = docResult.document
            showSuccess.value = true
            successMessage.value = 'Document generated successfully!'
            break
          } else if (docResult.document?.status === 'failed') {
            throw new Error('Document generation failed')
          }
        }
        
        // Wait 2 seconds before next poll
        await new Promise(resolve => setTimeout(resolve, 2000))
        attempts++
      }
      
      if (attempts >= maxAttempts) {
        generatedDocument.value = { status: 'processing' }
      }
    } else {
      // Document was generated synchronously
      generatedDocument.value = generateResult.document || { status: 'completed', sections: [] }
    }
  } catch (err) {
    handleError('Failed to generate document')
  } finally {
    isGeneratingDocument.value = false
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

// Convert HTML to plain text for copying
const htmlToPlainText = (html: string) => {
  const div = document.createElement('div')
  div.innerHTML = html
  // Replace h1, h2, h3 with text and newlines
  div.querySelectorAll('h1, h2, h3').forEach(el => {
    el.textContent = `${el.textContent}\n\n`
  })
  // Replace br and p with newlines
  div.querySelectorAll('br').forEach(el => {
    el.replaceWith('\n')
  })
  div.querySelectorAll('p').forEach(el => {
    el.textContent = `${el.textContent}\n\n`
  })
  return div.textContent?.trim() || ''
}

// Watch for new document generation and populate editor content
watch(generatedDocument, (newDoc) => {
  if (newDoc?.sections) {
    editableDocumentContent.value = sectionsToHtml(newDoc.sections)
  }
}, { immediate: true })

// Helper function to download file
const downloadFile = (content: string, filename: string, mimeType: string) => {
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

// Generate timestamp for filenames
const getTimestamp = () => {
  const now = new Date()
  return now.toISOString().replace(/[:.]/g, '-').slice(0, 19)
}

// Export document functions
const exportDocument = (format: 'txt' | 'html') => {
  if (!editableDocumentContent.value) return
  
  const timestamp = getTimestamp()
  const templateName = documentTemplates.find(t => t.key === selectedTemplate.value)?.name || 'Report'
  
  if (format === 'txt') {
    // Convert HTML content to plain text
    const content = htmlToPlainText(editableDocumentContent.value)
    
    downloadFile(content, `${templateName.toLowerCase().replace(/\s+/g, '-')}-${timestamp}.txt`, 'text/plain')
    showSuccess.value = true
    successMessage.value = 'Report downloaded as TXT!'
  } else {
    // Use the HTML content directly
    const formattedContent = editableDocumentContent.value
    
    let content = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>${templateName} - ${timestamp}</title>
  <style>
    body { 
      font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
      max-width: 800px; 
      margin: 0 auto; 
      padding: 40px 20px; 
      line-height: 1.8; 
      color: #333; 
    }
    .letter-content { 
      white-space: pre-wrap; 
      font-size: 14px;
    }
    .letter-content strong {
      display: block;
      margin-top: 20px;
      margin-bottom: 5px;
      color: #1976d2;
    }
    .footer { 
      margin-top: 40px; 
      padding-top: 20px; 
      border-top: 1px solid #eee; 
      color: #888; 
      font-size: 0.85em; 
    }
    @media print { 
      body { padding: 20px; } 
      .footer { display: none; }
    }
  </style>
</head>
<body>
  <div class="letter-content">${formattedContent}</div>
  <div class="footer">
    <p>Generated by <strong>XStek AI Medical Transcription</strong> | ${new Date().toLocaleString()}</p>
  </div>
</body>
</html>`
    
    downloadFile(content, `${templateName.toLowerCase().replace(/\s+/g, '-')}-${timestamp}.html`, 'text/html')
    showSuccess.value = true
    successMessage.value = 'Document downloaded as HTML!'
  }
}

const printDocument = () => {
  if (!generatedDocument.value?.sections) return
  
  const templateName = documentTemplates.find(t => t.key === selectedTemplate.value)?.name || 'Document'
  
  const printWindow = window.open('', '_blank')
  if (!printWindow) {
    handleError('Please allow popups to print')
    return
  }
  
  let content = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>${templateName} - Print</title>
  <style>
    body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; max-width: 800px; margin: 0 auto; padding: 40px 20px; line-height: 1.6; color: #333; }
    h1 { color: #2e7d32; border-bottom: 3px solid #2e7d32; padding-bottom: 10px; margin-bottom: 10px; }
    .meta { color: #666; font-size: 0.9em; margin-bottom: 30px; padding-bottom: 15px; border-bottom: 1px solid #eee; }
    .section { margin-bottom: 25px; }
    .section-title { font-size: 1.1em; font-weight: bold; color: #1976d2; margin-bottom: 10px; padding: 8px 12px; background: #e3f2fd; border-radius: 4px; }
    .section-content { padding: 0 12px; white-space: pre-wrap; }
    .footer { margin-top: 40px; padding-top: 20px; border-top: 2px solid #eee; color: #888; font-size: 0.85em; }
    @media print { .section { page-break-inside: avoid; } }
  </style>
</head>
<body>
  <h1>🏥 ${templateName}</h1>
  <div class="meta">
    <strong>Generated:</strong> ${new Date().toLocaleString()}<br>
    <strong>Template:</strong> ${templateName}<br>
    <strong>Interaction ID:</strong> ${interactionId.value || 'N/A'}
  </div>`
  
  generatedDocument.value.sections.forEach((section) => {
    content += `
  <div class="section">
    <div class="section-title">${section.name}</div>
    <div class="section-content">${section.text || section.content || 'No content'}</div>
  </div>`
  })
  
  content += `
  <div class="footer">
    <p>Generated by <strong>XStek AI Medical Transcription</strong></p>
    <p>This document was automatically generated from audio transcription.</p>
  </div>
</body>
</html>`
  
  printWindow.document.write(content)
  printWindow.document.close()
  printWindow.focus()
  setTimeout(() => printWindow.print(), 250)
}

// Save session functionality
const saveCurrentSession = () => {
  if (!transcript.value || !generatedDocument.value) {
    showError.value = true
    errorMessage.value = 'No transcript or report to save'
    return
  }
  
  // Generate default title based on current date/time
  const now = new Date()
  sessionTitle.value = `File Transcription - ${now.toLocaleDateString()} ${now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`
  
  showSaveDialog.value = true
}

const confirmSaveSession = async () => {
  await performSave()
}

const performSave = async () => {
  if (!transcript.value) return
  
  isSaving.value = true
  
  try {
    const rawTranscript = transcript.value as TranscriptResult
    const now = new Date().toISOString()
    
    // Prepare document data if exists (matching SavedDocument interface)
    const documentData = generatedDocument.value ? {
      id: `doc_${Date.now()}`,
      templateKey: selectedTemplate.value,
      templateName: documentTemplates.find(t => t.key === selectedTemplate.value)?.name || 'Report',
      sections: generatedDocument.value.sections || [],
      htmlContent: editableDocumentContent.value,
      // Store letter details for letter templates (for regeneration)
      letterDetails: (patientName.value || recipientDoctor.value || senderDoctor.value) ? {
        patientName: patientName.value || undefined,
        recipientDoctor: recipientDoctor.value || undefined,
        senderDoctor: senderDoctor.value || undefined,
        senderTitle: senderTitle.value || undefined
      } : undefined,
      createdAt: now,
      updatedAt: now
    } : undefined
    
    const savedSession = await saveSession({
      id: currentSessionId.value || undefined,
      title: sessionTitle.value || 'Untitled Session',
      type: 'file-transcription',
      transcript: rawTranscript.text,
      document: documentData,
      interactionId: interactionId.value || undefined
    })
    
    // Store the session ID for future updates
    currentSessionId.value = savedSession.id
    
    // Update last saved content
    lastSavedContent.value = JSON.stringify({
      transcript: rawTranscript.text,
      document: editableDocumentContent.value || generatedDocument.value
    })
    
    showSaveDialog.value = false
    showSuccess.value = true
    successMessage.value = 'Session saved successfully!'
  } catch (err) {
    showError.value = true
    errorMessage.value = 'Failed to save session'
  } finally {
    isSaving.value = false
  }
}

// Check if there are unsaved changes
const hasUnsavedChanges = computed(() => {
  // No changes if nothing has been transcribed or generated
  if (!transcript.value && !generatedDocument.value) {
    return false
  }
  
  // If currently uploading/processing, there are potential unsaved changes
  if (isUploading.value || isPolling.value) {
    return true
  }
  
  const rawTranscript = transcript.value as TranscriptResult | null
  
  // Check if content has changed since last save
  const currentContent = JSON.stringify({
    transcript: rawTranscript?.text || '',
    document: editableDocumentContent.value || generatedDocument.value
  })
  
  return currentContent !== lastSavedContent.value && (rawTranscript?.text || generatedDocument.value)
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

onMounted(() => {
  window.addEventListener('beforeunload', handleBeforeUnload)
})

onUnmounted(() => {
  window.removeEventListener('beforeunload', handleBeforeUnload)
})
</script>

<style scoped>
.async-page {
  min-height: calc(100vh - 120px);
}

.document-editor :deep(.v-field__input) {
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.document-content {
  font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
  font-size: 14px;
}
</style>

