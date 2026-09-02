<template>
  <div class="report-generator">
    <!-- Generation status -->
    <v-alert
      v-if="isGenerating"
      type="info"
      variant="tonal"
      class="mb-4"
    >
      <v-progress-circular indeterminate size="20" class="mr-2" />
      Generating {{ selectedTemplateName }}...
    </v-alert>

    <!-- Template selector -->
    <v-select
      v-model="localSelectedTemplate"
      :items="templates"
      item-title="name"
      item-value="key"
      label="Select Template"
      prepend-inner-icon="mdi-file-document-outline"
      density="compact"
      class="mb-3"
      :disabled="isGenerating"
      data-tour="template-select"
    >
      <template v-slot:item="{ item, props }">
        <v-list-item v-bind="props">
          <template v-slot:subtitle>
            {{ item.raw.description }}
          </template>
          <template v-slot:append v-if="item.raw.isCustom">
            <v-chip size="x-small" color="secondary" variant="tonal">Custom</v-chip>
          </template>
        </v-list-item>
      </template>
    </v-select>

    <!-- GP Letter Options (shown when GP Letter is selected) -->
    <v-card v-if="isGPLetterSelected" variant="outlined" class="mb-3 pa-3">
      <div class="d-flex align-center mb-3">
        <v-icon icon="mdi-email-edit-outline" class="mr-2" color="secondary" />
        <span class="text-subtitle-2 font-weight-medium">GP Letter Details</span>
      </div>
      
      <v-row dense>
        <v-col cols="12">
          <v-text-field
            v-model="localPatientName"
            label="Patient Name *"
            placeholder="e.g., David Chen"
            prepend-inner-icon="mdi-account"
            density="compact"
            hint="Re: [Patient Name], DOB..."
            persistent-hint
          />
        </v-col>
        <v-col cols="12">
          <v-text-field
            v-model="localRecipientDoctor"
            label="Recipient Doctor (Dear...)"
            placeholder="e.g., Dr Smith or Doctor"
            prepend-inner-icon="mdi-account-arrow-right"
            density="compact"
            hint="Dear [Recipient],"
            persistent-hint
          />
        </v-col>
        <v-col cols="12" sm="6">
          <v-text-field
            v-model="localSenderDoctor"
            label="Your Name (Sign-off)"
            placeholder="e.g., Dr Sheru George"
            prepend-inner-icon="mdi-doctor"
            density="compact"
          />
        </v-col>
        <v-col cols="12" sm="6">
          <v-text-field
            v-model="localSenderTitle"
            label="Your Title"
            placeholder="e.g., General Practitioner"
            prepend-inner-icon="mdi-badge-account"
            density="compact"
          />
        </v-col>
      </v-row>
      
      <v-alert
        type="info"
        variant="tonal"
        density="compact"
        class="mt-3"
      >
        <span class="text-caption">
          Generates a formal referral letter with narrative paragraphs and numbered lists for diagnoses, investigations, medications, and management plan.
        </span>
      </v-alert>
    </v-card>

    <!-- Verbosity Selector (ONLY for SOAP Note template and Super User) -->
    <v-card v-if="isSOAPSelected && showVerbosityOptions" variant="outlined" class="mb-3 pa-3">
      <div class="d-flex align-center mb-2">
        <v-icon icon="mdi-tune-vertical" class="mr-2" color="primary" />
        <span class="text-subtitle-2 font-weight-medium">Output Detail Level</span>
      </div>
      <v-radio-group
        v-model="localSelectedVerbosity"
        inline
        hide-details
        density="compact"
        class="mt-2"
      >
        <v-radio
          v-for="option in verbosityOptions"
          :key="option.key"
          :value="option.key"
          :label="option.name"
          color="primary"
        />
      </v-radio-group>
      <p class="text-caption text-medium-emphasis mt-2 mb-0">
        <v-icon :icon="verbosityOptions.find(o => o.key === localSelectedVerbosity)?.icon" size="14" class="mr-1" />
        {{ verbosityOptions.find(o => o.key === localSelectedVerbosity)?.description }}
      </p>

      <!-- Section Overrides Toggle -->
      <v-divider class="my-3" />
      <v-btn
        variant="text"
        size="small"
        class="px-0"
        @click="showSectionOverrides = !showSectionOverrides"
      >
        <v-icon :icon="showSectionOverrides ? 'mdi-chevron-up' : 'mdi-chevron-down'" class="mr-1" />
        {{ showSectionOverrides ? 'Hide' : 'Show' }} Section Customization
      </v-btn>

      <!-- Section Override Fields -->
      <v-expand-transition>
        <div v-if="showSectionOverrides" class="mt-3">
          <v-alert type="info" variant="tonal" density="compact" class="mb-3">
            <span class="text-caption">
              Customize the prompts for each SOAP section. These are pre-filled based on the selected verbosity level.
            </span>
          </v-alert>

          <v-expansion-panels variant="accordion" class="mb-2">
            <v-expansion-panel>
              <v-expansion-panel-title class="text-subtitle-2">
                <v-icon icon="mdi-account-voice" class="mr-2" size="small" color="primary" />
                Subjective
              </v-expansion-panel-title>
              <v-expansion-panel-text>
                <v-textarea
                  v-model="localSoapOverrides.subjective.writingStyle"
                  label="Writing Style"
                  variant="outlined"
                  density="compact"
                  rows="2"
                  class="mb-2"
                  hint="How the section should be written"
                  persistent-hint
                />
                <v-textarea
                  v-model="localSoapOverrides.subjective.additionalInstructions"
                  label="Additional Instructions"
                  variant="outlined"
                  density="compact"
                  rows="3"
                  hint="Specific instructions for this section"
                  persistent-hint
                />
              </v-expansion-panel-text>
            </v-expansion-panel>

            <v-expansion-panel>
              <v-expansion-panel-title class="text-subtitle-2">
                <v-icon icon="mdi-stethoscope" class="mr-2" size="small" color="secondary" />
                Objective
              </v-expansion-panel-title>
              <v-expansion-panel-text>
                <v-textarea
                  v-model="localSoapOverrides.objective.writingStyle"
                  label="Writing Style"
                  variant="outlined"
                  density="compact"
                  rows="2"
                  class="mb-2"
                  hint="How the section should be written"
                  persistent-hint
                />
                <v-textarea
                  v-model="localSoapOverrides.objective.additionalInstructions"
                  label="Additional Instructions"
                  variant="outlined"
                  density="compact"
                  rows="3"
                  hint="Specific instructions for this section"
                  persistent-hint
                />
              </v-expansion-panel-text>
            </v-expansion-panel>

            <v-expansion-panel>
              <v-expansion-panel-title class="text-subtitle-2">
                <v-icon icon="mdi-clipboard-text" class="mr-2" size="small" color="warning" />
                Assessment
              </v-expansion-panel-title>
              <v-expansion-panel-text>
                <v-textarea
                  v-model="localSoapOverrides.assessment.writingStyle"
                  label="Writing Style"
                  variant="outlined"
                  density="compact"
                  rows="2"
                  class="mb-2"
                  hint="How the section should be written"
                  persistent-hint
                />
                <v-textarea
                  v-model="localSoapOverrides.assessment.additionalInstructions"
                  label="Additional Instructions"
                  variant="outlined"
                  density="compact"
                  rows="3"
                  hint="Specific instructions for this section"
                  persistent-hint
                />
              </v-expansion-panel-text>
            </v-expansion-panel>

            <v-expansion-panel>
              <v-expansion-panel-title class="text-subtitle-2">
                <v-icon icon="mdi-clipboard-check" class="mr-2" size="small" color="success" />
                Plan
              </v-expansion-panel-title>
              <v-expansion-panel-text>
                <v-textarea
                  v-model="localSoapOverrides.plan.writingStyle"
                  label="Writing Style"
                  variant="outlined"
                  density="compact"
                  rows="2"
                  class="mb-2"
                  hint="How the section should be written"
                  persistent-hint
                />
                <v-textarea
                  v-model="localSoapOverrides.plan.additionalInstructions"
                  label="Additional Instructions"
                  variant="outlined"
                  density="compact"
                  rows="3"
                  hint="Specific instructions for this section"
                  persistent-hint
                />
              </v-expansion-panel-text>
            </v-expansion-panel>
          </v-expansion-panels>

          <v-btn
            variant="tonal"
            size="small"
            color="secondary"
            @click="applyVerbosityPreset"
            class="mt-2"
          >
            <v-icon icon="mdi-refresh" class="mr-1" />
            Reset to {{ localSelectedVerbosity }} Defaults
          </v-btn>
        </div>
      </v-expand-transition>
    </v-card>

    <v-btn
      block
      color="secondary"
      :disabled="!localSelectedTemplate || isGenerating"
      :loading="isGenerating"
      @click="$emit('generate')"
      data-tour="generate-document"
    >
      <v-icon icon="mdi-auto-fix" class="mr-2" />
      Generate Report
    </v-btn>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

