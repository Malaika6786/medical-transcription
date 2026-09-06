<template>
  <div class="d-flex align-center ga-1">
    <v-menu v-if="!isDemoAccount">
      <template v-slot:activator="{ props: menuProps }">
        <v-btn
          v-bind="menuProps"
          size="small"
          variant="tonal"
          color="secondary"
          :loading="!!exportingFormat"
        >
          <v-icon icon="mdi-download" class="mr-1" size="18" />
          Export
          <v-icon icon="mdi-chevron-down" size="16" class="ml-1" />
        </v-btn>
      </template>
      <v-list density="compact">
        <v-list-item
          v-for="format in REPORT_FORMATS"
          :key="format.value"
          :disabled="!!exportingFormat"
          @click="handleExport(format.value)"
        >
          <template v-slot:prepend>
            <v-icon :icon="format.icon" size="18" />
          </template>
          <v-list-item-title>{{ format.label }}</v-list-item-title>
        </v-list-item>
      </v-list>
    </v-menu>

    <!-- Demo/trial accounts keep the button so they can see the feature exists,
         but clicking it upsells instead of exporting — trial accounts don't
         get to download or share generated documents. -->
    <v-btn v-else size="small" variant="tonal" color="secondary" @click="showUpgrade = true">
      <v-icon icon="mdi-download-lock" class="mr-1" size="18" />
      Export
    </v-btn>

    <v-snackbar v-model="showError" color="error" timeout="4000">
      {{ errorMessage }}
    </v-snackbar>
    <v-snackbar v-model="showSuccess" color="success" timeout="2500">
      Downloaded {{ lastFilename }}
    </v-snackbar>

    <v-dialog v-model="showUpgrade" max-width="380">
      <v-card>
        <v-card-text class="text-center pa-6">
          <v-icon icon="mdi-lock-outline" size="32" color="warning" class="mb-3" />
          <div class="text-h6 font-weight-bold mb-2">Exporting is a Doctor feature</div>
          <p class="text-body-2 text-medium-emphasis mb-4">
            Trial accounts can generate documents but can't download or share them yet.
            Upgrade to a Doctor account to export as PDF, Word, or Markdown.
          </p>
          <v-btn color="primary" block :href="upgradeContactHref">Request Doctor access</v-btn>
          <v-btn variant="text" block class="mt-2" @click="showUpgrade = false">Close</v-btn>
        </v-card-text>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { exportGeneratedDocument, REPORT_FORMATS, type ReportFormat } from '@/services/documentExport'
import { isDemoAccount } from '@/stores/auth'

const props = defineProps<{
  /** The rendered HTML of the document as currently edited. */
  html: string
  title: string
  templateName?: string
  patientName?: string
  clinician?: string
  consultationDate?: string
}>()

const exportingFormat = ref<ReportFormat | null>(null)
const showError = ref(false)
const errorMessage = ref('')
const showSuccess = ref(false)
const lastFilename = ref('')
const showUpgrade = ref(false)
const upgradeContactHref = `mailto:admin@xstek.net?subject=${encodeURIComponent('Requesting full Doctor access')}`

async function handleExport(format: ReportFormat) {
  exportingFormat.value = format
  try {
    lastFilename.value = await exportGeneratedDocument(props.html, format, {
      title: props.title,
      templateName: props.templateName,
      patientName: props.patientName,
      clinician: props.clinician,
      consultationDate: props.consultationDate,
    })
    showSuccess.value = true
  } catch (err: unknown) {
    errorMessage.value = err instanceof Error ? err.message : 'Export failed. Please try again.'
    showError.value = true
  } finally {
    exportingFormat.value = null
  }
}
</script>
