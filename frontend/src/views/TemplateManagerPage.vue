<template>
  <div class="template-manager-page">
    <v-row>
      <v-col cols="12">
        <div class="d-flex justify-space-between align-center mb-4">
          <div class="d-flex align-center">
            <v-icon icon="mdi-file-cog" class="mr-2" color="primary" size="28" />
            <h1 class="text-h5 font-weight-medium mb-0">Template Manager</h1>
          </div>
          <v-chip color="info" variant="tonal">
            <v-icon icon="mdi-shield-account" class="mr-1" size="16" />
            Admin Only
          </v-chip>
        </div>
      </v-col>
    </v-row>

    <!-- Loading State -->
    <v-row v-if="isLoading">
      <v-col cols="12" class="text-center py-12">
        <v-progress-circular indeterminate color="primary" size="48" />
        <p class="mt-4 text-body-1 text-medium-emphasis">Loading templates...</p>
      </v-col>
    </v-row>

    <!-- Main Content -->
    <v-row v-else>
      <!-- Template List -->
      <v-col cols="12" md="4">
        <v-card class="mb-4">
          <v-card-title class="d-flex align-center">
            <v-icon icon="mdi-format-list-bulleted" class="mr-2" />
            Custom Templates
          </v-card-title>
          <v-divider />
          <v-list density="compact">
            <v-list-item
              v-for="template in templates"
              :key="template.key"
              :active="selectedTemplate?.key === template.key"
              @click="selectTemplate(template)"
              class="template-list-item"
            >
              <template #prepend>
                <v-icon 
                  :icon="template.category === 'letter' ? 'mdi-email-outline' : 'mdi-file-document-outline'" 
                  :color="selectedTemplate?.key === template.key ? 'primary' : 'grey'"
                />
              </template>
              <v-list-item-title class="d-flex align-center">
                {{ template.name }}
                <v-chip
                  :color="template.isActive !== false ? 'success' : 'grey'"
                  size="x-small"
                  variant="tonal"
                  class="ml-2"
                >
                  {{ template.isActive !== false ? 'Active' : 'Inactive' }}
                </v-chip>
              </v-list-item-title>
              <v-list-item-subtitle>{{ template.key }}</v-list-item-subtitle>
            </v-list-item>
          </v-list>
          <v-divider />
          <!-- Hidden for now - uncomment to enable template creation
          <v-card-actions>
            <v-btn color="primary" variant="tonal" block @click="showCreateDialog = true">
              <v-icon icon="mdi-plus" class="mr-1" />
              New Template
            </v-btn>
          </v-card-actions>
          -->
        </v-card>
      </v-col>

      <!-- Template Editor -->
      <v-col cols="12" md="8">
        <v-card v-if="selectedTemplate">
          <v-card-title class="d-flex align-center justify-space-between">
            <div class="d-flex align-center">
              <v-icon icon="mdi-pencil" class="mr-2" />
              Edit: {{ selectedTemplate.name }}
            </div>
            <div class="d-flex ga-2">
              <!-- Hidden for now - uncomment to enable template deletion
              <v-btn
                color="error"
                variant="tonal"
                size="small"
                @click="confirmDelete"
              >
                <v-icon icon="mdi-delete" class="mr-1" />
                Delete
              </v-btn>
              -->
              <v-btn
                color="primary"
                size="small"
                :loading="isSaving"
                @click="saveTemplate"
              >
                <v-icon icon="mdi-content-save" class="mr-1" />
                Save
              </v-btn>
            </div>
          </v-card-title>
          <v-divider />

          <v-card-text>
            <!-- Template Basic Info -->
            <v-expansion-panels v-model="expandedPanel" class="mb-4">
              <v-expansion-panel value="basic">
                <v-expansion-panel-title>
                  <v-icon icon="mdi-information-outline" class="mr-2" />
                  Basic Information
                </v-expansion-panel-title>
                <v-expansion-panel-text>
                  <v-row dense>
                    <v-col cols="12" md="6">
                      <v-text-field
                        v-model="editForm.name"
                        label="Template Name"
                        density="compact"
                        variant="outlined"
                      />
                    </v-col>
                    <v-col cols="12" md="6">
                      <v-select
                        v-model="editForm.category"
                        :items="categoryOptions"
                        label="Category"
                        density="compact"
                        variant="outlined"
                      />
                    </v-col>
                    <v-col cols="12">
                      <v-text-field
                        v-model="editForm.description"
                        label="Description"
                        density="compact"
                        variant="outlined"
                      />
                    </v-col>
                    <v-col cols="12" md="6">
                      <v-select
                        v-model="editForm.documentationMode"
                        :items="documentationModeOptions"
                        label="Documentation Mode"
                        density="compact"
                        variant="outlined"
                      />
                    </v-col>
                    <v-col cols="12" md="6">
                      <v-select
                        v-model="editForm.outputLanguage"
                        :items="languageOptions"
                        label="Output Language"
                        density="compact"
                        variant="outlined"
                      />
                    </v-col>
                    <v-col cols="12">
                      <v-card variant="outlined" class="pa-3">
                        <div class="d-flex align-center justify-space-between">
                          <div>
                            <div class="d-flex align-center">
                              <v-icon 
                                :icon="editForm.isActive !== false ? 'mdi-check-circle' : 'mdi-close-circle'" 
                                :color="editForm.isActive !== false ? 'success' : 'grey'" 
                                class="mr-2" 
                              />
                              <span class="text-subtitle-2 font-weight-medium">Template Status</span>
                            </div>
                            <p class="text-caption text-medium-emphasis mt-1 mb-0">
                              {{ editForm.isActive !== false 
                                ? 'Active - This template is visible to users in the document type dropdown' 
                                : 'Inactive - This template is hidden from users' 
                              }}
                            </p>
                          </div>
                          <v-switch
                            v-model="editForm.isActive"
                            :true-value="true"
                            :false-value="false"
                            color="success"
                            hide-details
                            density="compact"
                          />
                        </div>
                      </v-card>
                    </v-col>
                  </v-row>
                </v-expansion-panel-text>
              </v-expansion-panel>
            </v-expansion-panels>

            <!-- Sections Editor -->
            <div class="d-flex align-center mb-3">
              <v-icon icon="mdi-view-list" class="mr-2" color="primary" />
              <span class="text-subtitle-1 font-weight-medium">Section Overrides</span>
              <v-spacer />
              <!-- Hidden for now - uncomment to enable adding sections
              <v-btn size="small" variant="tonal" @click="addSection">
                <v-icon icon="mdi-plus" class="mr-1" />
                Add Section
              </v-btn>
              -->
            </div>

            <v-alert type="info" variant="tonal" density="compact" class="mb-4">
              <span class="text-caption">
                Configure each section's output. Use placeholders like <code v-pre>{{PATIENT_NAME}}</code>, <code v-pre>{{RECIPIENT_DOCTOR}}</code>, <code v-pre>{{SENDER_DOCTOR}}</code>, <code v-pre>{{SENDER_TITLE}}</code> in instructions.
              </span>
            </v-alert>

            <!-- Section Panels (Collapsible) -->
            <v-expansion-panels v-model="expandedSections" multiple variant="accordion">
              <v-expansion-panel
                v-for="(section, index) in editForm.template.sections"
                :key="index"
                :value="index"
                class="mb-2"
              >
                <v-expansion-panel-title class="py-3">
                  <div class="d-flex align-center">
                    <v-icon 
                      :icon="getSectionIcon(section.key)" 
                      :color="getSectionColor(index)" 
                      class="mr-3" 
                      size="20"
                    />
                    <div>
                      <span class="text-subtitle-2 font-weight-medium">
                        {{ section.nameOverride || getSectionName(section.key) }}
                      </span>
                      <div class="text-caption text-medium-emphasis">
                        {{ section.key }}
                      </div>
                    </div>
                  </div>
                  <!-- Hidden for now - uncomment to enable removing sections
                  <template #actions>
                    <v-btn
                      icon
                      variant="text"
                      size="x-small"
                      color="error"
                      @click.stop="removeSection(index)"
                    >
                      <v-icon icon="mdi-close" size="18" />
                    </v-btn>
                  </template>
                  -->
                </v-expansion-panel-title>
                <v-expansion-panel-text>
                  <v-row dense class="pt-2">
                    <v-col cols="12" md="6">
                      <v-select
                        v-model="section.key"
                        :items="availableSections"
                        item-title="name"
                        item-value="key"
                        label="Section Type"
                        density="compact"
                        variant="outlined"
                      >
                        <template #item="{ item, props }">
                          <v-list-item v-bind="props">
                            <template #subtitle>{{ item.raw.description }}</template>
                          </v-list-item>
                        </template>
                      </v-select>
                    </v-col>
                    <v-col cols="12" md="6">
                      <v-text-field
                        v-model="section.nameOverride"
                        label="Display Name Override"
                        density="compact"
                        variant="outlined"
                        placeholder="e.g., Patient History"
                      />
                    </v-col>
                    <v-col cols="12">
                      <v-textarea
                        v-model="section.contentOverride"
                        label="Content Override"
                        density="compact"
                        variant="outlined"
                        rows="2"
                        placeholder="e.g., Include: symptoms, history. Exclude: examination findings."
                        hint="Specify what to include/exclude in this section"
                        persistent-hint
                      />
                    </v-col>
                    <v-col cols="12" md="6">
                      <v-select
                        v-model="section.writingStyleOverride"
                        :items="writingStyleOptions"
                        label="Writing Style"
                        density="compact"
                        variant="outlined"
                        clearable
                      />
                    </v-col>
                    <v-col cols="12" md="6">
                      <v-text-field
                        v-model="section.formatRuleOverride"
                        label="Format Rule"
                        density="compact"
                        variant="outlined"
                        placeholder="e.g., Use numbered lists"
                      />
                    </v-col>
                    <v-col cols="12">
                      <v-textarea
                        v-model="section.additionalInstructionsOverride"
                        label="Additional Instructions"
                        density="compact"
                        variant="outlined"
                        rows="3"
                        placeholder="Additional prompts for this section..."
                        hint="Custom instructions for AI to follow when generating this section"
                        persistent-hint
                      />
                    </v-col>
                  </v-row>
                </v-expansion-panel-text>
              </v-expansion-panel>
            </v-expansion-panels>
          </v-card-text>
        </v-card>

        <!-- No Template Selected -->
        <v-card v-else class="text-center py-12">
          <v-icon icon="mdi-file-document-edit-outline" size="80" color="grey-lighten-1" />
          <p class="text-h6 text-medium-emphasis mt-4">Select a template to edit</p>
          <p class="text-body-2 text-medium-emphasis">
            Choose a template from the list or create a new one
          </p>
        </v-card>
      </v-col>
    </v-row>

    <!-- Create Template Dialog -->
    <v-dialog v-model="showCreateDialog" max-width="500">
      <v-card>
        <v-card-title>
          <v-icon icon="mdi-plus-circle" class="mr-2" />
          Create New Template
        </v-card-title>
        <v-divider />
        <v-card-text>
          <v-text-field
            v-model="newTemplateForm.key"
            label="Template Key"
            density="compact"
            variant="outlined"
            placeholder="e.g., my-custom-template"
            hint="Unique identifier (lowercase, hyphens allowed)"
            persistent-hint
            class="mb-3"
          />
          <v-text-field
            v-model="newTemplateForm.name"
            label="Template Name"
            density="compact"
            variant="outlined"
            placeholder="e.g., My Custom Template"
            class="mb-3"
          />
          <v-select
            v-model="newTemplateForm.category"
            :items="categoryOptions"
            label="Category"
            density="compact"
            variant="outlined"
            class="mb-3"
          />
          <v-text-field
            v-model="newTemplateForm.description"
            label="Description"
            density="compact"
            variant="outlined"
            placeholder="Brief description of the template"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="showCreateDialog = false">Cancel</v-btn>
          <v-btn color="primary" @click="createTemplate">Create</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="showDeleteDialog" max-width="400">
      <v-card>
        <v-card-title class="d-flex align-center">
          <v-icon icon="mdi-alert" color="error" class="mr-2" />
          Confirm Delete
        </v-card-title>
        <v-card-text>
          Are you sure you want to delete the template "{{ selectedTemplate?.name }}"?
          This action cannot be undone.
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="showDeleteDialog = false">Cancel</v-btn>
          <v-btn color="error" @click="deleteTemplate">Delete</v-btn>
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
import { ref, reactive, onMounted } from 'vue'
import api from '@/services/api'

