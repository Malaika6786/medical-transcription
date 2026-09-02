<template>
  <div class="corti-sections-page">
    <v-row>
      <v-col cols="12">
        <div class="d-flex align-center mb-4">
          <v-icon icon="mdi-format-list-bulleted-type" class="mr-2" color="primary" size="24" />
          <h1 class="text-h6 font-weight-medium mb-0">Corti Template Sections</h1>
          <v-chip color="warning" variant="tonal" size="small" class="ml-3">Admin</v-chip>
          <v-spacer />
          <v-btn
            variant="outlined"
            size="small"
            :loading="isLoading"
            @click="loadSections"
          >
            <v-icon icon="mdi-refresh" class="mr-1" />
            Refresh
          </v-btn>
        </div>
        <p class="text-body-2 text-medium-emphasis mb-4">
          Live template sections fetched directly from Corti API. Use these section keys when building custom templates.
        </p>
      </v-col>
    </v-row>

    <!-- Language Filter -->
    <v-row>
      <v-col cols="12" md="4">
        <v-select
          v-model="selectedLanguage"
          :items="languageOptions"
          item-title="name"
          item-value="code"
          label="Filter by Language"
          variant="outlined"
          density="compact"
          clearable
          prepend-inner-icon="mdi-translate"
          @update:model-value="loadSections"
        />
      </v-col>
      <v-col cols="12" md="4">
        <v-text-field
          v-model="searchQuery"
          label="Search sections"
          variant="outlined"
          density="compact"
          clearable
          prepend-inner-icon="mdi-magnify"
        />
      </v-col>
      <v-col cols="12" md="4" class="d-flex align-center">
        <v-chip color="primary" variant="tonal">
          {{ filteredSections.length }} sections
        </v-chip>
      </v-col>
    </v-row>

    <!-- Loading State -->
    <v-row v-if="isLoading">
      <v-col cols="12" class="text-center py-8">
        <v-progress-circular indeterminate color="primary" size="48" />
        <p class="text-body-2 text-medium-emphasis mt-4">Fetching sections from Corti API...</p>
      </v-col>
    </v-row>

    <!-- Error State -->
    <v-row v-else-if="error">
      <v-col cols="12">
        <v-alert type="error" variant="tonal">
          {{ error }}
        </v-alert>
      </v-col>
    </v-row>

    <!-- Sections List -->
    <v-row v-else>
      <v-col cols="12">
        <v-expansion-panels variant="accordion">
          <v-expansion-panel
            v-for="section in filteredSections"
            :key="section.key"
          >
            <v-expansion-panel-title>
              <div class="d-flex align-center flex-grow-1">
                <v-icon :icon="getSectionIcon(section.key)" color="primary" class="mr-3" />
                <div>
                  <div class="text-subtitle-2 font-weight-medium">{{ section.name }}</div>
                  <code class="text-caption">{{ section.key }}</code>
                </div>
                <v-spacer />
                <v-chip
                  v-if="section.documentationMode"
                  size="x-small"
                  variant="tonal"
                  color="secondary"
                  class="mr-2"
                >
                  {{ section.documentationMode }}
                </v-chip>
                <v-chip
                  size="x-small"
                  variant="outlined"
                  class="mr-4"
                >
                  {{ section.type }}
                </v-chip>
              </div>
            </v-expansion-panel-title>
            <v-expansion-panel-text>
              <v-row dense>
                <!-- Description -->
                <v-col cols="12">
                  <div class="text-body-2 mb-3">{{ section.description }}</div>
                </v-col>

                <!-- Writing Style & Format Rule -->
                <v-col cols="12" md="6">
                  <v-card variant="outlined" class="pa-3">
                    <div class="text-caption text-medium-emphasis mb-1">Default Writing Style</div>
                    <div class="text-body-2 font-weight-medium">
                      {{ section.defaultWritingStyle?.name || 'Not specified' }}
                    </div>
                  </v-card>
                </v-col>
                <v-col cols="12" md="6">
                  <v-card variant="outlined" class="pa-3">
                    <div class="text-caption text-medium-emphasis mb-1">Default Format Rule</div>
                    <div class="text-body-2 font-weight-medium">
                      {{ section.defaultFormatRule?.name || 'Not specified' }}
                    </div>
                  </v-card>
                </v-col>

                <!-- Content -->
                <v-col v-if="section.content" cols="12">
                  <v-card variant="outlined" class="pa-3 mt-2">
                    <div class="text-caption text-medium-emphasis mb-1">Content Guidelines</div>
                    <div class="text-body-2">{{ section.content }}</div>
                  </v-card>
                </v-col>

                <!-- Additional Instructions -->
                <v-col v-if="section.additionalInstructions" cols="12">
                  <v-card variant="outlined" class="pa-3 mt-2">
                    <div class="text-caption text-medium-emphasis mb-1">Additional Instructions</div>
                    <div class="text-body-2">{{ section.additionalInstructions }}</div>
                  </v-card>
                </v-col>

                <!-- Translations -->
                <v-col v-if="section.translations && section.translations.length > 0" cols="12">
                  <v-card variant="outlined" class="pa-3 mt-2">
                    <div class="text-caption text-medium-emphasis mb-2">Available Translations</div>
                    <div class="d-flex flex-wrap ga-2">
                      <v-chip
                        v-for="trans in section.translations"
                        :key="trans.languageId"
                        size="small"
                        variant="tonal"
                      >
                        {{ trans.languageId.toUpperCase() }}
                        <span v-if="trans.name" class="ml-1 text-medium-emphasis">- {{ trans.name }}</span>
                      </v-chip>
                    </div>
                  </v-card>
                </v-col>

                <!-- Copy Key Button -->
                <v-col cols="12" class="mt-2">
                  <v-btn
                    variant="tonal"
                    size="small"
                    @click="copyKey(section.key)"
                  >
                    <v-icon icon="mdi-content-copy" class="mr-1" />
                    Copy Section Key
                  </v-btn>
                </v-col>
              </v-row>
            </v-expansion-panel-text>
          </v-expansion-panel>
        </v-expansion-panels>
      </v-col>
    </v-row>

    <!-- Success Snackbar -->
    <v-snackbar v-model="showSuccess" color="success" :timeout="2000">
      {{ successMessage }}
    </v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import api from '@/services/api'

