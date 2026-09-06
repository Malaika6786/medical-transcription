<template>
  <v-card class="glass-card">
    <v-card-title class="d-flex justify-space-between align-center">
      <span>
        <v-icon icon="mdi-robot-outline" class="mr-2" color="secondary" />
        AI Assistance
        <v-chip size="x-small" color="secondary" variant="tonal" class="ml-2">Local model</v-chip>
      </span>
      <v-btn
        v-if="extraction"
        size="small"
        variant="text"
        prepend-icon="mdi-refresh"
        :loading="isSummarizing"
        @click="runSummarize"
      >
        Regenerate
      </v-btn>
    </v-card-title>

    <v-card-text>
      <DemoUsageBanner
        feature="ai_summary"
        label="AI Summary"
        description="Generates a structured clinical summary from the transcript."
      />

      <!-- Summarizing state -->
      <div v-if="isSummarizing" class="text-center py-8">
        <v-progress-circular indeterminate color="secondary" size="36" class="mb-3" />
        <p class="text-body-2 text-medium-emphasis mb-0">
          Analyzing the conversation… this runs on the configured model and can take a moment.
        </p>
      </div>

      <!-- Error state -->
      <v-alert v-else-if="error && !extraction" type="error" variant="tonal" density="compact" class="mb-3">
        {{ error }}
        <template v-slot:append>
          <v-btn size="small" variant="text" @click="runSummarize">Retry</v-btn>
        </template>
      </v-alert>

      <!-- Extraction -->
      <template v-if="extraction">
        <div class="mb-4">
          <div class="d-flex align-center mb-2">
            <v-icon icon="mdi-text-box-outline" size="18" class="mr-2" color="secondary" />
            <span class="text-subtitle-2 font-weight-medium">Summary</span>
          </div>
          <p class="text-body-2 mb-0">{{ extraction.summary }}</p>
        </div>

        <div v-if="extraction.chiefComplaint" class="mb-4">
          <div class="d-flex align-center mb-2">
            <v-icon icon="mdi-account-voice" size="18" class="mr-2" color="secondary" />
            <span class="text-subtitle-2 font-weight-medium">Chief Complaint</span>
          </div>
          <p class="text-body-2 mb-0">{{ extraction.chiefComplaint }}</p>
        </div>

        <v-row class="mb-1">
          <v-col
            v-for="section in visibleChipSections"
            :key="section.label"
            cols="12"
            sm="6"
          >
            <div class="d-flex align-center mb-2">
              <v-icon :icon="section.icon" size="18" class="mr-2" :color="section.color" />
              <span class="text-subtitle-2 font-weight-medium">{{ section.label }}</span>
            </div>
            <template v-if="section.items.length">
              <v-chip
                v-for="(item, i) in section.items"
                :key="i"
                size="small"
                :color="section.color"
                variant="tonal"
                class="mr-1 mb-1"
              >
                {{ item }}
              </v-chip>
            </template>
            <p v-else class="text-body-2 text-medium-emphasis mb-0">None mentioned</p>
          </v-col>
        </v-row>

        <div v-for="section in visibleListSections" :key="section.label" class="mb-4">
          <div class="d-flex align-center mb-2">
            <v-icon :icon="section.icon" size="18" class="mr-2" color="secondary" />
            <span class="text-subtitle-2 font-weight-medium">{{ section.label }}</span>
          </div>
          <ul class="key-points-list text-body-2">
            <li v-for="(item, i) in section.items" :key="i">{{ item }}</li>
          </ul>
        </div>

        <v-divider class="mb-4" />

        <!-- Grounded Q&A chat -->
        <div class="d-flex align-center mb-2">
          <v-icon icon="mdi-chat-question-outline" size="18" class="mr-2" color="secondary" />
          <span class="text-subtitle-2 font-weight-medium">Ask about this conversation</span>
        </div>

        <div v-if="chatMessages.length" ref="chatThreadRef" class="chat-thread mb-3">
          <div
            v-for="(msg, i) in chatMessages"
            :key="i"
            class="chat-message"
            :class="msg.role === 'user' ? 'chat-user' : 'chat-assistant'"
          >
            <div class="chat-bubble text-body-2">{{ msg.content }}</div>
          </div>
          <div v-if="isAsking" class="chat-message chat-assistant">
            <div class="chat-bubble text-body-2">
              <v-progress-circular indeterminate size="14" width="2" class="mr-2" />
              Thinking…
            </div>
          </div>
        </div>

        <v-alert v-if="error && extraction" type="error" variant="tonal" density="compact" class="mb-3">
          {{ error }}
        </v-alert>

        <v-text-field
          v-model="question"
          placeholder="e.g. What allergies were mentioned?"
          density="comfortable"
          hide-details
          :disabled="isAsking"
          @keydown.enter.prevent="submitQuestion"
        >
          <template v-slot:append-inner>
            <v-btn
              icon="mdi-send"
              size="small"
              variant="text"
              color="secondary"
              :loading="isAsking"
              :disabled="!question.trim()"
              @click="submitQuestion"
            />
          </template>
        </v-text-field>

        <!-- Report export -->
        <template v-if="allowReport">
          <v-divider class="my-4" />

          <v-btn
            block
            color="secondary"
            variant="tonal"
            prepend-icon="mdi-file-document-arrow-right-outline"
            @click="showReportDialog = true"
          >
            Generate Report
          </v-btn>

          <AiReportDialog
            v-model="showReportDialog"
            :extraction="extraction"
            :chat="chatMessages"
            :meta="reportMeta"
            @exported="onReportExported"
          />

          <v-snackbar v-model="showExportToast" :timeout="4000" color="success" location="bottom">
            Report downloaded — {{ exportedFilename }}
          </v-snackbar>
        </template>

        <p class="text-caption text-medium-emphasis mt-2 mb-0">
          <v-icon icon="mdi-information-outline" size="12" class="mr-1" />
          AI-generated documentation aid — answers come only from this conversation's extracted record. Verify clinically before use.
        </p>
      </template>
    </v-card-text>
  </v-card>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import { useAiAssistant } from '@/composables/useAiAssistant'