interface TemplateSection {
  key: string
  nameOverride?: string
  contentOverride?: string
  writingStyleOverride?: string
  formatRuleOverride?: string
  additionalInstructionsOverride?: string
}

interface CustomTemplate {
  key: string
  name: string
  category: string
  description: string
  documentationMode: string
  outputLanguage: string
  template: {
    sections: TemplateSection[]
  }
  isActive?: boolean
  createdAt?: string
  updatedAt?: string
}

interface AvailableSection {
  key: string
  name: string
  description: string
}

// State
const isLoading = ref(true)
const isSaving = ref(false)
const templates = ref<CustomTemplate[]>([])
const selectedTemplate = ref<CustomTemplate | null>(null)
const availableSections = ref<AvailableSection[]>([])
const expandedPanel = ref<string | undefined>('basic')
const expandedSections = ref<number[]>([])

// Dialogs
const showCreateDialog = ref(false)
const showDeleteDialog = ref(false)
const showSuccess = ref(false)
const showError = ref(false)
const successMessage = ref('')
const errorMessage = ref('')

// Form data
const editForm = reactive<CustomTemplate>({
  key: '',
  name: '',
  category: 'note',
  description: '',
  documentationMode: 'routed_parallel',
  outputLanguage: 'en',
  template: {
    sections: []
  },
  isActive: true
})

