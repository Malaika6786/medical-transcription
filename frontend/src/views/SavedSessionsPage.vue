<template>
  <div class="saved-sessions-page">
    <v-row>
      <v-col cols="12">
        <div class="d-flex flex-column flex-sm-row justify-space-between align-start align-sm-center mb-4">
          <div class="d-flex align-center">
            <v-icon icon="mdi-content-save-all" class="mr-2" color="primary" size="24" />
            <h1 class="text-h6 font-weight-medium mb-0">Saved Sessions</h1>
          </div>
          <v-chip
            v-if="sessionCount > 0"
            color="primary"
            variant="tonal"
            class="mt-2 mt-sm-0"
          >
            {{ sessionCount }} session{{ sessionCount !== 1 ? 's' : '' }} saved
          </v-chip>
        </div>
      </v-col>
    </v-row>

    <!-- Filter Tabs -->
    <v-row v-if="userSessions.length > 0">
      <v-col cols="12">
        <v-tabs v-model="selectedTab" color="primary" class="mb-4" data-tour="session-filters">
          <v-tab value="all">
            <v-icon icon="mdi-view-list" class="mr-2" />
            All
          </v-tab>
          <v-tab v-if="canAccessAmbient" value="ambient">
            <v-icon icon="mdi-broadcast" class="mr-2" />
            Ambient
          </v-tab>
          <v-tab v-if="canAccessFileTranscription" value="file-transcription">
            <v-icon icon="mdi-file-upload" class="mr-2" />
            File
          </v-tab>
          <v-tab v-if="canAccessDictation" value="dictation">
            <v-icon icon="mdi-microphone-message" class="mr-2" />
            Dictation
          </v-tab>
          <v-tab v-if="canAccessEmbeddedAssistant" value="embedded-assistant">
            <v-icon icon="mdi-robot-outline" class="mr-2" />
            Embedded Assistant
          </v-tab>
        </v-tabs>
      </v-col>
    </v-row>

    <!-- Sessions List -->
    <!-- Sessions Grid - Compact Cards -->
    <v-row v-if="filteredSessions.length > 0">
      <v-col 
        v-for="(session, index) in filteredSessions" 
        :key="session.id" 
        cols="12" 
        sm="6" 
        md="4" 
        lg="3"
      >
        <v-card 
          class="session-card" 
          hover 
          @click="openSession(session)"
          :data-tour="index === 0 ? 'session-card' : undefined"
        >
          <div class="pa-3">
            <!-- Line 1: Title with Delete Button -->
            <div class="d-flex align-center justify-space-between mb-1">
              <div class="text-subtitle-2 font-weight-medium text-truncate flex-grow-1 mr-2">
                {{ session.title }}
              </div>
              <v-btn 
                icon 
                variant="text" 
                size="x-small" 
                color="error"
                @click.stop="confirmDelete(session)"
              >
                <v-icon icon="mdi-delete-outline" size="18" />
                <v-tooltip activator="parent" location="top">Delete</v-tooltip>
              </v-btn>
            </div>
            
            <!-- Line 2: Session Type Badge -->
            <div class="mb-1">
              <v-chip
                :color="getTypeColor(session.type)"
                size="x-small"
                variant="tonal"
                :prepend-icon="getTypeIcon(session.type)"
              >
                {{ getTypeLabel(session.type) }}
              </v-chip>
            </div>

            <!-- Line 3: Template Type and Time -->
            <div class="d-flex align-center justify-space-between text-caption text-medium-emphasis">
              <span v-if="session.document" class="d-flex align-center">
                <v-icon icon="mdi-file-document-check" size="12" color="success" class="mr-1" />
                {{ session.document.templateName }}
              </span>
              <span v-else class="d-flex align-center">
                <v-icon icon="mdi-text" size="12" class="mr-1" />
                Transcript only
              </span>
              <span>{{ formatDate(session.updatedAt) }}</span>
            </div>
          </div>
        </v-card>
      </v-col>
    </v-row>

    <!-- Loading State -->
    <v-row v-else-if="sessionsLoading">
      <v-col cols="12" class="text-center py-12">
        <v-progress-circular indeterminate color="primary" size="48" class="mb-4" />
        <p class="text-body-1 text-medium-emphasis">Loading sessions...</p>
      </v-col>
    </v-row>

    <!-- Empty State -->
    <v-row v-else>
      <v-col cols="12">
        <v-card class="glass-card text-center py-12">
          <v-icon icon="mdi-folder-open-outline" size="80" color="medium-emphasis" class="mb-4" />
          <h3 class="text-h5 text-medium-emphasis mb-2">No Saved Sessions</h3>
          <p class="text-body-1 text-medium-emphasis mb-4">
            Start a transcription session and save it to see it here.
          </p>
          <div class="d-flex flex-column flex-sm-row justify-center ga-3">
            <v-btn 
              v-if="canAccessAmbient"
              color="primary" 
              variant="tonal" 
              to="/ambient-session"
            >
              <v-icon icon="mdi-broadcast" class="mr-2" />
              Start Ambient Session
            </v-btn>
            <v-btn 
              v-if="canAccessFileTranscription"
              color="secondary" 
              variant="tonal" 
              to="/async-transcription"
            >
              <v-icon icon="mdi-file-upload" class="mr-2" />
              Upload File
            </v-btn>
          </div>
        </v-card>
      </v-col>
    </v-row>

    <!-- Session Detail Dialog -->
    <v-dialog v-model="detailDialog" max-width="1600" width="95%">
      <v-card v-if="selectedSession" class="session-detail-dialog">
        <!-- Fixed Header -->
        <div class="dialog-header">
          <v-card-title class="d-flex align-center pa-4 pb-3 dialog-title-row">
            <v-icon 
              :icon="getTypeIcon(selectedSession.type)" 
              :color="getTypeColor(selectedSession.type)" 
              class="mr-2 flex-shrink-0"
            />
            <span class="dialog-title-text text-truncate">{{ selectedSession.title }}</span>
            <v-spacer />
            <!-- Close Button -->
            <v-btn icon variant="text" size="small" @click="tryCloseDialog" class="flex-shrink-0">
              <v-icon icon="mdi-close" />
            </v-btn>
          </v-card-title>

          <!-- Template Selection with Save Button -->
          <template v-if="selectedSession.document">
            <div class="px-4 pb-3 d-flex align-center justify-space-between">
              <div class="d-flex align-center ga-2">
                <v-select
                  v-model="selectedTemplate"
                  :items="availableTemplates"
                  item-title="name"
                  item-value="key"
                  density="compact"
                  variant="outlined"
                  hide-details
                  style="min-width: 200px; max-width: 300px;"
                  :loading="isRegenerating"
                  :disabled="isRegenerating || !selectedSession.interactionId"
                >
                  <template #prepend-inner>
                    <v-icon icon="mdi-file-document-check" size="18" color="success" />
                  </template>
                </v-select>
                <v-btn
                  v-if="selectedSession.interactionId && selectedTemplate !== selectedSession.document.templateKey"
                  size="small"
                  color="warning"
                  variant="tonal"
                  :loading="isRegenerating"
                  @click="regenerateWithTemplate"
                >
                  <v-icon icon="mdi-refresh" class="mr-1" />
                  Regenerate
                </v-btn>
              </div>
              <v-btn 
                icon 
                variant="text" 
                size="small" 
                color="primary"
                @click="saveDocumentEdit"
              >
                <v-icon icon="mdi-content-save" />
                <v-tooltip activator="parent" location="left">Save</v-tooltip>
              </v-btn>
            </div>
          </template>

          <!-- Tab bar: shown when there's more than one view (report / transcript / AI) -->
          <v-tabs
            v-if="selectedSession.document || selectedSession.extraction"
            v-model="dialogTab"
            color="primary"
            density="compact"
            class="px-4"
          >
            <v-tab v-if="selectedSession.document" value="report">
              <v-icon icon="mdi-file-document-check" size="16" class="mr-1" />
              Report
            </v-tab>
            <v-tab value="transcript" :disabled="!selectedSession.transcript">
              <v-icon icon="mdi-text" size="16" class="mr-1" />
              Transcript
            </v-tab>
            <v-tab v-if="selectedSession.extraction" value="ai">
              <v-icon icon="mdi-robot-outline" size="16" class="mr-1" />
              AI Extraction
            </v-tab>
          </v-tabs>

          <v-divider />
        </div>

        <!-- Scrollable Content Area -->
        <v-card-text class="pa-4 dialog-content">
          <!-- AI extraction tab — restored from the saved record, chat resumes grounded in it -->
          <div v-if="dialogTab === 'ai' && selectedSession.extraction" class="ai-content">
            <AiAssistantPanel
              :transcript="selectedSession.transcript"
              :initial-extraction="selectedSession.extraction"
            />
          </div>

          <!-- Session has a document: show report or transcript depending on tab -->
          <template v-else-if="selectedSession.document">
            <!-- Report tab -->
            <div v-if="dialogTab === 'report'" class="report-content">
              <RichTextEditor
                ref="richTextEditorRef"
                v-model="editableDocument"
                :editable="true"
                :full-height="true"
                :hide-toolbar="true"
              />
            </div>

            <!-- Transcript tab -->
            <div v-else-if="dialogTab === 'transcript'" class="transcript-content">
              <template v-if="selectedSession.transcript">
                <div class="transcript-text">{{ selectedSession.transcript }}</div>
              </template>
              <v-alert v-else type="info" variant="tonal">
                No transcript text was saved with this session.
              </v-alert>
            </div>
          </template>

          <!-- No document — show transcript directly -->
          <template v-else>
            <template v-if="selectedSession.transcript">
              <div class="d-flex align-center mb-3">
                <v-icon icon="mdi-text" class="mr-2" color="primary" />
                <span class="text-subtitle-2 font-weight-medium">Transcript</span>
              </div>
              <div class="transcript-text">{{ selectedSession.transcript }}</div>
            </template>
            <v-alert v-else type="info" variant="tonal" class="mb-0">
              <v-icon icon="mdi-information" class="mr-2" />
              This session has no transcript or report saved.
            </v-alert>
          </template>
        </v-card-text>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog" max-width="400">
      <v-card class="glass-card">
        <v-card-title class="text-h6">
          <v-icon icon="mdi-alert" color="error" class="mr-2" />
          Confirm Delete
        </v-card-title>
        <v-card-text>
          Are you sure you want to delete "{{ sessionToDelete?.title }}"? This action cannot be undone.
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn color="error" @click="handleDelete">Delete</v-btn>
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
          <p>You have unsaved changes to this document.</p>
          <p class="text-medium-emphasis mb-0">Would you like to stay and save your work, or leave without saving?</p>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="confirmLeave">Leave Without Saving</v-btn>
          <v-btn color="primary" @click="cancelLeave">
            <v-icon icon="mdi-content-save" class="mr-1" />
            Stay and Save
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Letter Details Dialog (for regenerating letter templates) -->
    <v-dialog v-model="showLetterDetailsDialog" max-width="500" persistent>
      <v-card>
        <v-card-title class="d-flex align-center">
          <v-icon icon="mdi-email-edit-outline" color="secondary" class="mr-2" />
          GP Letter Details
        </v-card-title>
        <v-card-text>
          <p class="text-body-2 text-medium-emphasis mb-4">
            Enter the details for the letter placeholders:
          </p>
          <v-text-field
            v-model="letterDetailsForm.patientName"
            label="Patient Name *"
            placeholder="e.g., David Chen"
            prepend-inner-icon="mdi-account"
            density="compact"
            class="mb-3"
          />
          <v-text-field
            v-model="letterDetailsForm.recipientDoctor"
            label="Recipient Doctor (Dear...)"
            placeholder="e.g., Dr Smith"
            prepend-inner-icon="mdi-account-arrow-right"
            density="compact"
            class="mb-3"
          />
          <v-text-field
            v-model="letterDetailsForm.senderDoctor"
            label="Your Name (Sign-off)"
            placeholder="e.g., Dr Sheru George"
            prepend-inner-icon="mdi-doctor"
            density="compact"
            class="mb-3"
          />
          <v-text-field
            v-model="letterDetailsForm.senderTitle"
            label="Your Title"
            placeholder="e.g., General Practitioner"
            prepend-inner-icon="mdi-badge-account"
            density="compact"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="showLetterDetailsDialog = false">Cancel</v-btn>
          <v-btn 
            color="warning" 
            variant="tonal"
            :loading="isRegenerating"
            @click="confirmRegenerateWithLetterDetails"
          >
            <v-icon icon="mdi-refresh" class="mr-1" />
            Regenerate
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Snackbars -->
    <v-snackbar v-model="showSuccess" color="success" timeout="3000">
      {{ successMessage }}
    </v-snackbar>

    <v-snackbar v-model="showError" color="error" timeout="5000">
      {{ errorMessage }}
    </v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { onBeforeRouteLeave, useRoute } from 'vue-router'