interface WritingStyle {
  name: string
}

interface FormatRule {
  name: string
}

interface Translation {
  languageId: string
  name?: string
  description?: string
}

interface CortiSection {
  name: string
  alternateName?: string
  key: string
  description: string
  defaultWritingStyle: WritingStyle
  defaultFormatRule?: FormatRule
  additionalInstructions?: string
  content?: string
  documentationMode?: string
  type: string
  translations: Translation[]
  updatedAt?: string
}

// State
const sections = ref<CortiSection[]>([])
const isLoading = ref(false)
const error = ref('')
const selectedLanguage = ref('')
const searchQuery = ref('')
const showSuccess = ref(false)
const successMessage = ref('')

const languageOptions = [
  { name: 'All Languages', code: '' },
  { name: 'English', code: 'en' },
  { name: 'German', code: 'de' },
  { name: 'Spanish', code: 'es' },
  { name: 'French', code: 'fr' },
  { name: 'Italian', code: 'it' },
  { name: 'Dutch', code: 'nl' },
  { name: 'Portuguese', code: 'pt' },
]

// Computed
const filteredSections = computed(() => {
  if (!searchQuery.value) return sections.value
  
  const query = searchQuery.value.toLowerCase()
  return sections.value.filter(s => 
    s.name.toLowerCase().includes(query) ||
    s.key.toLowerCase().includes(query) ||
    s.description.toLowerCase().includes(query)
  )
})

// Methods
const loadSections = async () => {
  isLoading.value = true
  error.value = ''
  
  try {
    const params = selectedLanguage.value ? `?lang=${selectedLanguage.value}` : ''
    const response = await api.get(`/templates/corti-sections${params}`)
    
    if (response.data.success) {
      sections.value = response.data.sections || []
    } else {
      throw new Error(response.data.error || 'Failed to fetch sections')
    }
  } catch (err: any) {
    console.error('Failed to load sections:', err)
    error.value = err.response?.data?.error || err.message || 'Failed to fetch sections from Corti API'
  } finally {
    isLoading.value = false
  }
}

const getSectionIcon = (key: string): string => {
  const iconMap: Record<string, string> = {
    'corti-subjective': 'mdi-account-voice',
    'corti-objective': 'mdi-stethoscope',
    'corti-assessment': 'mdi-clipboard-text',
    'corti-plan': 'mdi-clipboard-check',
    'corti-diagnoses': 'mdi-medical-bag',
    'corti-medications': 'mdi-pill',
    'corti-allergies': 'mdi-alert-circle',
    'corti-vital-signs': 'mdi-heart-pulse',
    'corti-hpi': 'mdi-history',
    'corti-chief-complaint': 'mdi-account-question',
    'corti-diagnostic-results': 'mdi-test-tube',
    'corti-physical-exam-with-vitals': 'mdi-human',
    'corti-referral': 'mdi-account-arrow-right',
    'corti-social-history': 'mdi-account-group',
    'corti-family-history': 'mdi-family-tree',
    'corti-past-medical-history': 'mdi-clipboard-pulse',
    'corti-review-of-systems': 'mdi-format-list-checks',
    'corti-patient-summary': 'mdi-text-box',
    'corti-discharge-summary': 'mdi-exit-to-app',
    'corti-brief-clinical-note': 'mdi-note-text',
  }
  return iconMap[key] || 'mdi-file-document-outline'
}

const copyKey = async (key: string) => {
  try {
    await navigator.clipboard.writeText(key)
    showSuccess.value = true
    successMessage.value = `Copied: ${key}`
  } catch (err) {
    console.error('Failed to copy:', err)
  }
}

// Lifecycle
onMounted(() => {
  loadSections()
})
</script>

<style scoped>
.corti-sections-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 16px;
}
</style>