import type { AiExtraction } from '@/composables/useAiAssistant'
import AiReportDialog from '@/components/AiReportDialog.vue'
import DemoUsageBanner from '@/components/DemoUsageBanner.vue'
import type { ReportMetaSource } from '@/services/report'

const props = defineProps<{
  transcript: string
  language?: string
  /** Pre-existing extraction from a saved session — restored instead of re-running summarize. */
  initialExtraction?: AiExtraction
  /**
   * Opt in to the report export flow. Only the live ambient session passes this:
   * a reopened session has no chat thread to append, so the feature is offered
   * where the consultation actually just happened.
   */
  allowReport?: boolean
  /** Header metadata for the exported report. */
  reportMeta?: ReportMetaSource
}>()

const emit = defineEmits<{
  summarized: [extraction: AiExtraction]
}>()

const {
  isSummarizing,
  isAsking,
  extraction,
  chatMessages,
  error,
  summarize,
  ask,
  restore,
} = useAiAssistant()

const question = ref('')
const chatThreadRef = ref<HTMLElement | null>(null)
const showReportDialog = ref(false)
const showExportToast = ref(false)
const exportedFilename = ref('')

const onReportExported = (filename: string) => {
  exportedFilename.value = filename
  showExportToast.value = true
}

interface Section {
  label: string
  icon: string
  color: string
  items: string[]
  showEmpty?: boolean
}

const chipSection = (
  key: keyof AiExtraction,
  label: string,
  icon: string,
  color: string,
  showEmpty = false
): Section => ({
  label,
  icon,
  color,
  items: (extraction.value?.[key] as string[]) ?? [],
  showEmpty,
})

// Diagnoses and allergies always render ("None mentioned" is clinically
// meaningful); the rest appear only when the extraction found something.
const visibleChipSections = computed<Section[]>(() =>
  [
    chipSection('diagnosis', 'Diagnosis', 'mdi-clipboard-pulse', 'error', true),
    chipSection('differentialDiagnosis', 'Differential Diagnosis', 'mdi-stethoscope', 'error'),
    chipSection('allergies', 'Allergies', 'mdi-alert-octagon', 'warning', true),
    chipSection('medications', 'Medications', 'mdi-pill', 'secondary'),
    chipSection('symptoms', 'Symptoms', 'mdi-thermometer', 'secondary'),
    chipSection('labTests', 'Lab Tests', 'mdi-test-tube', 'info'),
    chipSection('procedures', 'Procedures', 'mdi-medical-bag', 'info'),
    chipSection('riskFactors', 'Risk Factors', 'mdi-alert', 'warning'),
    chipSection('medicalTerms', 'Medical Terms', 'mdi-book-open-variant', 'secondary'),
  ].filter(s => s.showEmpty || s.items.length > 0)
)

const visibleListSections = computed<Section[]>(() =>
  [
    chipSection('history', 'History', 'mdi-history', 'secondary'),
    chipSection('treatment', 'Treatment', 'mdi-bandage', 'secondary'),
    chipSection('followUp', 'Follow-up', 'mdi-calendar-clock', 'secondary'),
    chipSection('actionItems', 'Action Items', 'mdi-check-circle-outline', 'secondary'),
  ].filter(s => s.items.length > 0)
)

const runSummarize = async () => {
  try {
    await summarize(props.transcript, props.language || 'en')
    if (extraction.value) {
      emit('summarized', extraction.value)
    }
  } catch {
    // error state already handled by the composable
  }
}

const submitQuestion = async () => {
  const q = question.value.trim()
  if (!q || isAsking.value) return
  question.value = ''
  try {
    await ask(q)
  } catch {
    // restore the question so the user can retry
    question.value = q
  }
}

// Keep the chat scrolled to the newest message.
watch(
  () => chatMessages.value.length,
  async () => {
    await nextTick()
    chatThreadRef.value?.scrollTo({ top: chatThreadRef.value.scrollHeight, behavior: 'smooth' })
  }
)

onMounted(() => {
  if (props.initialExtraction) {
    restore(props.initialExtraction)
  } else {
    runSummarize()
  }
})
</script>

<style scoped>
.key-points-list {
  padding-left: 1.25rem;
}

.key-points-list li {
  margin-bottom: 4px;
}

.chat-thread {
  max-height: 320px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-right: 4px;
}

.chat-message {
  display: flex;
}

.chat-user {
  justify-content: flex-end;
}

.chat-assistant {
  justify-content: flex-start;
}

.chat-bubble {
  max-width: 85%;
  padding: 8px 12px;
  border-radius: 12px;
  white-space: pre-wrap;
  word-break: break-word;
}

.chat-user .chat-bubble {
  background: rgba(var(--v-theme-secondary), 0.15);
  border-bottom-right-radius: 4px;
}

.chat-assistant .chat-bubble {
  background: rgba(var(--v-theme-surface-variant), 0.6);
  border-bottom-left-radius: 4px;
}
</style>