import RichTextEditor from '@/components/RichTextEditor.vue'
import AiAssistantPanel from '@/components/AiAssistantPanel.vue'
import { 
  userSessions, 
  deleteSession, 
  updateSessionDocument,
  sessionCount,
  loadSessions,
  sessionsLoading,
  type SavedSession,
  type SavedDocument
} from '@/stores/sessions'
import {
  canAccessAmbient,
  canAccessFileTranscription,
  canAccessDictation,
  canAccessEmbeddedAssistant
} from '@/stores/auth'
import api from '@/services/api'

// Template types
interface DocumentTemplate {
  key: string
  name: string
  description?: string
}

// Load sessions and templates on mount
const route = useRoute()

onMounted(async () => {
  await Promise.all([loadSessions(), fetchTemplates()])

  // Deep link from Semantic Search: /saved-sessions?open=<sessionId>
  const openId = route.query.open
  if (typeof openId === 'string' && openId) {
    const target = userSessions.value.find(s => s.id === openId)
    if (target) openSession(target)
  }
})

const selectedTab = ref('all')
const detailDialog = ref(false)
const deleteDialog = ref(false)
const selectedSession = ref<SavedSession | null>(null)
const sessionToDelete = ref<SavedSession | null>(null)
const editableDocument = ref('')
const dialogTab = ref<'report' | 'transcript' | 'ai'>('report')
const showSuccess = ref(false)
const showError = ref(false)
const successMessage = ref('')
const errorMessage = ref('')

