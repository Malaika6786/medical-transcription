<template>
  <div class="transcript-test-page">
    <v-row>
      <v-col cols="12">
        <div class="d-flex align-center mb-4">
          <v-icon icon="mdi-test-tube" class="mr-2" color="primary" size="24" />
          <h1 class="text-h6 font-weight-medium mb-0">Template Test (Transcript Only)</h1>
          <v-chip color="warning" variant="tonal" size="small" class="ml-3">Admin</v-chip>
        </div>
        <p class="text-body-2 text-medium-emphasis mb-4">
          Test custom templates by pasting transcript text directly. No audio upload required.
        </p>
      </v-col>
    </v-row>

    <v-row>
      <!-- Input Section -->
      <v-col cols="12" md="5">
        <v-card class="mb-4">
          <v-card-title class="text-subtitle-1">
            <v-icon icon="mdi-text-box" class="mr-2" />
            Transcript Input
          </v-card-title>
          <v-card-text>
            <v-textarea
              v-model="transcript"
              label="Paste Transcript Text"
              placeholder="Enter or paste the clinical transcript here..."
              rows="12"
              variant="outlined"
              counter
              :disabled="isGenerating"
            />
          </v-card-text>
        </v-card>

        <!-- Template Selection -->
        <v-card class="mb-4">
          <v-card-title class="text-subtitle-1">
            <v-icon icon="mdi-file-document-outline" class="mr-2" />
            Template Selection
          </v-card-title>
          <v-card-text>
            <v-select
              v-model="selectedTemplate"
              :items="customTemplates"
              item-title="name"
              item-value="key"
              label="Select Template"
              variant="outlined"
              :disabled="isGenerating"
              :loading="isLoadingTemplates"
            >
              <template #item="{ item, props }">
                <v-list-item v-bind="props">
                  <template #subtitle>{{ item.raw.description }}</template>
                </v-list-item>
              </template>
            </v-select>

            <v-alert v-if="selectedTemplate && !isCustomTemplate" type="warning" variant="tonal" density="compact" class="mt-3">
              Only custom templates are supported for direct transcript testing.
            </v-alert>
          </v-card-text>
        </v-card>

        <!-- Letter Details (for GP Letter templates) -->
        <v-card v-if="isLetterTemplate" class="mb-4">
          <v-card-title class="text-subtitle-1">
            <v-icon icon="mdi-card-account-details" class="mr-2" />
            Letter Details
          </v-card-title>
          <v-card-text>
            <v-text-field
              v-model="patientName"
              label="Patient Name"
              variant="outlined"
              density="compact"
              placeholder="e.g., John Smith"
              class="mb-3"
            />
            <v-text-field
              v-model="recipientDoctor"
              label="Recipient Doctor"
              variant="outlined"
              density="compact"
              placeholder="e.g., Dr. Ahmed"
              class="mb-3"
            />
            <v-text-field
              v-model="senderDoctor"
              label="Sender Doctor"
              variant="outlined"
              density="compact"
              placeholder="e.g., Dr. Smith"
              class="mb-3"
            />
            <v-text-field
              v-model="senderTitle"
              label="Sender Title"
              variant="outlined"
              density="compact"
              placeholder="e.g., General Practitioner"
            />
          </v-card-text>
        </v-card>

        <!-- Generate Button -->
        <v-btn
          block
          size="large"
          color="primary"
          :loading="isGenerating"
          :disabled="!canGenerate"
          @click="generateDocument"
        >
          <v-icon icon="mdi-file-document-plus" class="mr-2" />
          Generate Document
        </v-btn>
      </v-col>

      <!-- Output Section -->
      <v-col cols="12" md="7">
        <v-card class="output-card">
          <v-card-title class="d-flex align-center text-subtitle-1">
            <v-icon icon="mdi-file-document-check" class="mr-2" />
            Generated Document
            <v-spacer />
            <v-btn
              v-if="generatedDocument"
              variant="text"
              size="small"
              @click="copyDocument"
            >
              <v-icon icon="mdi-content-copy" class="mr-1" />
              Copy
            </v-btn>
          </v-card-title>
          <v-card-text>
            <!-- Loading State -->
            <div v-if="isGenerating" class="text-center py-8">
              <v-progress-circular indeterminate color="primary" size="48" />
              <p class="text-body-2 text-medium-emphasis mt-4">Generating document...</p>
            </div>

            <!-- Empty State -->
            <div v-else-if="!generatedDocument" class="text-center py-12">
              <v-icon icon="mdi-file-document-outline" size="64" color="grey-lighten-1" />
              <p class="text-body-2 text-medium-emphasis mt-4">
                Enter transcript and select a template to generate a document
              </p>
            </div>

            <!-- Document Output -->
            <div v-else class="document-output">
              <div
                v-for="section in generatedDocument.sections"
                :key="section.key"
                class="section-block mb-4"
              >
                <h3 class="text-subtitle-2 font-weight-bold text-primary mb-2">
                  {{ section.name }}
                </h3>
                <div class="section-content" v-html="formatSectionContent(section.text || section.content || '')" />
              </div>
            </div>
          </v-card-text>
        </v-card>

        <!-- Interaction Info -->
        <v-card v-if="interactionId" class="mt-4" variant="tonal">
          <v-card-text class="py-2">
            <div class="d-flex align-center text-caption">
              <v-icon icon="mdi-identifier" size="16" class="mr-2" />
              <span class="text-medium-emphasis">Interaction ID:</span>
              <code class="ml-2">{{ interactionId }}</code>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Error Snackbar -->
    <v-snackbar v-model="showError" color="error" :timeout="5000">
      {{ errorMessage }}
    </v-snackbar>

    <!-- Success Snackbar -->
    <v-snackbar v-model="showSuccess" color="success" :timeout="3000">
      {{ successMessage }}
    </v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import api from '@/services/api'