interface DocumentTemplate {
  key: string
  name: string
  description: string
  isCustom?: boolean
  category?: string
}

interface SoapSectionOverride {
  writingStyle: string
  formatRule?: string
  additionalInstructions: string
}

interface SoapOverrides {
  subjective: SoapSectionOverride
  objective: SoapSectionOverride
  assessment: SoapSectionOverride
  plan: SoapSectionOverride
}

const props = withDefaults(defineProps<{
  templates: DocumentTemplate[]
  selectedTemplate: string
  selectedVerbosity?: string
  isGenerating?: boolean
  showVerbosityOptions?: boolean
  patientName?: string
  recipientDoctor?: string
  senderDoctor?: string
  senderTitle?: string
  soapOverrides?: SoapOverrides
}>(), {
  selectedVerbosity: 'concise',
  isGenerating: false,
  showVerbosityOptions: true,
  patientName: '',
  recipientDoctor: '',
  senderDoctor: '',
  senderTitle: 'General Practitioner'
})

const emit = defineEmits<{
  (e: 'update:selectedTemplate', value: string): void
  (e: 'update:selectedVerbosity', value: string): void
  (e: 'update:patientName', value: string): void
  (e: 'update:recipientDoctor', value: string): void
  (e: 'update:senderDoctor', value: string): void
  (e: 'update:senderTitle', value: string): void
  (e: 'update:soapOverrides', value: SoapOverrides): void
  (e: 'generate'): void
}>()