// Template selection
const availableTemplates = ref<DocumentTemplate[]>([])
const selectedTemplate = ref('')
const isRegenerating = ref(false)

// Letter details for regeneration (when not stored in session)
const showLetterDetailsDialog = ref(false)
const letterDetailsForm = ref({
  patientName: '',
  recipientDoctor: '',
  senderDoctor: '',
  senderTitle: 'General Practitioner'
})

// Check if selected template is a letter template
const isLetterTemplate = computed(() => {
  const template = availableTemplates.value.find(t => t.key === selectedTemplate.value)
  return template?.key?.includes('letter') || template?.key?.includes('gp-letter')
})

// Unsaved changes tracking
const originalDocumentContent = ref('')
const showUnsavedDialog = ref(false)
const pendingNavigation = ref<(() => void) | null>(null)

// Pending regenerated document (not yet saved)
const pendingRegeneratedDocument = ref<SavedDocument | null>(null)

// Fetch templates on mount
const fetchTemplates = async () => {
  try {
    const response = await api.get('/templates')
    if (response.data.success) {
      availableTemplates.value = response.data.templates
    }
  } catch (err) {
    console.error('Failed to fetch templates:', err)
  }
}

// Filter sessions by type
const filteredSessions = computed(() => {
  if (selectedTab.value === 'all') {
    return userSessions.value
  }
  return userSessions.value.filter(s => s.type === selectedTab.value)
})

