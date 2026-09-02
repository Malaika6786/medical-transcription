<template>
  <div class="demo-guide-page">
    <!-- Header -->
    <v-row class="mb-6">
      <v-col cols="12">
        <div class="d-flex flex-column flex-md-row justify-space-between align-start align-md-center">
          <div>
            <h1 class="text-h3 text-md-h2 font-weight-bold mb-2">
              <v-icon icon="mdi-presentation-play" class="mr-3" color="primary" />
              Demo Guide
            </h1>
            <p class="text-body-1 text-medium-emphasis">
              A comprehensive walkthrough of XStek AI Medical Transcription features and capabilities
            </p>
          </div>
          <v-chip color="primary" variant="tonal" class="mt-3 mt-md-0">
            <v-icon icon="mdi-clock-outline" class="mr-1" />
            ~10 min read
          </v-chip>
        </div>
      </v-col>
    </v-row>

    <!-- Quick Navigation -->
    <v-row class="mb-6">
      <v-col cols="12">
        <v-card class="glass-card">
          <v-card-title>
            <v-icon icon="mdi-table-of-contents" class="mr-2" />
            Quick Navigation
          </v-card-title>
          <v-card-text>
            <v-chip-group>
              <v-chip 
                v-for="section in sections" 
                :key="section.id"
                variant="outlined"
                color="primary"
                @click="scrollToSection(section.id)"
              >
                <v-icon :icon="section.icon" class="mr-1" size="small" />
                {{ section.title }}
              </v-chip>
            </v-chip-group>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Overview Section -->
    <v-row class="mb-6" id="overview">
      <v-col cols="12">
        <v-card class="glass-card">
          <v-card-title class="text-h5">
            <v-icon icon="mdi-information-outline" class="mr-2" color="primary" />
            Application Overview
          </v-card-title>
          <v-card-text>
            <p class="text-body-1 mb-4">
              <strong>XStek AI Medical Transcription</strong> is an AI-powered platform designed to transform 
              clinical conversations into accurate, structured medical documentation. The application leverages 
              advanced speech recognition and ambient AI technology to streamline the documentation workflow 
              for healthcare professionals.
            </p>
            
            <v-alert type="info" variant="tonal" class="mb-4">
              <strong>Key Benefits:</strong>
              <ul class="mt-2 mb-0">
                <li>Reduce documentation time by up to 70%</li>
                <li>Improve accuracy with medical-specific AI models</li>
                <li>Generate structured clinical notes automatically</li>
                <li>Support for multiple documentation templates</li>
              </ul>
            </v-alert>

            <!-- User Roles Table - Super User Only -->
            <template v-if="isSuperUser">
              <h3 class="text-h6 mb-3">User Roles</h3>
              <v-table density="compact" class="rounded">
                <thead>
                  <tr>
                    <th>Role</th>
                    <th>Ambient AI</th>
                    <th>File Transcription</th>
                    <th>Dictation</th>
                    <th>Admin Features</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td><v-chip size="small" color="info">User</v-chip></td>
                    <td><v-icon icon="mdi-check" color="success" /></td>
                    <td><v-icon icon="mdi-close" color="error" /></td>
                    <td><v-icon icon="mdi-close" color="error" /></td>
                    <td><v-icon icon="mdi-close" color="error" /></td>
                  </tr>
                  <tr>
                    <td><v-chip size="small" color="secondary">Doctor</v-chip></td>
                    <td><v-icon icon="mdi-check" color="success" /></td>
                    <td><v-icon icon="mdi-check" color="success" /></td>
                    <td><v-icon icon="mdi-close" color="error" /></td>
                    <td><v-icon icon="mdi-close" color="error" /></td>
                  </tr>
                  <tr>
                    <td><v-chip size="small" color="primary">Super Admin</v-chip></td>
                    <td><v-icon icon="mdi-check" color="success" /></td>
                    <td><v-icon icon="mdi-check" color="success" /></td>
                    <td><v-icon icon="mdi-check" color="success" /></td>
                    <td><v-icon icon="mdi-check" color="success" /></td>
                  </tr>
                </tbody>
              </v-table>
            </template>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Ambient AI Section -->
    <v-row class="mb-6" id="ambient">
      <v-col cols="12">
        <v-card class="glass-card">
          <v-card-title class="text-h5">
            <v-icon icon="mdi-broadcast" class="mr-2" color="secondary" />
            Ambient AI Transcription
          </v-card-title>
          <v-card-text>
            <p class="text-body-1 mb-4">
              The Ambient AI module captures real-time clinical conversations and automatically generates 
              structured documentation. This is ideal for capturing doctor-patient interactions during 
              consultations.
            </p>

            <v-expansion-panels variant="accordion" class="mb-4">
              <v-expansion-panel>
                <v-expansion-panel-title>
                  <v-icon icon="mdi-numeric-1-circle" class="mr-2" color="primary" />
                  <strong>Step 1: Start a Session</strong>
                </v-expansion-panel-title>
                <v-expansion-panel-text>
                  <ol class="ml-4">
                    <li>Navigate to <strong>Ambient AI</strong> from the navigation menu</li>
                    <li>Select your preferred microphone from the dropdown</li>
                    <li>Choose the language for transcription (default: English US)</li>
                    <li>Click <strong>"Start New Recording"</strong> to begin capturing audio</li>
                  </ol>
                  <v-alert type="warning" variant="tonal" density="compact" class="mt-3">
                    Ensure your browser has microphone permissions enabled. If you have 5 saved sessions, you'll be prompted to manage them first.
                  </v-alert>
                </v-expansion-panel-text>
              </v-expansion-panel>

              <v-expansion-panel>
                <v-expansion-panel-title>
                  <v-icon icon="mdi-numeric-2-circle" class="mr-2" color="primary" />
                  <strong>Step 2: Real-Time Transcription</strong>
                </v-expansion-panel-title>
                <v-expansion-panel-text>
                  <ul class="ml-4">
                    <li>Watch the live transcript appear as you speak</li>
                    <li>The recording animation is displayed in the Stop button</li>
                    <li>Clinical facts are extracted in real-time (visible to Super Admin only)</li>
                    <li>Click <strong>"Stop Recording"</strong> when the conversation ends</li>
                  </ul>
                </v-expansion-panel-text>
              </v-expansion-panel>

              <v-expansion-panel>
                <v-expansion-panel-title>
                  <v-icon icon="mdi-numeric-3-circle" class="mr-2" color="primary" />
                  <strong>Step 3: Generate Documentation</strong>
                </v-expansion-panel-title>
                <v-expansion-panel-text>
                  <ul class="ml-4">
                    <li>Select a template from the dropdown (e.g., SOAP Note, GP Letter)</li>
                    <li>Click <strong>"Generate Document"</strong></li>
                    <li>The transcript auto-collapses and the generated report expands</li>
                    <li>Review and edit the document using the rich text editor</li>
                    <li>Click <strong>"Save Session"</strong> to store for future reference</li>
                  </ul>
                </v-expansion-panel-text>
              </v-expansion-panel>
            </v-expansion-panels>

            <v-btn color="secondary" variant="tonal" to="/ambient-session" v-if="canAccessAmbient">
              <v-icon icon="mdi-arrow-right" class="mr-2" />
              Try Ambient AI
            </v-btn>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- File Transcription Section (Only for users with access) -->
    <v-row v-if="canAccessFileTranscription" class="mb-6" id="file-transcription">
      <v-col cols="12">
        <v-card class="glass-card">
          <v-card-title class="text-h5">
            <v-icon icon="mdi-file-upload" class="mr-2" color="primary" />
            File Transcription
          </v-card-title>
          <v-card-text>
            <p class="text-body-1 mb-4">
              Upload pre-recorded audio files for transcription. Supports various audio formats 
              including WAV, MP3, M4A, and more.
            </p>

            <v-expansion-panels variant="accordion" class="mb-4">
              <v-expansion-panel>
                <v-expansion-panel-title>
                  <v-icon icon="mdi-numeric-1-circle" class="mr-2" color="primary" />
                  <strong>Step 1: Upload Audio File</strong>
                </v-expansion-panel-title>
                <v-expansion-panel-text>
                  <ol class="ml-4">
                    <li>Navigate to <strong>File Transcription</strong> from the menu</li>
                    <li>Drag and drop an audio file or click to browse</li>
                    <li>Supported formats: WAV, MP3, M4A, FLAC, OGG, WEBM</li>
                    <li>Select the audio language</li>
                    <li>Click <strong>"Start Transcription"</strong></li>
                  </ol>
                  <v-alert type="warning" variant="tonal" density="compact" class="mt-3">
                    If you have 5 saved sessions, you'll be prompted to manage them before starting.
                  </v-alert>
                </v-expansion-panel-text>
              </v-expansion-panel>

              <v-expansion-panel>
                <v-expansion-panel-title>
                  <v-icon icon="mdi-numeric-2-circle" class="mr-2" color="primary" />
                  <strong>Step 2: Review Transcript</strong>
                </v-expansion-panel-title>
                <v-expansion-panel-text>
                  <ul class="ml-4">
                    <li>Wait for processing to complete (progress shown)</li>
                    <li>Review the generated transcript with timestamps</li>
                    <li>Speaker diarization identifies different speakers</li>
                  </ul>
                </v-expansion-panel-text>
              </v-expansion-panel>

              <v-expansion-panel>
                <v-expansion-panel-title>
                  <v-icon icon="mdi-numeric-3-circle" class="mr-2" color="primary" />
                  <strong>Step 3: Generate Clinical Document</strong>
                </v-expansion-panel-title>
                <v-expansion-panel-text>
                  <ul class="ml-4">
                    <li>Select a documentation template</li>
                    <li>Configure template options if available</li>
                    <li>Click <strong>"Generate Document"</strong></li>
                    <li>Edit and format using the rich text editor</li>
                    <li>Save the session for future reference</li>
                  </ul>
                </v-expansion-panel-text>
              </v-expansion-panel>
            </v-expansion-panels>

            <v-btn color="primary" variant="tonal" to="/async-transcription">
              <v-icon icon="mdi-arrow-right" class="mr-2" />
              Try File Transcription
            </v-btn>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Templates Section -->
    <v-row class="mb-6" id="templates">
      <v-col cols="12">
        <v-card class="glass-card">
          <v-card-title class="text-h5">
            <v-icon icon="mdi-file-document-multiple" class="mr-2" color="warning" />
            Document Templates
          </v-card-title>
          <v-card-text>
            <p class="text-body-1 mb-4">
              The application includes several pre-configured clinical documentation templates 
              designed for common healthcare documentation needs.
            </p>

            <v-row>
              <v-col cols="12" md="6">
                <v-card variant="outlined" class="h-100">
                  <v-card-title class="text-subtitle-1">
                    <v-icon icon="mdi-note-text" class="mr-2" color="primary" />
                    SOAP Note
                  </v-card-title>
                  <v-card-text>
                    <p class="text-body-2 mb-3">
                      Standard medical documentation format with four sections:
                    </p>
                    <v-list density="compact" class="bg-transparent">
                      <v-list-item>
                        <template v-slot:prepend>
                          <v-icon icon="mdi-account-voice" size="small" color="primary" />
                        </template>
                        <v-list-item-title><strong>S</strong>ubjective</v-list-item-title>
                        <v-list-item-subtitle>Patient's reported symptoms and history</v-list-item-subtitle>
                      </v-list-item>
                      <v-list-item>
                        <template v-slot:prepend>
                          <v-icon icon="mdi-stethoscope" size="small" color="primary" />
                        </template>
                        <v-list-item-title><strong>O</strong>bjective</v-list-item-title>
                        <v-list-item-subtitle>Clinical observations and test results</v-list-item-subtitle>
                      </v-list-item>
                      <v-list-item>
                        <template v-slot:prepend>
                          <v-icon icon="mdi-brain" size="small" color="primary" />
                        </template>
                        <v-list-item-title><strong>A</strong>ssessment</v-list-item-title>
                        <v-list-item-subtitle>Diagnosis and clinical impressions</v-list-item-subtitle>
                      </v-list-item>
                      <v-list-item>
                        <template v-slot:prepend>
                          <v-icon icon="mdi-clipboard-list" size="small" color="primary" />
                        </template>
                        <v-list-item-title><strong>P</strong>lan</v-list-item-title>
                        <v-list-item-subtitle>Treatment plan and follow-up</v-list-item-subtitle>
                      </v-list-item>
                    </v-list>
                  </v-card-text>
                </v-card>
              </v-col>

              <v-col cols="12" md="6">
                <v-card variant="outlined" class="h-100">
                  <v-card-title class="text-subtitle-1">
                    <v-icon icon="mdi-email-outline" class="mr-2" color="secondary" />
                    GP Referral Letter
                  </v-card-title>
                  <v-card-text>
                    <p class="text-body-2 mb-3">
                      Professional referral letter format including:
                    </p>
                    <v-list density="compact" class="bg-transparent">
                      <v-list-item>
                        <template v-slot:prepend>
                          <v-icon icon="mdi-account" size="small" color="secondary" />
                        </template>
                        <v-list-item-title>Patient Information</v-list-item-title>
                        <v-list-item-subtitle>Demographics and contact details</v-list-item-subtitle>
                      </v-list-item>
                      <v-list-item>
                        <template v-slot:prepend>
                          <v-icon icon="mdi-history" size="small" color="secondary" />
                        </template>
                        <v-list-item-title>Clinical History</v-list-item-title>
                        <v-list-item-subtitle>Relevant medical background</v-list-item-subtitle>
                      </v-list-item>
                      <v-list-item>
                        <template v-slot:prepend>
                          <v-icon icon="mdi-target" size="small" color="secondary" />
                        </template>
                        <v-list-item-title>Reason for Referral</v-list-item-title>
                        <v-list-item-subtitle>Purpose and urgency</v-list-item-subtitle>
                      </v-list-item>
                      <v-list-item>
                        <template v-slot:prepend>
                          <v-icon icon="mdi-pill" size="small" color="secondary" />
                        </template>
                        <v-list-item-title>Current Medications</v-list-item-title>
                        <v-list-item-subtitle>Active prescriptions</v-list-item-subtitle>
                      </v-list-item>
                    </v-list>
                  </v-card-text>
                </v-card>
              </v-col>
            </v-row>

            <v-alert v-if="isSuperUser" type="info" variant="tonal" class="mt-4">
              <strong>Tip:</strong> You can customize verbosity levels for SOAP notes 
              (Concise, Standard, or Detailed) and override individual section prompts.
            </v-alert>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Session Management Section -->
    <v-row class="mb-6" id="sessions">
      <v-col cols="12">
        <v-card class="glass-card">
          <v-card-title class="text-h5">
            <v-icon icon="mdi-content-save-all" class="mr-2" color="success" />
            Session Management
          </v-card-title>
          <v-card-text>
            <p class="text-body-1 mb-4">
              Save and manage your transcription sessions for future reference. Each user can 
              store up to 5 sessions.
            </p>

            <v-row class="mb-4">
              <v-col cols="12" md="4">
                <v-card variant="tonal" color="primary" class="text-center pa-4">
                  <v-icon icon="mdi-content-save" size="48" class="mb-2" />
                  <h4 class="text-subtitle-1 font-weight-bold">Save Sessions</h4>
                  <p class="text-body-2 mb-0">Store transcripts and documents (max 5)</p>
                </v-card>
              </v-col>
              <v-col cols="12" md="4">
                <v-card variant="tonal" color="secondary" class="text-center pa-4">
                  <v-icon icon="mdi-pencil" size="48" class="mb-2" />
                  <h4 class="text-subtitle-1 font-weight-bold">Edit Documents</h4>
                  <p class="text-body-2 mb-0">Modify with rich text editor</p>
                </v-card>
              </v-col>
              <v-col cols="12" md="4">
                <v-card variant="tonal" color="success" class="text-center pa-4">
                  <v-icon icon="mdi-swap-horizontal" size="48" class="mb-2" />
                  <h4 class="text-subtitle-1 font-weight-bold">Change Template</h4>
                  <p class="text-body-2 mb-0">Regenerate with different template</p>
                </v-card>
              </v-col>
            </v-row>

            <v-btn color="success" variant="tonal" to="/saved-sessions">
              <v-icon icon="mdi-arrow-right" class="mr-2" />
              View Saved Sessions
            </v-btn>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Tips Section -->
    <v-row class="mb-6" id="tips">
      <v-col cols="12">
        <v-card class="glass-card">
          <v-card-title class="text-h5">
            <v-icon icon="mdi-lightbulb-on" class="mr-2" color="warning" />
            Pro Tips
          </v-card-title>
          <v-card-text>
            <v-row>
              <v-col cols="12" md="6">
                <v-list density="compact" class="bg-transparent">
                  <v-list-subheader>For Best Transcription Quality</v-list-subheader>
                  <v-list-item>
                    <template v-slot:prepend>
                      <v-icon icon="mdi-microphone" color="primary" />
                    </template>
                    <v-list-item-title>Use a quality microphone</v-list-item-title>
                    <v-list-item-subtitle>External mics provide better audio</v-list-item-subtitle>
                  </v-list-item>
                  <v-list-item>
                    <template v-slot:prepend>
                      <v-icon icon="mdi-volume-off" color="primary" />
                    </template>
                    <v-list-item-title>Minimize background noise</v-list-item-title>
                    <v-list-item-subtitle>Quiet environment improves accuracy</v-list-item-subtitle>
                  </v-list-item>
                  <v-list-item>
                    <template v-slot:prepend>
                      <v-icon icon="mdi-account-voice" color="primary" />
                    </template>
                    <v-list-item-title>Speak clearly</v-list-item-title>
                    <v-list-item-subtitle>Natural pace, avoid mumbling</v-list-item-subtitle>
                  </v-list-item>
                </v-list>
              </v-col>
              <v-col cols="12" md="6">
                <v-list density="compact" class="bg-transparent">
                  <v-list-subheader>For Efficient Documentation</v-list-subheader>
                  <v-list-item>
                    <template v-slot:prepend>
                      <v-icon icon="mdi-file-document-check" color="secondary" />
                    </template>
                    <v-list-item-title>Review before saving</v-list-item-title>
                    <v-list-item-subtitle>Always verify generated content</v-list-item-subtitle>
                  </v-list-item>
                  <v-list-item>
                    <template v-slot:prepend>
                      <v-icon icon="mdi-format-bold" color="secondary" />
                    </template>
                    <v-list-item-title>Use the editor toolbar</v-list-item-title>
                    <v-list-item-subtitle>Format documents professionally</v-list-item-subtitle>
                  </v-list-item>
                  <v-list-item>
                    <template v-slot:prepend>
                      <v-icon icon="mdi-content-save" color="secondary" />
                    </template>
                    <v-list-item-title>Manage saved sessions</v-list-item-title>
                    <v-list-item-subtitle>Max 5 per user, delete old ones to continue</v-list-item-subtitle>
                  </v-list-item>
                </v-list>
              </v-col>
            </v-row>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Interactive Tours Section -->
    <v-row class="mb-6" id="tours">
      <v-col cols="12">
        <v-card class="glass-card">
          <v-card-title class="text-h5">
            <v-icon icon="mdi-cursor-default-click" class="mr-2" color="info" />
            Interactive Guided Tours
          </v-card-title>
          <v-card-text>
            <p class="text-body-1 mb-4">
              Take an interactive walkthrough of the application. Each tour highlights key features 
              and shows you exactly where to click.
            </p>

            <v-row>
              <v-col cols="12" sm="6" md="3">
                <v-card 
                  variant="outlined" 
                  class="text-center pa-4 h-100 tour-card"
                  hover
                  @click="launchTour('welcome')"
                >
                  <v-icon icon="mdi-home" size="40" color="primary" class="mb-2" />
                  <h4 class="text-subtitle-1 font-weight-bold">Welcome Tour</h4>
                  <p class="text-body-2 text-medium-emphasis mb-3">Navigation & basics</p>
                  <v-btn color="primary" size="small" variant="tonal">
                    <v-icon icon="mdi-play" class="mr-1" />
                    Start
                  </v-btn>
                </v-card>
              </v-col>

              <v-col cols="12" sm="6" md="3">
                <v-card 
                  variant="outlined" 
                  class="text-center pa-4 h-100 tour-card"
                  hover
                  @click="launchTour('ambient')"
                >
                  <v-icon icon="mdi-broadcast" size="40" color="secondary" class="mb-2" />
                  <h4 class="text-subtitle-1 font-weight-bold">Ambient AI Tour</h4>
                  <p class="text-body-2 text-medium-emphasis mb-3">Real-time transcription</p>
                  <v-btn color="secondary" size="small" variant="tonal">
                    <v-icon icon="mdi-play" class="mr-1" />
                    Start
                  </v-btn>
                </v-card>
              </v-col>

              <v-col v-if="canAccessFileTranscription" cols="12" sm="6" md="3">
                <v-card 
                  variant="outlined" 
                  class="text-center pa-4 h-100 tour-card"
                  hover
                  @click="launchTour('file-transcription')"
                >
                  <v-icon icon="mdi-file-upload" size="40" color="primary" class="mb-2" />
                  <h4 class="text-subtitle-1 font-weight-bold">File Upload Tour</h4>
                  <p class="text-body-2 text-medium-emphasis mb-3">Upload audio files</p>
                  <v-btn color="primary" size="small" variant="tonal">
                    <v-icon icon="mdi-play" class="mr-1" />
                    Start
                  </v-btn>
                </v-card>
              </v-col>

              <v-col cols="12" sm="6" md="3">
                <v-card 
                  variant="outlined" 
                  class="text-center pa-4 h-100 tour-card"
                  hover
                  @click="launchTour('saved-sessions')"
                >
                  <v-icon icon="mdi-content-save-all" size="40" color="success" class="mb-2" />
                  <h4 class="text-subtitle-1 font-weight-bold">Sessions Tour</h4>
                  <p class="text-body-2 text-medium-emphasis mb-3">Manage saved work</p>
                  <v-btn color="success" size="small" variant="tonal">
                    <v-icon icon="mdi-play" class="mr-1" />
                    Start
                  </v-btn>
                </v-card>
              </v-col>
            </v-row>

            <v-divider class="my-4" />

            <div class="d-flex justify-center">
              <v-btn variant="text" color="warning" @click="resetAllToursHandler">
                <v-icon icon="mdi-refresh" class="mr-2" />
                Reset All Tours
              </v-btn>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Support Section -->
    <v-row id="support">
      <v-col cols="12">
        <v-card class="glass-card" color="primary" variant="tonal">
          <v-card-text class="text-center py-8">
            <v-icon icon="mdi-help-circle" size="64" class="mb-4" />
            <h3 class="text-h5 font-weight-bold mb-2">Need Help?</h3>
            <p class="text-body-1 mb-4">
              Contact xstek support for assistance with the application or to request additional features.
            </p>
            <v-btn color="primary" variant="elevated" href="mailto:support@xstek.com">
              <v-icon icon="mdi-email" class="mr-2" />
              Contact Support
            </v-btn>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { canAccessAmbient, canAccessFileTranscription, isSuperUser } from '@/stores/auth'