const newTemplateForm = reactive({
  key: '',
  name: '',
  category: 'note',
  description: ''
})

// Options
const categoryOptions = ['note', 'letter', 'report', 'summary']
const documentationModeOptions = [
  { title: 'Routed Parallel', value: 'routed_parallel' },
  { title: 'Sequential', value: 'sequential' },
  { title: 'Standard', value: 'standard' }
]
const languageOptions = [
  { title: 'English', value: 'en' },
  { title: 'English (UK)', value: 'en-GB' },
  { title: 'English (US)', value: 'en-US' }
]
const writingStyleOptions = ['formal', 'concise', 'detailed', 'narrative', 'bullet-points']

// Section display helpers
const sectionIconMap: Record<string, string> = {
  'corti-subjective': 'mdi-account-voice',
  'corti-objective': 'mdi-stethoscope',
  'corti-assessment': 'mdi-clipboard-text',
  'corti-plan': 'mdi-clipboard-check',
  'corti-diagnoses': 'mdi-medical-bag',
  'corti-medications': 'mdi-pill',
  'corti-investigations': 'mdi-test-tube',
  'corti-notes': 'mdi-note-text',
  'corti-summary': 'mdi-text-box',
  'corti-referral': 'mdi-account-arrow-right'
}

const sectionColors = ['primary', 'secondary', 'success', 'info', 'warning', 'error', 'indigo', 'teal']

