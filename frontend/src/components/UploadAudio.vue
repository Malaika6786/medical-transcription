<template>
  <div
    class="upload-zone"
    :class="{
      'dragging': isDragging,
      'has-file': selectedFile
    }"
    @dragover.prevent="onDragOver"
    @dragleave.prevent="onDragLeave"
    @drop.prevent="onDrop"
    @click="triggerFileInput"
  >
    <input
      ref="fileInput"
      type="file"
      accept="audio/*,.wav,.mp3,.m4a,.flac,.ogg,.webm,.aac,.aiff,.aif,.mp4"
      class="d-none"
      @change="onFileChange"
    />

    <template v-if="selectedFile">
      <v-icon
        icon="mdi-file-music"
        size="64"
        color="success"
        class="mb-4"
      />
      <h3 class="text-h6 font-weight-bold mb-2">{{ selectedFile.name }}</h3>
      <p class="text-body-2 text-medium-emphasis mb-4">
        {{ formatFileSize(selectedFile.size) }}
      </p>
      <v-btn
        variant="text"
        color="error"
        size="small"
        @click.stop="clearFile"
      >
        <v-icon icon="mdi-close" class="mr-1" />
        Remove
      </v-btn>
    </template>

    <template v-else>
      <v-icon
        icon="mdi-cloud-upload-outline"
        size="64"
        color="primary"
        class="mb-4"
      />
      <h3 class="text-h6 font-weight-medium mb-2">
        Drop audio file here
      </h3>
      <p class="text-body-2 text-medium-emphasis">
        or click to browse
      </p>
      <p class="text-caption text-medium-emphasis mt-2">
        Supported: WAV, MP3, M4A, FLAC, OGG, WebM (max 100MB)
      </p>
    </template>

    <v-progress-linear
      v-if="loading"
      indeterminate
      color="primary"
      class="mt-4"
      style="max-width: 200px; margin: 0 auto;"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

defineProps<{
  loading?: boolean
}>()

const emit = defineEmits<{
  (e: 'file-selected', file: File): void
  (e: 'upload-complete'): void
  (e: 'error', message: string): void
}>()

const fileInput = ref<HTMLInputElement | null>(null)
const selectedFile = ref<File | null>(null)
const isDragging = ref(false)

const SUPPORTED_TYPES = [
  'audio/wav',
  'audio/wave',
  'audio/x-wav',
  'audio/mpeg',
  'audio/mp3',
  'audio/mp4',
  'audio/m4a',
  'audio/x-m4a',
  'audio/ogg',
  'audio/flac',
  'audio/x-flac',
  'audio/webm',
  'audio/aac',
  'audio/aiff',
  'audio/x-aiff',
]

const MAX_FILE_SIZE = 100 * 1024 * 1024 // 100MB

const triggerFileInput = () => {
  fileInput.value?.click()
}

const validateFile = (file: File): boolean => {
  // Check file size
  if (file.size > MAX_FILE_SIZE) {
    emit('error', 'File size exceeds 100MB limit')
    return false
  }

  // Check file type
  const isValidType = SUPPORTED_TYPES.includes(file.type) ||
    file.name.match(/\.(wav|mp3|m4a|flac|ogg|webm|aac|aiff|aif|mp4)$/i)
  
  if (!isValidType) {
    emit('error', 'Unsupported file format')
    return false
  }

  return true
}

const handleFile = (file: File) => {
  if (validateFile(file)) {
    selectedFile.value = file
    emit('file-selected', file)
  }
}

const onFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (file) {
    handleFile(file)
  }
}

const onDragOver = (_event: DragEvent) => {
  isDragging.value = true
}

const onDragLeave = (_event: DragEvent) => {
  isDragging.value = false
}

const onDrop = (event: DragEvent) => {
  isDragging.value = false
  const file = event.dataTransfer?.files?.[0]
  if (file) {
    handleFile(file)
  }
}

const clearFile = () => {
  selectedFile.value = null
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

const formatFileSize = (bytes: number): string => {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

// Expose file for parent component
defineExpose({
  selectedFile,
  clearFile,
})
</script>