// Local state with two-way binding
const localSelectedTemplate = computed({
  get: () => props.selectedTemplate,
  set: (value) => emit('update:selectedTemplate', value)
})

const localSelectedVerbosity = computed({
  get: () => props.selectedVerbosity,
  set: (value) => emit('update:selectedVerbosity', value)
})

const localPatientName = computed({
  get: () => props.patientName,
  set: (value) => emit('update:patientName', value)
})

const localRecipientDoctor = computed({
  get: () => props.recipientDoctor,
  set: (value) => emit('update:recipientDoctor', value)
})

const localSenderDoctor = computed({
  get: () => props.senderDoctor,
  set: (value) => emit('update:senderDoctor', value)
})

const localSenderTitle = computed({
  get: () => props.senderTitle,
  set: (value) => emit('update:senderTitle', value)
})

// SOAP overrides with default
const defaultSoapOverrides: SoapOverrides = {
  subjective: { writingStyle: '', additionalInstructions: '' },
  objective: { writingStyle: '', additionalInstructions: '' },
  assessment: { writingStyle: '', additionalInstructions: '' },
  plan: { writingStyle: '', additionalInstructions: '' }
}

const localSoapOverrides = ref<SoapOverrides>(props.soapOverrides || { ...defaultSoapOverrides })

