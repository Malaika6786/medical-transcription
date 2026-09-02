<template>
  <v-dialog v-model="isOpen" max-width="1080" scrollable>
    <v-card class="glass-card">
      <v-card-title class="d-flex align-center">
        <v-icon icon="mdi-file-document-arrow-right-outline" class="mr-2" color="secondary" />
        Generate Report
      </v-card-title>

      <v-divider />

      <v-card-text>
        <v-row>
          <!-- Controls -->
          <v-col cols="12" md="5">
            <v-select
              v-model="templateKey"
              :items="REPORT_TEMPLATES"
              item-title="name"
              item-value="key"
              label="Report template"
              density="comfortable"
              variant="outlined"
              hide-details
              class="mb-1"
            >
              <template v-slot:item="{ props: itemProps, item }">
                <v-list-item v-bind="itemProps" :subtitle="item.raw.description" />
              </template>
            </v-select>
            <p class="text-caption text-medium-emphasis mb-4">{{ activeTemplate.description }}</p>

            <div class="d-flex align-center justify-space-between mb-1">
              <span class="text-subtitle-2 font-weight-medium">Include sections</span>
              <div>
                <v-btn size="x-small" variant="text" @click="selectAll">All</v-btn>
                <v-btn size="x-small" variant="text" @click="selectNone">None</v-btn>
              </div>
            </div>

            <v-row dense class="mb-2">
              <v-col v-for="choice in choices" :key="choice.key" cols="12" sm="6">
                <v-checkbox
                  v-model="selectedKeys"
                  :value="choice.key"
                  :disabled="!choice.selectable"
                  density="compact"
                  hide-details
                  color="secondary"
                >
                  <template v-slot:label>
                    <span class="text-body-2">{{ choice.label }}</span>
                    <span v-if="choice.empty" class="text-caption text-disabled ml-1">· empty</span>
                  </template>
                </v-checkbox>
              </v-col>
            </v-row>

            <v-switch
              v-if="chat.length"
              v-model="includeQa"
              color="secondary"
              density="compact"
              hide-details
              class="mb-2"
            >
              <template v-slot:label>
                <span class="text-body-2">
                  Append follow-up Q&amp;A
                  <span class="text-caption text-medium-emphasis">({{ qaTurnCount }})</span>
                </span>
              </template>
            </v-switch>
          </v-col>

          <!-- Preview -->
          <v-col cols="12" md="7">
            <div class="d-flex align-center mb-2">
              <v-icon icon="mdi-eye-outline" size="18" class="mr-2" color="secondary" />
              <span class="text-subtitle-2 font-weight-medium">Preview</span>
              <v-chip size="x-small" variant="tonal" class="ml-2">{{ sectionCount }} sections</v-chip>
            </div>

            <v-sheet class="report-preview" rounded border>
              <h2 class="preview-title">{{ doc.meta.title }}</h2>

              <dl class="preview-meta">
                <template v-for="row in metaRowsForPreview" :key="row.label">
                  <dt>{{ row.label }}</dt>
                  <dd>{{ row.value }}</dd>
                </template>
              </dl>

              <p v-if="!doc.groups.length" class="preview-placeholder">
                No sections selected — tick at least one to produce a report.
              </p>

              <div v-for="(group, gi) in doc.groups" :key="gi">
                <h3 v-if="group.heading" class="preview-group">{{ group.heading }}</h3>
                <div v-for="block in group.blocks" :key="block.key" class="preview-block">
                  <h4 class="preview-section">{{ block.heading }}</h4>
                  <p v-if="block.empty" class="preview-placeholder">{{ EMPTY_PLACEHOLDER }}</p>
                  <p v-else-if="block.kind === 'paragraph'" class="preview-body">{{ block.text }}</p>
                  <ul v-else class="preview-list">
                    <li v-for="(item, i) in block.items" :key="i">{{ item }}</li>
                  </ul>
                </div>
              </div>

              <template v-if="doc.qa.length">
                <h3 class="preview-group">Follow-up Questions</h3>
                <p v-for="(turn, i) in doc.qa" :key="i" class="preview-body">
                  <strong>{{ turn.role === 'user' ? 'Q:' : 'A:' }}</strong> {{ turn.content }}
                </p>
              </template>

              <p class="preview-disclaimer">{{ doc.disclaimer }}</p>
            </v-sheet>
          </v-col>
        </v-row>

        <v-alert v-if="error" type="error" variant="tonal" density="compact" class="mt-3 mb-0">
          {{ error }}
        </v-alert>
      </v-card-text>

      <v-divider />

      <v-card-actions class="pa-4">
        <v-select
          v-model="format"
          :items="REPORT_FORMATS"
          item-title="label"
          item-value="value"
          label="Format"
          density="compact"
          variant="outlined"
          hide-details
          style="max-width: 200px"
        >
          <template v-slot:item="{ props: itemProps, item }">
            <v-list-item v-bind="itemProps" :prepend-icon="item.raw.icon" />
          </template>
        </v-select>

        <v-spacer />

        <v-btn variant="text" :disabled="isExporting" @click="isOpen = false">Cancel</v-btn>
        <v-btn
          color="secondary"
          variant="flat"
          prepend-icon="mdi-download"
          :loading="isExporting"
          :disabled="!doc.groups.length"
          @click="runExport"
        >
          Export
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { AiChatMessage, AiExtraction } from '@/composables/useAiAssistant'
import {
  DEFAULT_TEMPLATE_KEY,
  EMPTY_PLACEHOLDER,
  REPORT_FORMATS,
  REPORT_TEMPLATES,
  buildReportDoc,
  defaultSelection,
  exportReport,
  getTemplate,
  metaRows,
  sectionChoices,
} from '@/services/report'
import type { ExtractionKey, ReportFormat, ReportMetaSource } from '@/services/report'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    extraction: AiExtraction
    chat?: AiChatMessage[]
    meta?: ReportMetaSource
  }>(),
  { chat: () => [], meta: () => ({}) }
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  exported: [filename: string]
}>()

