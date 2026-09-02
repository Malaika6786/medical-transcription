<template>
  <div :class="{ 'glass-card': !hideTitle }">
    <v-card-title v-if="!hideTitle" class="d-flex justify-space-between align-center flex-wrap">
      <span>
        <v-icon icon="mdi-file-document-check" class="mr-2" color="success" />
        Generated Report
      </span>
      <div class="d-flex ga-2 mt-2 mt-sm-0">
        <!-- Export menu (optional) -->
        <slot name="export-actions"></slot>
      </div>
    </v-card-title>

    <div :class="hideTitle ? '' : 'pa-4'">
      <RichTextEditor
        v-model="localContent"
        :editable="true"
        :hide-toolbar="true"
      />
      
      <v-alert
        v-if="isProcessing"
        type="info"
        variant="tonal"
        class="mt-3"
      >
        Report is being generated. Please wait...
      </v-alert>
    </div>

    <!-- Save Button at bottom of document -->
    <div v-if="showSaveButton" :class="hideTitle ? 'pt-4' : 'px-4 pb-4'">
      <div class="d-flex justify-end">
        <v-btn
          color="primary"
          variant="elevated"
          :loading="isSaving"
          @click="$emit('save')"
        >
          <v-icon icon="mdi-content-save" class="mr-2" />
          Save Session
        </v-btn>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import RichTextEditor from '@/components/RichTextEditor.vue'

const props = withDefaults(defineProps<{
  modelValue: string
  isProcessing?: boolean
  showSaveButton?: boolean
  isSaving?: boolean
  hideTitle?: boolean  // Hide title when used inside expansion panel
}>(), {
  isProcessing: false,
  showSaveButton: true,
  isSaving: false,
  hideTitle: false
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'save'): void
}>()

// Two-way binding for content
const localContent = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

// Convert HTML to plain text for exporting
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

// Expose for parent component usage
defineExpose({
  htmlToPlainText,
  getPlainText: () => htmlToPlainText(localContent.value)
})
</script>

<style scoped>
/* Component specific styles */
</style>