const getSectionIcon = (key: string): string => {
  return sectionIconMap[key] || 'mdi-file-document-outline'
}

const getSectionColor = (index: number): string => {
  return sectionColors[index % sectionColors.length]
}

const getSectionName = (key: string): string => {
  const section = availableSections.value.find(s => s.key === key)
  return section?.name || key
}

// Load data on mount
onMounted(async () => {
  await Promise.all([loadTemplates(), loadAvailableSections()])
  isLoading.value = false
})

// API calls
const loadTemplates = async () => {
  try {
    const response = await api.get('/templates/custom')
    if (response.data.success) {
      templates.value = response.data.templates || []
    }
  } catch (err) {
    console.error('Failed to load templates:', err)
    showError.value = true
    errorMessage.value = 'Failed to load templates'
  }
}

const loadAvailableSections = async () => {
  try {
    const response = await api.get('/templates/custom/sections')
    if (response.data.success) {
      availableSections.value = response.data.sections || []
    }
  } catch (err) {
    console.error('Failed to load sections:', err)
  }
}

const selectTemplate = (template: CustomTemplate) => {
  selectedTemplate.value = template
  // Deep copy to edit form
  Object.assign(editForm, JSON.parse(JSON.stringify(template)))
}

const saveTemplate = async () => {
  if (!selectedTemplate.value) return

  isSaving.value = true
  try {
    const response = await api.put(`/templates/custom/${editForm.key}`, editForm)
    if (response.data.success) {
      // Update local state
      const index = templates.value.findIndex(t => t.key === editForm.key)
      if (index !== -1) {
        templates.value[index] = response.data.template
      }
      selectedTemplate.value = response.data.template
      showSuccess.value = true
      successMessage.value = 'Template saved successfully!'
    }
  } catch (err: any) {
    showError.value = true
    errorMessage.value = err.response?.data?.error || 'Failed to save template'
  } finally {
    isSaving.value = false
  }
}