// Get icon for session type
const getTypeIcon = (type: SavedSession['type']): string => {
  const icons: Record<string, string> = {
    'ambient': 'mdi-broadcast',
    'file-transcription': 'mdi-file-upload',
    'dictation': 'mdi-microphone-message',
    'embedded-assistant': 'mdi-robot-outline',
  }
  return icons[type] || 'mdi-file'
}

// Get color for session type
const getTypeColor = (type: SavedSession['type']): string => {
  const colors: Record<string, string> = {
    'ambient': 'secondary',
    'file-transcription': 'primary',
    'dictation': 'info',
    'embedded-assistant': 'purple',
  }
  return colors[type] || 'grey'
}

// Get human-readable label for session type
const getTypeLabel = (type: SavedSession['type']): string => {
  const labels: Record<string, string> = {
    'ambient': 'Ambient',
    'file-transcription': 'File',
    'dictation': 'Dictation',
    'embedded-assistant': 'Embedded Assistant',
  }
  return labels[type] || type
}

// Format date
const formatDate = (dateString: string): string => {
  const date = new Date(dateString)
  return date.toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
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


// Open session detail
const openSession = (session: SavedSession) => {
  selectedSession.value = session
  dialogTab.value = session.document ? 'report' : (session.extraction ? 'ai' : 'transcript')
  // Clear any pending regenerated document from previous session
  pendingRegeneratedDocument.value = null
  if (session.document) {
    // Use saved HTML content if available, otherwise generate from sections
    editableDocument.value = session.document.htmlContent || sectionsToHtml(session.document.sections)
    // Store original content for comparison
    originalDocumentContent.value = editableDocument.value
    // Set the selected template to the current document's template
    selectedTemplate.value = session.document.templateKey
  }
  detailDialog.value = true
}

// Confirm delete
const confirmDelete = (session: SavedSession) => {
  sessionToDelete.value = session
  deleteDialog.value = true
}

// Handle delete
const handleDelete = async () => {
  if (sessionToDelete.value) {
    const success = await deleteSession(sessionToDelete.value.id)
    if (success) {
      showSuccess.value = true
      successMessage.value = 'Session deleted successfully'
    } else {
      showError.value = true
      errorMessage.value = 'Failed to delete session'
    }
    deleteDialog.value = false
    sessionToDelete.value = null
  }
}


// Save document edit
const saveDocumentEdit = async () => {
  if (!selectedSession.value?.document) return
  
  // Use pending regenerated document if available, otherwise update existing
  let updatedDocument: SavedDocument
  
  if (pendingRegeneratedDocument.value) {
    // Use the regenerated document with the current edited content
    updatedDocument = {
      ...pendingRegeneratedDocument.value,
      htmlContent: editableDocument.value,
      updatedAt: new Date().toISOString()
    }
  } else {
    // Just update the HTML content of the existing document
    updatedDocument = {
      ...selectedSession.value.document,
      htmlContent: editableDocument.value,
      updatedAt: new Date().toISOString()
    }
  }
  
  const success = await updateSessionDocument(selectedSession.value.id, updatedDocument)
  
  if (success) {
    // Update the local reference
    selectedSession.value.document = updatedDocument
    // Update original content to match saved state
    originalDocumentContent.value = editableDocument.value
    // Clear the pending regenerated document
    pendingRegeneratedDocument.value = null
    showSuccess.value = true
    successMessage.value = 'Report saved successfully!'
  } else {
    showError.value = true
    errorMessage.value = 'Failed to save report'
  }
}

// Regenerate document with new template
const regenerateWithTemplate = async () => {
  if (!selectedSession.value?.interactionId || !selectedTemplate.value) return
  
  // Check if this is a letter template and we don't have stored letter details
  const storedLetterDetails = selectedSession.value.document?.letterDetails
  if (isLetterTemplate.value && !storedLetterDetails) {
    // Pre-fill form with any existing values from the document
    letterDetailsForm.value = {
      patientName: '',
      recipientDoctor: '',
      senderDoctor: '',
      senderTitle: 'General Practitioner'
    }
    showLetterDetailsDialog.value = true
    return
  }
  
  // Proceed with regeneration using stored details
  await performRegeneration(storedLetterDetails)
}

// Confirm regeneration with letter details from dialog
const confirmRegenerateWithLetterDetails = async () => {
  showLetterDetailsDialog.value = false
  await performRegeneration(letterDetailsForm.value)
}

// Perform the actual regeneration
const performRegeneration = async (letterDetails?: { patientName?: string; recipientDoctor?: string; senderDoctor?: string; senderTitle?: string }) => {
  if (!selectedSession.value?.interactionId || !selectedTemplate.value) return
  
  isRegenerating.value = true
  
  try {
    // Build request payload - include letter details if available
    const requestPayload: Record<string, string> = {
      templateKey: selectedTemplate.value
    }
    
    // Include letter details for letter templates
    if (letterDetails) {
      if (letterDetails.patientName) requestPayload.patientName = letterDetails.patientName
      if (letterDetails.recipientDoctor) requestPayload.recipientDoctor = letterDetails.recipientDoctor
      if (letterDetails.senderDoctor) requestPayload.senderDoctor = letterDetails.senderDoctor
      if (letterDetails.senderTitle) requestPayload.senderTitle = letterDetails.senderTitle
    }
    
    // Step 1: Request document generation
    const generateResponse = await api.post(`/transcribe/${selectedSession.value.interactionId}/document`, requestPayload)
    
    const documentId = generateResponse.data.document_id
    if (!documentId) {
      throw new Error('No document ID returned')
    }
    
    // Step 2: Poll for document completion
    let attempts = 0
    const maxAttempts = 30
    let documentData = null
    
    while (attempts < maxAttempts) {
      const docResponse = await api.get(
        `/transcribe/${selectedSession.value.interactionId}/document/${documentId}`
      )
      
      if (docResponse.data.success && docResponse.data.document) {
        const doc = docResponse.data.document
        
        if (doc.sections && doc.sections.length > 0) {
          documentData = doc
          break
        } else if (doc.status === 'failed') {
          throw new Error('Document generation failed')
        }
      }
      
      // Wait 2 seconds before next poll
      await new Promise(resolve => setTimeout(resolve, 2000))
      attempts++
    }
    
    if (!documentData) {
      throw new Error('Document generation timed out')
    }
    
    // Step 3: Update the preview (don't auto-save - let user save manually)
    const template = availableTemplates.value.find(t => t.key === selectedTemplate.value)
    
    // Store the pending document data for when user saves
    pendingRegeneratedDocument.value = {
      id: selectedSession.value.document?.id || `doc_${Date.now()}`,
      templateKey: selectedTemplate.value,
      templateName: template?.name || selectedTemplate.value,
      sections: documentData.sections,
      htmlContent: sectionsToHtml(documentData.sections),
      // Store the letter details used for this regeneration
      letterDetails: letterDetails ? {
        patientName: letterDetails.patientName,
        recipientDoctor: letterDetails.recipientDoctor,
        senderDoctor: letterDetails.senderDoctor,
        senderTitle: letterDetails.senderTitle
      } : selectedSession.value.document?.letterDetails,
      createdAt: selectedSession.value.document?.createdAt || new Date().toISOString(),
      updatedAt: new Date().toISOString()
    }
    
    // Update the editable document preview (this will trigger hasUnsavedChanges)
    editableDocument.value = pendingRegeneratedDocument.value.htmlContent || ''
    
    showSuccess.value = true
    successMessage.value = `Report regenerated with ${template?.name || selectedTemplate.value} template! Click Save to keep changes.`
  } catch (err: any) {
    console.error('Failed to regenerate document:', err)
    showError.value = true
    errorMessage.value = err.response?.data?.error || err.message || 'Failed to regenerate report'
  } finally {
    isRegenerating.value = false
  }
}

// Check if there are unsaved changes in the document editor
const hasUnsavedChanges = computed(() => {
  if (!detailDialog.value || !selectedSession.value?.document) {
    return false
  }
  return editableDocument.value !== originalDocumentContent.value
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

// Try to close dialog - check for unsaved changes first
const tryCloseDialog = () => {
  if (hasUnsavedChanges.value) {
    showUnsavedDialog.value = true
    pendingNavigation.value = () => {
      detailDialog.value = false
      originalDocumentContent.value = ''
    }
  } else {
    detailDialog.value = false
    originalDocumentContent.value = ''
  }
}

onMounted(() => {
  window.addEventListener('beforeunload', handleBeforeUnload)
})

onUnmounted(() => {
  window.removeEventListener('beforeunload', handleBeforeUnload)
})

</script>

<style scoped>
.saved-sessions-page {
  min-height: calc(100vh - 120px);
}

/* Compact session card styles */
.session-card {
  border-radius: 12px;
  transition: all 0.2s ease;
  cursor: pointer;
  background: rgba(var(--v-theme-surface), 0.8);
  border: 1px solid rgba(var(--v-border-color), 0.1);
}

.session-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  border-color: rgba(var(--v-theme-primary), 0.3);
}

/* Session Detail Dialog Styles */
.session-detail-dialog {
  display: flex;
  flex-direction: column;
  max-height: 95vh;
  min-height: 70vh;
  overflow: hidden;
}

.dialog-header {
  flex-shrink: 0;
  background: rgb(var(--v-theme-surface));
  z-index: 2;
}

.dialog-title-row {
  min-width: 0; /* Allow flex children to shrink below content size */
}

.dialog-title-text {
  min-width: 0; /* Allow text to truncate */
  flex: 1 1 auto;
}

.dialog-content {
  flex: 1 1 auto;
  overflow-y: auto;
  min-height: 0; /* Important for flex child scrolling */
}

.dialog-footer {
  flex-shrink: 0;
  background: rgb(var(--v-theme-surface));
  z-index: 2;
}

.report-content {
  /* No max-height - let the dialog-content handle scrolling */
}

/* Remove border from RichTextEditor when used in dialog (since dialog provides structure) */
.report-content :deep(.rich-text-editor) {
  border: none;
  border-radius: 0;
}

.transcript-text {
  font-family: 'Segoe UI', sans-serif;
  font-size: 0.9rem;
  line-height: 1.8;
  white-space: pre-wrap;
  word-break: break-word;
  color: rgb(var(--v-theme-on-surface));
  padding: 4px 0;
}

/* Mobile responsiveness */
@media (max-width: 600px) {
  .saved-sessions-page h1 {
    font-size: 1.5rem !important;
  }
  
  .session-card {
    border-radius: 8px;
  }
  
  .session-detail-dialog {
    max-height: 95vh;
  }
}
</style>