import { useProductTour, type TourType, resetAllTours } from '@/composables/useProductTour'

const router = useRouter()
const { startTour } = useProductTour()

import { computed } from 'vue'

// Define all sections with optional permission requirement
const allSections = [
  { id: 'overview', title: 'Overview', icon: 'mdi-information-outline', permission: null },
  { id: 'ambient', title: 'Ambient AI', icon: 'mdi-broadcast', permission: null },
  { id: 'file-transcription', title: 'File Transcription', icon: 'mdi-file-upload', permission: 'fileTranscription' },
  { id: 'templates', title: 'Templates', icon: 'mdi-file-document-multiple', permission: null },
  { id: 'sessions', title: 'Sessions', icon: 'mdi-content-save-all', permission: null },
  { id: 'tours', title: 'Interactive Tours', icon: 'mdi-cursor-default-click', permission: null },
  { id: 'tips', title: 'Pro Tips', icon: 'mdi-lightbulb-on', permission: null },
  { id: 'support', title: 'Support', icon: 'mdi-help-circle', permission: null },
]

// Filter sections based on user permissions
const sections = computed(() => 
  allSections.filter(section => {
    if (!section.permission) return true
    if (section.permission === 'fileTranscription') return canAccessFileTranscription.value
    return true
  })
)

const scrollToSection = (id: string) => {
  const element = document.getElementById(id)
  if (element) {
    element.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
}

// Launch a tour - navigate to the relevant page first if needed
const launchTour = (tourType: TourType) => {
  const routeMap: Record<TourType, string> = {
    'welcome': '/home',
    'ambient': '/ambient-session',
    'file-transcription': '/async-transcription',
    'saved-sessions': '/saved-sessions'
  }
  
  const targetRoute = routeMap[tourType]
  
  if (router.currentRoute.value.path !== targetRoute) {
    // Navigate first, then start tour
    router.push(targetRoute).then(() => {
      setTimeout(() => startTour(tourType, true), 800)
    })
  } else {
    startTour(tourType, true)
  }
}

// Reset all tours handler
const resetAllToursHandler = () => {
  resetAllTours()
  alert('All tours have been reset. They will appear again when you visit each page.')
}
</script>

<style scoped>
.demo-guide-page {
  min-height: calc(100vh - 120px);
}

.tour-card {
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.tour-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.15);
}