const createTemplate = async () => {
  try {
    const newTemplate: CustomTemplate = {
      key: newTemplateForm.key,
      name: newTemplateForm.name,
      category: newTemplateForm.category,
      description: newTemplateForm.description,
      documentationMode: 'routed_parallel',
      outputLanguage: 'en',
      template: {
        sections: [
          {
            key: 'corti-subjective',
            nameOverride: '',
            contentOverride: '',
            writingStyleOverride: 'formal',
            formatRuleOverride: '',
            additionalInstructionsOverride: ''
          }
        ]
      }
    }

    const response = await api.post('/templates/custom', newTemplate)
    if (response.data.success) {
      templates.value.push(response.data.template)
      selectTemplate(response.data.template)
      showCreateDialog.value = false
      // Reset form
      newTemplateForm.key = ''
      newTemplateForm.name = ''
      newTemplateForm.category = 'note'
      newTemplateForm.description = ''
      showSuccess.value = true
      successMessage.value = 'Template created successfully!'
    }
  } catch (err: any) {
    showError.value = true
    errorMessage.value = err.response?.data?.error || 'Failed to create template'
  }
}

const deleteTemplate = async () => {
  if (!selectedTemplate.value) return

  try {
    const response = await api.delete(`/templates/custom/${selectedTemplate.value.key}`)
    if (response.data.success) {
      templates.value = templates.value.filter(t => t.key !== selectedTemplate.value?.key)
      selectedTemplate.value = null
      showDeleteDialog.value = false
      showSuccess.value = true
      successMessage.value = 'Template deleted successfully!'
    }
  } catch (err: any) {
    showError.value = true
    errorMessage.value = err.response?.data?.error || 'Failed to delete template'
  }
}

/* Hidden for now - uncomment when enabling add/remove section features
const addSection = () => {
  editForm.template.sections.push({
    key: 'corti-subjective',
    nameOverride: '',
    contentOverride: '',
    writingStyleOverride: '',
    formatRuleOverride: '',
    additionalInstructionsOverride: ''
  })
}

const removeSection = (index: number) => {
  editForm.template.sections.splice(index, 1)
}
*/
</script>

<style scoped>
.template-manager-page {
  max-width: 1400px;
  margin: 0 auto;
  padding: 16px;
}

.template-list-item {
  cursor: pointer;
}

.section-card {
  transition: box-shadow 0.2s;
}

.section-card:hover {
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
}

.drag-handle {
  cursor: grab;
}

code {
  background-color: rgba(0, 0, 0, 0.05);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 0.85em;
}
</style>