const isOpen = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const templateKey = ref(DEFAULT_TEMPLATE_KEY)
const selectedKeys = ref<ExtractionKey[]>([])
const includeQa = ref(true)
const format = ref<ReportFormat>('pdf')
const isExporting = ref(false)
const error = ref('')
/** Frozen when the dialog opens so the preview and the file agree. */
const generatedAt = ref('')

const chat = computed(() => props.chat)
const activeTemplate = computed(() => getTemplate(templateKey.value))
const choices = computed(() => sectionChoices(activeTemplate.value, props.extraction))

const formatTimestamp = (value: Date) =>
  value.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })

const consultationDate = computed(() => {
  const raw = props.meta.consultationDate
  if (!raw) return undefined
  return raw instanceof Date ? formatTimestamp(raw) : raw
})

const doc = computed(() =>
  buildReportDoc({
    extraction: props.extraction,
    templateKey: templateKey.value,
    selectedKeys: selectedKeys.value,
    chat: chat.value,
    includeQa: includeQa.value,
    meta: {
      title: props.meta.title || 'Consultation Report',
      patientName: props.meta.patientName,
      clinician: props.meta.clinician,
      consultationDate: consultationDate.value,
      sessionId: props.meta.sessionId ?? undefined,
      generatedAt: generatedAt.value,
    },
  })
)

const metaRowsForPreview = computed(() => metaRows(doc.value.meta))
const sectionCount = computed(() => doc.value.groups.reduce((n, group) => n + group.blocks.length, 0))
const qaTurnCount = computed(() => `${chat.value.length} turn${chat.value.length === 1 ? '' : 's'}`)

const resetSelection = () => {
  selectedKeys.value = defaultSelection(activeTemplate.value, props.extraction)
}

const selectAll = () => {
  selectedKeys.value = choices.value.filter(choice => choice.selectable).map(choice => choice.key)
}

const selectNone = () => {
  selectedKeys.value = []
}

watch(templateKey, resetSelection)

// immediate, so a dialog that is already open when mounted is seeded too —
// without it the section selection and the timestamp stay unset.
watch(
  isOpen,
  open => {
    if (!open) return
    error.value = ''
    generatedAt.value = formatTimestamp(new Date())
    includeQa.value = chat.value.length > 0
    resetSelection()
  },
  { immediate: true }
)

const runExport = async () => {
  isExporting.value = true
  error.value = ''
  try {
    const filename = await exportReport(doc.value, format.value)
    emit('exported', filename)
    isOpen.value = false
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to generate the report'
  } finally {
    isExporting.value = false
  }
}
</script>

<style scoped>
.report-preview {
  max-height: 420px;
  overflow-y: auto;
  padding: 20px 24px;
  background: rgba(var(--v-theme-surface-variant), 0.35);
}

.preview-title {
  font-size: 1.15rem;
  font-weight: 600;
  margin: 0 0 12px;
}

.preview-meta {
  display: grid;
  grid-template-columns: auto 1fr;
  column-gap: 12px;
  row-gap: 2px;
  font-size: 0.75rem;
  opacity: 0.75;
  margin: 0 0 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(var(--v-border-color), 0.25);
}

.preview-meta dt {
  font-weight: 600;
}

.preview-meta dd {
  margin: 0;
}

.preview-group {
  font-size: 0.95rem;
  font-weight: 600;
  margin: 18px 0 6px;
}

.preview-block {
  margin-bottom: 12px;
}

.preview-section {
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  opacity: 0.7;
  margin: 0 0 4px;
}

.preview-body {
  font-size: 0.875rem;
  margin: 0 0 6px;
  white-space: pre-wrap;
}

.preview-list {
  font-size: 0.875rem;
  margin: 0;
  padding-left: 1.25rem;
}

.preview-list li {
  margin-bottom: 2px;
}

.preview-placeholder {
  font-size: 0.875rem;
  font-style: italic;
  opacity: 0.6;
  margin: 0 0 6px;
}

.preview-disclaimer {
  font-size: 0.7rem;
  opacity: 0.6;
  margin: 20px 0 0;
  padding-top: 12px;
  border-top: 1px solid rgba(var(--v-border-color), 0.25);
}
</style>