/* Mobile responsiveness */
@media (max-width: 600px) {
  .demo-guide-page h1 {
    font-size: 1.75rem !important;
  }
}
</style>

<!-- Global styles for Driver.js tour popover -->
<style>
.driver-popover.xstek-tour-popover {
  background: linear-gradient(135deg, rgba(30, 30, 46, 0.98) 0%, rgba(24, 24, 37, 0.98) 100%);
  border: 1px solid rgba(124, 92, 255, 0.3);
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.4), 0 0 40px rgba(124, 92, 255, 0.1);
}

.driver-popover.xstek-tour-popover .driver-popover-title {
  font-size: 18px;
  font-weight: 700;
  color: #00D9C4;
}

.driver-popover.xstek-tour-popover .driver-popover-description {
  color: rgba(255, 255, 255, 0.85);
  font-size: 14px;
  line-height: 1.6;
}

.driver-popover.xstek-tour-popover .driver-popover-progress-text {
  color: rgba(255, 255, 255, 0.5);
}

.driver-popover.xstek-tour-popover button {
  border-radius: 6px;
  font-weight: 600;
  padding: 8px 16px;
  transition: all 0.2s ease;
  text-rendering: optimizeLegibility;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

.driver-popover.xstek-tour-popover .driver-popover-next-btn,
.driver-popover.xstek-tour-popover .driver-popover-prev-btn,
.driver-popover.xstek-tour-popover .driver-popover-close-btn {
  text-shadow: none !important;
  filter: none !important;
  backdrop-filter: none !important;
  -webkit-backdrop-filter: none !important;
}

.driver-popover.xstek-tour-popover .driver-popover-next-btn {
  background: #7C5CFF;
  color: #ffffff !important;
  border: none;
  font-weight: 700;
}

.driver-popover.xstek-tour-popover .driver-popover-next-btn:hover {
  background: #6B4DE6;
  box-shadow: 0 4px 15px rgba(124, 92, 255, 0.4);
}

.driver-popover.xstek-tour-popover .driver-popover-prev-btn {
  background: rgba(255, 255, 255, 0.1);
  color: #ffffff !important;
  border: 1px solid rgba(255, 255, 255, 0.3);
  font-weight: 600;
}

.driver-popover.xstek-tour-popover .driver-popover-prev-btn:hover {
  background: rgba(255, 255, 255, 0.2);
  color: #ffffff !important;
}

.driver-popover.xstek-tour-popover .driver-popover-close-btn {
  color: rgba(255, 255, 255, 0.5);
}

.driver-popover.xstek-tour-popover .driver-popover-close-btn:hover {
  color: white;
}

.driver-popover.xstek-tour-popover .driver-popover-arrow-side-left.driver-popover-arrow,
.driver-popover.xstek-tour-popover .driver-popover-arrow-side-right.driver-popover-arrow,
.driver-popover.xstek-tour-popover .driver-popover-arrow-side-top.driver-popover-arrow,
.driver-popover.xstek-tour-popover .driver-popover-arrow-side-bottom.driver-popover-arrow {
  border-color: rgba(30, 30, 46, 0.98);
}
</style>
