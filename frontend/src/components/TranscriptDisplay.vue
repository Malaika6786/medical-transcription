<template>
  <div class="transcript-display">
    <div
      ref="transcriptContainer"
      class="transcript-container"
      :class="{ 'is-streaming': isStreaming }"
    >
      <template v-if="hasContent">
        <!-- Flat text display - continuous paragraph style -->
        <div class="flat-transcript-content">
          <p class="text-body-1 mb-0" style="line-height: 1.8; white-space: pre-wrap;">
            <!-- For segments array (ambient/live mode) -->
            <template v-if="segments && segments.length">
              <template v-for="(segment, index) in segments" :key="segment.id || index">
                <span :class="{ 'text-medium-emphasis font-italic': !segment.isFinal }">{{ segment.text }}</span>
                <span v-if="!segment.isFinal" class="typing-indicator">
                  <span></span><span></span><span></span>
                </span>
                <span v-if="index < segments.length - 1"> </span>
              </template>
            </template>
            <!-- For plain text (file transcription mode) -->
            <template v-else-if="text">
              {{ text }}
            </template>
          </p>
        </div>

        <!-- Streaming indicator -->
        <div v-if="isStreaming" class="d-flex align-center mt-3 text-medium-emphasis">
          <v-progress-circular indeterminate size="16" width="2" class="mr-2" />
          <span class="text-caption">Listening...</span>
        </div>
      </template>

      <template v-else>
        <div class="text-center py-12">
          <v-icon
            v-if="isStreaming"
            icon="mdi-microphone-message"
            size="64"
            color="primary"
            class="mb-4 animate-pulse"
          />
          <v-icon
            v-else
            icon="mdi-text-box-outline"
            size="64"
            color="medium-emphasis"
            class="mb-4"
          />
          <p class="text-h6 text-medium-emphasis">
            {{ isStreaming ? 'Listening...' : emptyTitle }}
          </p>
          <p class="text-body-2 text-medium-emphasis">
            {{ isStreaming ? 'Speak clearly into your microphone' : emptySubtitle }}
          </p>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, computed } from 'vue'

interface TranscriptSegment {
  id?: string
  text: string
  speaker?: string
  isFinal?: boolean
  timestamp?: number
}

const props = withDefaults(defineProps<{
  segments?: TranscriptSegment[]
  text?: string
  isStreaming?: boolean
  emptyTitle?: string
  emptySubtitle?: string
}>(), {
  segments: () => [],
  text: '',
  isStreaming: false,
  emptyTitle: 'No transcript yet',
  emptySubtitle: 'Start a session to begin transcribing'
})

const transcriptContainer = ref<HTMLElement | null>(null)

const hasContent = computed(() => {
  return (props.segments && props.segments.length > 0) || (props.text && props.text.trim().length > 0)
})

// Auto-scroll to bottom when new segments are added
watch(() => props.segments?.length, async () => {
  await nextTick()
  if (transcriptContainer.value) {
    transcriptContainer.value.scrollTop = transcriptContainer.value.scrollHeight
  }
})
</script>

<style scoped>
.transcript-display {
  min-height: 200px;
}

.transcript-container {
  max-height: 400px;
  overflow-y: auto;
  padding: 1rem;
}

.transcript-container.is-streaming {
  border: 1px solid rgba(0, 217, 196, 0.3);
  border-radius: 12px;
  background: rgba(0, 217, 196, 0.02);
}

/* Flat transcript content - plain text paragraphs */
.flat-transcript-content {
  font-size: 1rem;
  line-height: 1.8;
}

/* Typing indicator */
.typing-indicator {
  display: inline-flex;
  margin-left: 4px;
  gap: 2px;
  vertical-align: middle;
}

.typing-indicator span {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.5;
  animation: typing 1.4s infinite both;
}

.typing-indicator span:nth-child(2) {
  animation-delay: 0.2s;
}

.typing-indicator span:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes typing {
  0%, 60%, 100% {
    transform: translateY(0);
    opacity: 0.5;
  }
  30% {
    transform: translateY(-4px);
    opacity: 1;
  }
}

/* Pulse animation for listening indicator */
.animate-pulse {
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}
</style>