interface Template {
  key: string
  name: string
  description: string
  category?: string
  isCustom?: boolean
}

interface DocumentSection {
  key?: string
  name: string
  text?: string
  content?: string
}

interface GeneratedDocument {
  name: string
  sections: DocumentSection[]
}

// State
const transcript = ref('')
const selectedTemplate = ref('')
const patientName = ref('')
const recipientDoctor = ref('')
const senderDoctor = ref('')
const senderTitle = ref('General Practitioner')

const templates = ref<Template[]>([])
const isLoadingTemplates = ref(false)
const isGenerating = ref(false)
const generatedDocument = ref<GeneratedDocument | null>(null)
const interactionId = ref('')

const showError = ref(false)
const errorMessage = ref('')
const showSuccess = ref(false)
const successMessage = ref('')

// Computed
const customTemplates = computed(() => {
  return templates.value.filter(t => t.isCustom)
})

const isCustomTemplate = computed(() => {
  const template = templates.value.find(t => t.key === selectedTemplate.value)
  return template?.isCustom ?? false
})

const isLetterTemplate = computed(() => {
  const template = templates.value.find(t => t.key === selectedTemplate.value)
  return template?.category === 'letter' || selectedTemplate.value.includes('letter')
})

const canGenerate = computed(() => {
  return transcript.value.trim().length > 0 && selectedTemplate.value && isCustomTemplate.value
})

// Methods
const loadTemplates = async () => {
  isLoadingTemplates.value = true
  try {
    const response = await api.get('/templates')
    if (response.data.success) {
      templates.value = response.data.templates || []
    }
  } catch (err) {
    console.error('Failed to load templates:', err)
    showError.value = true
    errorMessage.value = 'Failed to load templates'
  } finally {
    isLoadingTemplates.value = false
  }
}

const generateDocument = async () => {
  if (!canGenerate.value) return

  isGenerating.value = true
  generatedDocument.value = null
  interactionId.value = ''

  try {
    const response = await api.post('/transcribe/generate-from-transcript', {
      transcript: transcript.value,
      templateKey: selectedTemplate.value,
      patientName: patientName.value || undefined,
      recipientDoctor: recipientDoctor.value || undefined,
      senderDoctor: senderDoctor.value || undefined,
      senderTitle: senderTitle.value || undefined
    })

    if (response.data.success) {
      generatedDocument.value = response.data.document
      interactionId.value = response.data.interactionId || ''
      showSuccess.value = true
      successMessage.value = 'Document generated successfully!'
    } else {
      throw new Error(response.data.error || 'Failed to generate document')
    }
  } catch (err: any) {
    console.error('Failed to generate document:', err)
    showError.value = true
    errorMessage.value = err.response?.data?.error || err.message || 'Failed to generate document'
  } finally {
    isGenerating.value = false
  }
}

const formatSectionContent = (content: string): string => {
  if (!content) return ''

  // Escape HTML first — content can contain arbitrary transcript/document
  // text, so this must never be trusted as markup (this is rendered via
  // v-html below).
  let formatted = content
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

  // Convert markdown-style bold to HTML
  formatted = formatted.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')

  // Convert newlines to <br>
  formatted = formatted.replace(/\n/g, '<br>')

  return formatted
}

const copyDocument = async () => {
  if (!generatedDocument.value) return

  const text = generatedDocument.value.sections
    .map(s => `${s.name}\n\n${s.text || s.content}`)
    .join('\n\n---\n\n')

  try {
    await navigator.clipboard.writeText(text)
    showSuccess.value = true
    successMessage.value = 'Document copied to clipboard!'
  } catch (err) {
    console.error('Failed to copy:', err)
    showError.value = true
    errorMessage.value = 'Failed to copy to clipboard'
  }
}

// Lifecycle
onMounted(() => {
  loadTemplates()
})
</script>

<style scoped>
.transcript-test-page {
  max-width: 1400px;
  margin: 0 auto;
  padding: 16px;
}

.output-card {
  min-height: 500px;
}

.document-output {
  max-height: 600px;
  overflow-y: auto;
}

.section-block {
  padding-bottom: 16px;
  border-bottom: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}

.section-block:last-child {
  border-bottom: none;
}

.section-content {
  line-height: 1.6;
  color: rgb(var(--v-theme-on-surface));
}

.section-content :deep(strong) {
  font-weight: 600;
}
</style>
