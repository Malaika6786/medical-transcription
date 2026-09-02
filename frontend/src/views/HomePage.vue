<template>
  <div class="home-page">
    <!-- Hero Section -->
    <v-row class="mb-12" justify="center">
      <v-col cols="12" md="10" lg="8" class="text-center">
        <h1 class="text-h2 font-weight-bold mb-4">
          <span class="gradient-text">AI-Powered</span> Medical Transcription
        </h1>
        <p class="text-h6 text-medium-emphasis mb-8" style="max-width: 600px; margin: 0 auto;">
          Transform clinical conversations into accurate, structured documentation 
          with xstek's advanced speech recognition and ambient AI technology.
        </p>
        
        <v-row justify="center" class="ga-4">
          <v-col cols="auto" v-if="canAccessFileTranscription">
            <v-btn
              size="x-large"
              color="primary"
              to="/async-transcription"
              prepend-icon="mdi-file-upload"
            >
              Upload Audio File
            </v-btn>
          </v-col>
          <v-col cols="auto" v-if="canAccessAmbient">
            <v-btn
              size="x-large"
              variant="outlined"
              color="primary"
              to="/ambient-session"
              prepend-icon="mdi-broadcast"
            >
              Ambient AI
            </v-btn>
          </v-col>
          <v-col cols="auto" v-if="canAccessDictation">
            <v-btn
              size="x-large"
              variant="outlined"
              color="secondary"
              to="/dictation"
              prepend-icon="mdi-microphone-message"
            >
              Dictation
            </v-btn>
          </v-col>
        </v-row>
      </v-col>
    </v-row>

    <!-- Features Grid -->
    <v-row class="mb-12">
      <v-col
        v-for="feature in features"
        :key="feature.title"
        cols="12"
        md="6"
        lg="3"
      >
        <v-card
          class="pa-6 h-100 glass-card"
          :style="{ borderTop: `3px solid ${feature.color}` }"
        >
          <v-icon
            :icon="feature.icon"
            :color="feature.color"
            size="48"
            class="mb-4"
          />
          <h3 class="text-h6 font-weight-bold mb-2">{{ feature.title }}</h3>
          <p class="text-body-2 text-medium-emphasis">
            {{ feature.description }}
          </p>
        </v-card>
      </v-col>
    </v-row>

    <!-- Quick Start Cards -->
    <v-row>
      <v-col cols="12" md="4" v-if="canAccessFileTranscription">
        <v-card class="pa-6 glass-card h-100">
          <div class="d-flex align-center mb-4">
            <v-avatar color="primary" size="56" class="mr-4">
              <v-icon icon="mdi-file-music" size="28" />
            </v-avatar>
            <div>
              <h3 class="text-h5 font-weight-bold">Async Transcription</h3>
              <p class="text-body-2 text-medium-emphasis">Upload pre-recorded audio</p>
            </div>
          </div>
          
          <v-list density="compact" class="bg-transparent">
            <v-list-item
              v-for="step in asyncSteps"
              :key="step"
              :prepend-icon="'mdi-check-circle'"
              :title="step"
              class="px-0"
            />
          </v-list>

          <v-btn
            block
            color="primary"
            variant="tonal"
            class="mt-4"
            to="/async-transcription"
          >
            Get Started
            <v-icon icon="mdi-arrow-right" class="ml-2" />
          </v-btn>
        </v-card>
      </v-col>

      <v-col cols="12" md="4" v-if="canAccessAmbient">
        <v-card class="pa-6 glass-card h-100">
          <div class="d-flex align-center mb-4">
            <v-avatar color="secondary" size="56" class="mr-4">
              <v-icon icon="mdi-broadcast" size="28" />
            </v-avatar>
            <div>
              <h3 class="text-h5 font-weight-bold">Ambient AI</h3>
              <p class="text-body-2 text-medium-emphasis">Real-time with facts extraction</p>
            </div>
          </div>
          
          <v-list density="compact" class="bg-transparent">
            <v-list-item
              v-for="step in ambientSteps"
              :key="step"
              :prepend-icon="'mdi-check-circle'"
              :title="step"
              class="px-0"
            />
          </v-list>

          <v-btn
            block
            color="secondary"
            variant="tonal"
            class="mt-4"
            to="/ambient-session"
          >
            Start Session
            <v-icon icon="mdi-arrow-right" class="ml-2" />
          </v-btn>
        </v-card>
      </v-col>

      <v-col cols="12" md="4" v-if="canAccessDictation">
        <v-card class="pa-6 glass-card h-100">
          <div class="d-flex align-center mb-4">
            <v-avatar color="info" size="56" class="mr-4">
              <v-icon icon="mdi-microphone-message" size="28" />
            </v-avatar>
            <div>
              <h3 class="text-h5 font-weight-bold">Dictation</h3>
              <p class="text-body-2 text-medium-emphasis">Stateless speech-to-text</p>
            </div>
          </div>
          
          <v-list density="compact" class="bg-transparent">
            <v-list-item
              v-for="step in dictationSteps"
              :key="step"
              :prepend-icon="'mdi-check-circle'"
              :title="step"
              class="px-0"
            />
          </v-list>

          <v-btn
            block
            color="info"
            variant="tonal"
            class="mt-4"
            to="/dictation"
          >
            Start Dictation
            <v-icon icon="mdi-arrow-right" class="ml-2" />
          </v-btn>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import { 
  canAccessAmbient, 
  canAccessFileTranscription, 
  canAccessDictation 
} from '@/stores/auth'

const features = [
  {
    icon: 'mdi-lightning-bolt',
    title: 'Real-Time Processing',
    description: 'Get instant transcriptions as you speak with our low-latency streaming technology.',
    color: '#00D9C4',
  },
  {
    icon: 'mdi-stethoscope',
    title: 'Medical Accuracy',
    description: 'Trained specifically for healthcare terminology with industry-leading accuracy.',
    color: '#7C5CFF',
  },
  {
    icon: 'mdi-account-voice',
    title: 'Speaker Diarization',
    description: 'Automatically distinguish between doctor and patient voices in conversations.',
    color: '#FFE66D',
  },
  {
    icon: 'mdi-file-document-edit',
    title: 'Smart Documentation',
    description: 'Generate structured clinical notes and summaries automatically.',
    color: '#FF6B6B',
  },
]

const asyncSteps = [
  'Upload WAV, MP3, or other audio formats',
  'Automatic language detection',
  'Receive structured transcript with timestamps',
  'Generate clinical documentation',
]

const ambientSteps = [
  'Capture audio directly from microphone',
  'See real-time partial transcripts',
  'Medical entity detection on-the-fly',
  'Generate clinical documents',
]

const dictationSteps = [
  'Simple stateless transcription',
  'Automatic punctuation & formatting',
  'Voice commands support',
  'Low-latency real-time output',
]
</script>

<style scoped>
.home-page {
  min-height: calc(100vh - 120px);
  padding-top: 2rem;
}
</style>