watch(localSoapOverrides, (newVal) => {
  emit('update:soapOverrides', newVal)
}, { deep: true })

watch(() => props.soapOverrides, (newVal) => {
  if (newVal) {
    localSoapOverrides.value = newVal
  }
}, { deep: true })

const showSectionOverrides = ref(false)

// Computed
const selectedTemplateName = computed(() => {
  return props.templates.find(t => t.key === localSelectedTemplate.value)?.name || 'Report'
})

const isGPLetterSelected = computed(() => {
  if (localSelectedTemplate.value === 'gp-letter-with-summary') return true
  const template = props.templates.find(t => t.key === localSelectedTemplate.value)
  return template?.category === 'letter'
})
const isSOAPSelected = computed(() => localSelectedTemplate.value === 'corti-soap')

// Verbosity options
const verbosityOptions = [
  { 
    key: 'concise', 
    name: 'Concise', 
    description: 'Brief, fact-based output focusing on key clinical findings (Default)',
    icon: 'mdi-text-short'
  },
  { 
    key: 'standard', 
    name: 'Standard', 
    description: 'Moderate detail with relevant history and context',
    icon: 'mdi-text'
  },
  { 
    key: 'detailed', 
    name: 'Detailed', 
    description: 'Comprehensive narrative output with full clinical documentation',
    icon: 'mdi-text-long'
  },
]

// Verbosity presets
const verbosityPresets: Record<string, SoapOverrides> = {
  concise: {
    subjective: { writingStyle: 'Concise summary in third person, focus on key clinical findings only', additionalInstructions: '' },
    objective: { writingStyle: 'Brief objective findings, essential data only', additionalInstructions: '' },
    assessment: { writingStyle: 'Concise clinical assessment with primary diagnosis', additionalInstructions: '' },
    plan: { writingStyle: 'Brief management plan with key action items', additionalInstructions: '' }
  },
  standard: {
    subjective: { writingStyle: 'Clear narrative in third person, include relevant history and context.', additionalInstructions: 'Include relevant history and context. Use bullet points for multiple findings.' },
    objective: { writingStyle: 'Complete examination findings with relevant diagnostic results.', additionalInstructions: 'Include all significant findings.' },
    assessment: { writingStyle: 'Clinical assessment with reasoning and differential considerations', additionalInstructions: 'Include primary diagnosis and key reasoning' },
    plan: { writingStyle: 'Detailed management plan with specific instructions.', additionalInstructions: 'Include specific action items with details.' }
  },
  detailed: {
    subjective: { writingStyle: 'Write in third person using bullet points.', additionalInstructions: 'Include reason for visit, symptom characteristics, changes since last visit, impact on activities, associated symptoms, patient concerns.' },
    objective: { writingStyle: 'Write using bullet points for each finding.', additionalInstructions: 'Include examination findings, vital signs, test results.' },
    assessment: { writingStyle: 'State the diagnosis clearly.', additionalInstructions: 'Include primary diagnosis, secondary diagnoses, status of condition.' },
    plan: { writingStyle: 'Write using bullet points for each action item.', additionalInstructions: 'Include counselling, treatment, medications, procedures, follow-up, safety netting.' }
  }
}

const applyVerbosityPreset = () => {
  const preset = verbosityPresets[localSelectedVerbosity.value]
  if (preset) {
    localSoapOverrides.value = JSON.parse(JSON.stringify(preset))
  }
}

// Watch for verbosity changes and apply preset
watch(localSelectedVerbosity, () => {
  applyVerbosityPreset()
}, { immediate: true })

// Expose for parent
defineExpose({
  selectedTemplateName,
  isGPLetterSelected,
  isSOAPSelected
})
</script>

<style scoped>
.report-generator {
  width: 100%;
}
</style>
