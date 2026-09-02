<template>
  <v-container class="search-page" max-width="900">
    <!-- Header -->
    <div class="mb-6">
      <h1 class="text-h5 font-weight-bold d-flex align-center">
        <v-icon icon="mdi-magnify" class="mr-2" color="primary" />
        Semantic Search
      </h1>
      <p class="text-body-2 text-medium-emphasis mt-1">
        Search your saved sessions by meaning — results are ranked by similarity
        between your query and the stored transcript &amp; AI extraction embeddings.
      </p>
    </div>

    <!-- Query input -->
    <v-text-field
      v-model="query"
      placeholder="e.g. patient with childhood asthma and night cough"
      variant="outlined"
      density="comfortable"
      prepend-inner-icon="mdi-magnify"
      :loading="searching"
      :disabled="searching"
      clearable
      autofocus
      hide-details
      @keyup.enter="runSearch"
    >
      <template #append-inner>
        <v-btn
          color="primary"
          variant="flat"
          size="small"
          :loading="searching"
          :disabled="!query || !query.trim()"
          @click="runSearch"
        >
          Search
        </v-btn>
      </template>
    </v-text-field>

    <!-- Error state -->
    <v-alert
      v-if="errorMessage"
      type="warning"
      variant="tonal"
      class="mt-4"
      closable
      @click:close="errorMessage = ''"
    >
      {{ errorMessage }}
    </v-alert>

    <!-- Results -->
    <div v-if="searched && !errorMessage" class="mt-6">
      <div class="text-body-2 text-medium-emphasis mb-3">
        {{ results.length === 0
          ? 'No sessions matched your query.'
          : `${results.length} matching session${results.length === 1 ? '' : 's'}` }}
      </div>

      <v-card
        v-for="result in results"
        :key="result.sessionId"
        class="mb-3 result-card"
        variant="outlined"
        hover
        @click="openResult(result)"
      >
        <div class="pa-4">
          <div class="d-flex align-center mb-2">
            <v-icon :icon="typeIcon(result.type)" size="18" class="mr-2" color="primary" />
            <span class="text-subtitle-1 font-weight-medium text-truncate flex-grow-1 mr-2">
              {{ result.title }}
            </span>
            <v-chip size="x-small" variant="tonal" color="primary" class="mr-2">
              {{ Math.round(result.score * 100) }}% match
            </v-chip>
            <v-chip size="x-small" variant="outlined">
              {{ result.source === 'extraction' ? 'AI Extraction' : 'Transcript' }}
            </v-chip>
          </div>
          <p class="text-body-2 text-medium-emphasis snippet mb-2">{{ result.snippet }}</p>
          <div class="text-caption text-disabled">
            {{ typeLabel(result.type) }} · {{ formatDate(result.updatedAt) }}
          </div>
        </div>
      </v-card>
    </div>

    <!-- Initial hint -->
    <div v-else-if="!searched" class="text-center mt-12 text-medium-emphasis">
      <v-icon icon="mdi-text-search" size="48" class="mb-3" />
      <p class="text-body-2">
        Try describing symptoms, diagnoses, or treatments in your own words —<br />
        semantic search matches meaning, not just exact keywords.
      </p>
    </div>
  </v-container>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/services/api'

interface SearchResult {
  sessionId: string
  title: string
  type: string
  updatedAt: string
  source: 'transcript' | 'extraction'
  snippet: string
  score: number
}

const router = useRouter()

const query = ref('')
const searching = ref(false)
const searched = ref(false)
const results = ref<SearchResult[]>([])
const errorMessage = ref('')

async function runSearch() {
  const q = query.value?.trim()
  if (!q || searching.value) return
  searching.value = true
  errorMessage.value = ''
  try {
    const { data } = await api.post('/search', { query: q, limit: 10 })
    results.value = data.results ?? []
    searched.value = true
  } catch (err: any) {
    results.value = []
    searched.value = true
    errorMessage.value =
      err?.response?.data?.error ||
      'Search failed — please try again.'
  } finally {
    searching.value = false
  }
}

function openResult(result: SearchResult) {
  router.push({ path: '/saved-sessions', query: { open: result.sessionId } })
}

function typeIcon(type: string): string {
  switch (type) {
    case 'ambient': return 'mdi-broadcast'
    case 'dictation': return 'mdi-microphone-message'
    case 'file-transcription': return 'mdi-file-upload'
    default: return 'mdi-content-save'
  }
}

function typeLabel(type: string): string {
  switch (type) {
    case 'ambient': return 'Ambient Session'
    case 'dictation': return 'Dictation'
    case 'file-transcription': return 'File Transcription'
    default: return type
  }
}

function formatDate(iso: string): string {
  const d = new Date(iso)
  return isNaN(d.getTime()) ? iso : d.toLocaleString()
}
</script>

<style scoped>
.search-page {
  padding-top: 24px;
}
.result-card {
  cursor: pointer;
  transition: border-color 0.2s;
}
.snippet {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
  white-space: pre-line;
}
</style>
