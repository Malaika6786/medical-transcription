<template>
  <div class="dictation-page">
    <v-row>
      <v-col cols="12">
        <h1 class="text-h4 font-weight-bold mb-2">
          <v-icon icon="mdi-microphone-message" class="mr-2" color="secondary" />
          Real-Time Dictation
        </h1>
        <p class="text-body-1 text-medium-emphasis mb-4">
          Voice-controlled clinical note-taking with automatic formatting and NHS templates.
        </p>

        <!-- Demo Guide Card -->
        <v-card class="glass-card mb-6" :class="{ 'demo-guide-expanded': showDemoGuide }">
          <v-card-title 
            class="d-flex justify-space-between align-center cursor-pointer"
            @click="showDemoGuide = !showDemoGuide"
          >
            <span>
              <v-icon icon="mdi-school" class="mr-2" color="warning" />
              <span class="text-h6">Quick Demo Guide</span>
              <v-chip size="x-small" color="success" class="ml-2">Step-by-Step</v-chip>
            </span>
            <v-icon :icon="showDemoGuide ? 'mdi-chevron-up' : 'mdi-chevron-down'" />
          </v-card-title>
          
          <v-expand-transition>
            <v-card-text v-show="showDemoGuide">
              <v-stepper 
                v-model="demoStep" 
                :items="demoSteps" 
                alt-labels
                flat
                hide-actions
                class="demo-stepper"
              >
                <template v-slot:item.1>
                  <v-card flat class="demo-step-card">
                    <v-card-text class="text-center pa-4">
                      <v-icon icon="mdi-play-circle" size="48" color="primary" class="mb-3" />
                      <h3 class="text-h6 mb-2">Start Recording</h3>
                      <p class="text-body-2 text-medium-emphasis mb-3">
                        Click the green "Start Dictation" button on the left panel.
                      </p>
                      <v-btn 
                        v-if="!isStreaming"
                        color="primary" 
                        @click="startDictation(); demoStep = 2"
                      >
                        <v-icon icon="mdi-microphone" class="mr-2" />
                        Start Now
                      </v-btn>
                      <v-chip v-else color="success" variant="flat">
                        <v-icon icon="mdi-check" class="mr-1" />
                        Recording Active
                      </v-chip>
                    </v-card-text>
                  </v-card>
                </template>

                <template v-slot:item.2>
                  <v-card flat class="demo-step-card">
                    <v-card-text class="pa-4">
                      <div class="text-center mb-4">
                        <v-icon icon="mdi-file-document-plus" size="48" color="warning" class="mb-3" />
                        <h3 class="text-h6 mb-2">Insert a Template</h3>
                        <p class="text-body-2 text-medium-emphasis">
                          Choose a template by saying:
                        </p>
                      </div>
                      <v-row class="mb-3">
                        <v-col cols="12" md="6">
                          <v-card variant="outlined" color="secondary" class="pa-3 text-center h-100">
                            <v-icon icon="mdi-email-edit" size="32" color="secondary" class="mb-2" />
                            <p class="text-body-2 font-weight-bold">"Insert GP Letter with Summary"</p>
                            <p class="text-caption text-medium-emphasis">13 sections - formal letter format</p>
                          </v-card>
                        </v-col>
                        <v-col cols="12" md="6">
                          <v-card variant="outlined" class="pa-3 text-center h-100">
                            <v-icon icon="mdi-file-document" size="32" color="primary" class="mb-2" />
                            <p class="text-body-2 font-weight-bold">"Insert GP consultation template"</p>
                            <p class="text-caption text-medium-emphasis">4 sections - SOAP format</p>
                          </v-card>
                        </v-col>
                      </v-row>
                      <div class="d-flex justify-center mt-3">
                        <v-btn variant="text" @click="demoStep = 1" class="mr-2">Back</v-btn>
                        <v-btn color="primary" @click="demoStep = 3">Next Step</v-btn>
                      </div>
                    </v-card-text>
                  </v-card>
                </template>

                <template v-slot:item.3>
                  <v-card flat class="demo-step-card">
                    <v-card-text class="pa-4">
                      <div class="text-center mb-4">
                        <v-icon icon="mdi-navigation" size="48" color="secondary" class="mb-3" />
                        <h3 class="text-h6 mb-2">Navigate & Dictate</h3>
                        <p class="text-body-2 text-medium-emphasis">
                          Move between sections and dictate content:
                        </p>
                      </div>
                      
                      <!-- GP Letter Sections -->
                      <div class="mb-3">
                        <p class="text-body-2 font-weight-bold mb-2">GP Letter Sections:</p>
                        <v-chip-group>
                          <v-chip size="small" color="secondary" variant="tonal">"Go to diagnoses"</v-chip>
                          <v-chip size="small" color="secondary" variant="tonal">"Go to medications"</v-chip>
                          <v-chip size="small" color="secondary" variant="tonal">"Go to management"</v-chip>
                          <v-chip size="small" color="secondary" variant="tonal">"Go to investigations"</v-chip>
                        </v-chip-group>
                      </div>
                      
                      <!-- Clinical Note Sections -->
                      <div class="mb-3">
                        <p class="text-body-2 font-weight-bold mb-2">Clinical Note Sections:</p>
                        <v-chip-group>
                          <v-chip size="small" color="primary" variant="tonal">"Go to presenting"</v-chip>
                          <v-chip size="small" color="primary" variant="tonal">"Go to examination"</v-chip>
                          <v-chip size="small" color="primary" variant="tonal">"Go to impression"</v-chip>
                          <v-chip size="small" color="primary" variant="tonal">"Go to plan"</v-chip>
                        </v-chip-group>
                      </div>
                      
                      <v-alert type="success" density="compact" variant="tonal" class="mb-3">
                        <strong>Tip:</strong> Also try <code>"Go to next"</code> or <code>"Go to previous"</code>
                      </v-alert>
                      <div class="d-flex justify-center">
                        <v-btn variant="text" @click="demoStep = 2" class="mr-2">Back</v-btn>
                        <v-btn color="primary" @click="demoStep = 4">Next Step</v-btn>
                      </div>
                    </v-card-text>
                  </v-card>
                </template>

                <template v-slot:item.4>
                  <v-card flat class="demo-step-card">
                    <v-card-text class="pa-4">
                      <div class="text-center mb-4">
                        <v-icon icon="mdi-check-circle" size="48" color="success" class="mb-3" />
                        <h3 class="text-h6 mb-2">Review & Export</h3>
                        <p class="text-body-2 text-medium-emphasis">
                          Your report is ready! You can:
                        </p>
                      </div>
                      <v-row class="mb-3">
                        <v-col cols="6">
                          <v-card variant="outlined" class="pa-3 text-center h-100">
                            <v-icon icon="mdi-pencil" size="32" color="primary" class="mb-2" />
                            <p class="text-body-2"><strong>Edit mode</strong> - Make changes directly in the report</p>
                          </v-card>
                        </v-col>
                        <v-col cols="6">
                          <v-card variant="outlined" class="pa-3 text-center h-100">
                            <v-icon icon="mdi-eye" size="32" color="secondary" class="mb-2" />
                            <p class="text-body-2"><strong>Preview mode</strong> - See the formatted final report</p>
                          </v-card>
                        </v-col>
                      </v-row>
                      <v-alert type="info" density="compact" variant="tonal" class="mb-3">
                        Click <strong>Copy</strong> to copy the report to your clipboard
                      </v-alert>
                      <div class="d-flex justify-center">
                        <v-btn variant="text" @click="demoStep = 3" class="mr-2">Back</v-btn>
                        <v-btn color="success" @click="showDemoGuide = false">
                          <v-icon icon="mdi-check" class="mr-1" />
                          Got It!
                        </v-btn>
                      </div>
                    </v-card-text>
                  </v-card>
                </template>
              </v-stepper>

              <!-- Quick Reference Section -->
              <v-divider class="my-4" />
              <h4 class="text-subtitle-1 font-weight-bold mb-3">
                <v-icon icon="mdi-lightning-bolt" class="mr-1" color="warning" />
                Quick Reference - What to Say
              </h4>
              <v-row dense>
                <v-col cols="12" sm="6" md="3">
                  <v-card variant="tonal" color="primary" class="pa-2">
                    <p class="text-caption font-weight-bold mb-1">🎯 Templates</p>
                    <code class="text-caption">"Insert GP consultation"</code>
                  </v-card>
                </v-col>
                <v-col cols="12" sm="6" md="3">
                  <v-card variant="tonal" color="secondary" class="pa-2">
                    <p class="text-caption font-weight-bold mb-1">📍 Navigate</p>
                    <code class="text-caption">"Go to [section] section"</code>
                  </v-card>
                </v-col>
                <v-col cols="12" sm="6" md="3">
                  <v-card variant="tonal" color="error" class="pa-2">
                    <p class="text-caption font-weight-bold mb-1">🗑️ Delete</p>
                    <code class="text-caption">"Delete last word"</code>
                  </v-card>
                </v-col>
                <v-col cols="12" sm="6" md="3">
                  <v-card variant="tonal" color="success" class="pa-2">
                    <p class="text-caption font-weight-bold mb-1">⏹️ Stop</p>
                    <code class="text-caption">"Stop dictation"</code>
                  </v-card>
                </v-col>
              </v-row>
            </v-card-text>
          </v-expand-transition>
        </v-card>
      </v-col>
    </v-row>

    <v-row>
      <!-- Control Panel -->
      <v-col cols="12" md="4">
        <v-card class="glass-card">
          <v-card-title class="text-h6">
            <v-icon icon="mdi-cog" class="mr-2" />
            Session Controls
          </v-card-title>

          <v-card-text>
            <!-- Connection Status -->
            <div class="mb-4">
              <span class="status-indicator" :class="connectionStatusClass">
                {{ connectionStatusText }}
              </span>
            </div>

            <!-- Language Selection -->
            <v-select
              v-model="selectedLanguage"
              :items="languages"
              item-title="name"
              item-value="code"
              label="Language"
              :disabled="isStreaming"
              prepend-inner-icon="mdi-translate"
              class="mb-4"
            />

            <!-- Microphone Selection -->
            <v-select
              v-model="selectedMicrophone"
              :items="microphones"
              item-title="label"
              item-value="deviceId"
              label="Microphone"
              :disabled="isStreaming"
              prepend-inner-icon="mdi-microphone"
              class="mb-4"
            />

            <!-- Recording Controls -->
            <div class="d-flex flex-column ga-3">
              <v-btn
                v-if="!isStreaming"
                block
                size="large"
                color="primary"
                :loading="isConnecting"
                @click="startDictation"
              >
                <v-icon icon="mdi-record" class="mr-2" />
                Start Dictation
              </v-btn>

              <v-btn
                v-else
                block
                size="large"
                color="error"
                @click="stopDictation"
              >
                <v-icon icon="mdi-stop" class="mr-2" />
                Stop Dictation
              </v-btn>

              <v-btn
                v-if="isStreaming"
                block
                variant="outlined"
                color="secondary"
                @click="flushBuffer"
              >
                <v-icon icon="mdi-refresh" class="mr-2" />
                Flush Buffer
              </v-btn>

              <v-btn
                block
                variant="outlined"
                :disabled="!transcript.length"
                @click="copyTranscript"
              >
                <v-icon icon="mdi-content-copy" class="mr-2" />
                Copy Transcript
              </v-btn>

              <v-btn
                block
                variant="text"
                :disabled="!transcript.length"
                @click="clearTranscript"
              >
                <v-icon icon="mdi-delete" class="mr-2" />
                Clear
              </v-btn>
            </div>
          </v-card-text>
        </v-card>

        <!-- Audio Visualizer -->
        <v-card v-if="isStreaming" class="glass-card mt-4">
          <v-card-title class="text-h6">
            <v-icon icon="mdi-waveform" class="mr-2" />
            Audio Input
          </v-card-title>
          <v-card-text class="text-center py-4">
            <div class="waveform-container">
              <span
                v-for="i in 5"
                :key="i"
                class="waveform-bar"
              />
            </div>
            <p class="text-body-2 text-medium-emphasis mt-3">
              Dictating...
            </p>
          </v-card-text>
        </v-card>

        <!-- Usage Info -->
        <v-card v-if="credits > 0" class="glass-card mt-4">
          <v-card-title class="text-h6">
            <v-icon icon="mdi-information" class="mr-2" />
            Usage
          </v-card-title>
          <v-card-text>
            <p class="text-body-2">
              Credits used: <strong>{{ credits.toFixed(2) }}</strong>
            </p>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Transcript & Commands -->
      <v-col cols="12" md="8">
        <v-row>
          <!-- Template Editor View (GP Letter OR Clinical Note) -->
          <v-col cols="12" v-if="templateMode">
            <v-card class="glass-card">
              <v-card-title class="d-flex justify-space-between align-center flex-wrap">
                <span>
                  <v-icon :icon="activeTemplateIcon" class="mr-2" :color="activeTemplateColor" />
                  {{ activeTemplateName }}
                  <v-chip v-if="isStreaming" size="x-small" color="error" variant="flat" class="ml-2">
                    <v-icon icon="mdi-microphone" size="x-small" class="mr-1 pulse-icon" />
                    Recording
                  </v-chip>
                </span>
                <div class="d-flex align-center ga-2 mt-2 mt-md-0">
                  <!-- Edit/Preview Toggle -->
                  <v-btn-toggle v-model="templateViewMode" mandatory density="compact" color="primary">
                    <v-btn value="edit" size="small">
                      <v-icon icon="mdi-pencil" size="small" class="mr-1" />
                      Edit
                    </v-btn>
                    <v-btn value="preview" size="small">
                      <v-icon icon="mdi-eye" size="small" class="mr-1" />
                      Preview
                    </v-btn>
                  </v-btn-toggle>
                  <v-btn
                    size="small"
                    variant="outlined"
                    :color="activeTemplateColor"
                    @click="copyTemplateDocument"
                  >
                    <v-icon icon="mdi-content-copy" class="mr-1" />
                    Copy
                  </v-btn>
                  <v-btn
                    size="small"
                    variant="text"
                    @click="exitTemplateMode"
                  >
                    <v-icon icon="mdi-close" />
                  </v-btn>
                </div>
              </v-card-title>

              <v-card-text>
                <!-- Edit Mode - Editable textarea -->
                <div v-if="templateViewMode === 'edit'" class="template-editor-container">
                  <textarea
                    ref="templateEditorRef"
                    v-model="templateFullText"
                    class="template-editor"
                    spellcheck="false"
                    @click="handleEditorClick"
                    @keyup="handleEditorClick"
                  ></textarea>
                  
                  <!-- Current Section Indicator -->
                  <div class="d-flex align-center justify-space-between mt-3">
                    <div class="text-caption text-medium-emphasis">
                      <v-icon icon="mdi-cursor-text" size="small" class="mr-1" />
                      Current section: <strong :class="`text-${activeTemplateColor}`">{{ currentSectionName }}</strong>
                    </div>
                    <div class="text-caption text-medium-emphasis">
                      Say: "Go to [section name]" to navigate
                    </div>
                  </div>
                </div>

                <!-- Preview Mode - Read-only formatted view -->
                <div v-else class="template-preview-container">
                  <div class="template-preview" v-html="formattedTemplatePreview"></div>
                </div>
              </v-card-text>
            </v-card>
          </v-col>

          <!-- Raw Transcript View -->
          <!-- Raw Transcript - shown when no template is active -->
          <v-col cols="12" v-else>
            <v-card class="glass-card">
              <v-card-title class="d-flex justify-space-between align-center">
                <span>
                  <v-icon icon="mdi-text" class="mr-2" />
                  Raw Dictation
                </span>
                <v-chip v-if="transcript.length" size="small" color="primary">
                  {{ transcript.filter(s => s.isFinal).length }} segments
                </v-chip>
              </v-card-title>

              <v-card-text>
                <!-- Prompt to insert template -->
                <v-alert 
                  type="info" 
                  variant="tonal" 
                  density="compact"
                  class="mb-4"
                >
                  <strong>Say:</strong> "Insert GP Letter" or "Insert Clinical Note" to start structured dictation
                </v-alert>

                <div v-if="transcript.length" class="transcript-container">
                  <template v-for="(segment, index) in transcript" :key="index">
                    <span
                      :class="[
                        'transcript-segment',
                        segment.isFinal ? 'final' : 'partial'
                      ]"
                    >
                      {{ segment.text }}
                    </span>
                    <span v-if="index < transcript.length - 1"> </span>
                  </template>
                </div>
                <div v-else class="text-center py-8">
                  <v-icon
                    icon="mdi-microphone-off"
                    size="48"
                    color="medium-emphasis"
                    class="mb-2"
                  />
                  <p class="text-body-2 text-medium-emphasis">
                    Start dictating to see your transcript here
                  </p>
                  <p class="text-caption text-medium-emphasis mt-2">
                    Or say <strong>"Insert GP Letter"</strong> to use a structured template
                  </p>
                </div>
              </v-card-text>
            </v-card>
          </v-col>

          <!-- Voice Commands -->
          <v-col cols="12" v-if="commands.length">
            <v-card class="glass-card">
              <v-card-title class="d-flex justify-space-between align-center">
                <span>
                  <v-icon icon="mdi-gesture-tap" class="mr-2" />
                  Detected Commands
                </span>
                <v-chip size="small" color="secondary">
                  {{ commands.length }} commands
                </v-chip>
              </v-card-title>

              <v-card-text>
                <v-list density="compact">
                  <v-list-item
                    v-for="(cmd, index) in commands"
                    :key="index"
                    :title="cmd.id"
                    :subtitle="cmd.variables ? JSON.stringify(cmd.variables) : ''"
                  >
                    <template v-slot:prepend>
                      <v-icon icon="mdi-console" color="primary" />
                    </template>
                  </v-list-item>
                </v-list>
              </v-card-text>
            </v-card>
          </v-col>

          <!-- Available Voice Commands -->
          <v-col cols="12">
            <v-card class="glass-card">
              <v-card-title class="text-h6">
                <v-icon icon="mdi-microphone-message" class="mr-2" />
                Available Voice Commands
              </v-card-title>
              <v-card-text>
                <v-expansion-panels variant="accordion">
                  <!-- Navigation Commands -->
                  <v-expansion-panel>
                    <v-expansion-panel-title>
                      <v-icon icon="mdi-navigation" class="mr-2" color="primary" />
                      Navigation
                    </v-expansion-panel-title>
                    <v-expansion-panel-text>
                      <v-alert type="info" density="compact" class="mb-3" variant="tonal">
                        <strong>Tip:</strong> Speak commands clearly and pause slightly before/after
                      </v-alert>
                      <v-list density="compact" class="command-list">
                        <v-list-item>
                          <template v-slot:prepend>
                            <v-icon icon="mdi-arrow-right" size="small" color="primary" />
                          </template>
                          <div>
                            <div class="command-phrase">"Go to next section"</div>
                            <div class="command-phrase">"Go to clinical findings"</div>
                            <div class="command-phrase">"Go to management plan"</div>
                            <div class="command-phrase">"Previous section"</div>
                          </div>
                          <div class="text-caption text-medium-emphasis mt-1">
                            NHS sections: presenting complaint, clinical findings, clinical impression, management plan
                          </div>
                          <div class="text-caption text-medium-emphasis">
                            Also supports: examination, diagnosis, treatment, next, previous
                          </div>
                        </v-list-item>
                      </v-list>
                    </v-expansion-panel-text>
                  </v-expansion-panel>

                  <!-- Editing Commands -->
                  <v-expansion-panel>
                    <v-expansion-panel-title>
                      <v-icon icon="mdi-pencil" class="mr-2" color="secondary" />
                      Text Editing
                    </v-expansion-panel-title>
                    <v-expansion-panel-text>
                      <v-list density="compact" class="command-list">
                        <v-list-item subtitle="Delete text">
                          <template v-slot:prepend>
                            <v-icon icon="mdi-delete" size="small" />
                          </template>
                          <strong>"Delete [range]"</strong>
                          <div class="text-caption text-medium-emphasis">
                            Range: everything, all, the last word, last sentence, that, this
                          </div>
                        </v-list-item>
                        <v-list-item subtitle="Select text">
                          <template v-slot:prepend>
                            <v-icon icon="mdi-selection" size="small" />
                          </template>
                          <strong>"Select [range]"</strong>
                          <div class="text-caption text-medium-emphasis">
                            Range: all, everything, the last word, the last sentence
                          </div>
                        </v-list-item>
                        <v-list-item subtitle="Undo/Redo">
                          <template v-slot:prepend>
                            <v-icon icon="mdi-undo" size="small" />
                          </template>
                          <strong>"Undo"</strong> / <strong>"Undo that"</strong> / <strong>"Redo"</strong>
                        </v-list-item>
                        <v-list-item subtitle="Clear all text">
                          <template v-slot:prepend>
                            <v-icon icon="mdi-close-circle" size="small" />
                          </template>
                          <strong>"Clear all"</strong> / <strong>"Clear everything"</strong> / <strong>"Start over"</strong>
                        </v-list-item>
                      </v-list>
                    </v-expansion-panel-text>
                  </v-expansion-panel>

                  <!-- Formatting Commands -->
                  <v-expansion-panel>
                    <v-expansion-panel-title>
                      <v-icon icon="mdi-format-paragraph" class="mr-2" color="success" />
                      Formatting
                    </v-expansion-panel-title>
                    <v-expansion-panel-text>
                      <v-list density="compact" class="command-list">
                        <v-list-item subtitle="New line">
                          <template v-slot:prepend>
                            <v-icon icon="mdi-keyboard-return" size="small" />
                          </template>
                          <strong>"New line"</strong> / <strong>"Next line"</strong> / <strong>"Line break"</strong>
                        </v-list-item>
                        <v-list-item subtitle="New paragraph">
                          <template v-slot:prepend>
                            <v-icon icon="mdi-format-pilcrow" size="small" />
                          </template>
                          <strong>"New paragraph"</strong> / <strong>"Next paragraph"</strong>
                        </v-list-item>
                      </v-list>
                    </v-expansion-panel-text>
                  </v-expansion-panel>

                  <!-- NHS Template Commands -->
                  <v-expansion-panel>
                    <v-expansion-panel-title>
                      <v-icon icon="mdi-file-document" class="mr-2" color="warning" />
                      NHS Templates
                    </v-expansion-panel-title>
                    <v-expansion-panel-text>
                      <v-list density="compact" class="command-list">
                        <v-list-item>
                          <template v-slot:prepend>
                            <v-icon icon="mdi-email-edit" size="small" color="secondary" />
                          </template>
                          <div>
                            <div class="command-phrase font-weight-bold">"Insert GP Letter with Summary"</div>
                            <div class="text-caption text-medium-emphasis mt-1">
                              Opens formal referral letter with 13 sections
                            </div>
                          </div>
                        </v-list-item>
                        <v-divider class="my-2" />
                        <v-list-item>
                          <template v-slot:prepend>
                            <v-icon icon="mdi-file-plus" size="small" color="warning" />
                          </template>
                          <div>
                            <div class="command-phrase">"Insert GP consultation template"</div>
                            <div class="command-phrase">"Insert clinical note template"</div>
                            <div class="command-phrase">"Insert discharge summary template"</div>
                            <div class="command-phrase">"Insert AE triage template"</div>
                          </div>
                        </v-list-item>
                        <v-list-item class="mt-2">
                          <template v-slot:prepend>
                            <v-icon icon="mdi-stethoscope" size="small" color="success" />
                          </template>
                          <div>
                            <div class="command-phrase">"Insert physical exam template"</div>
                            <div class="command-phrase">"Insert vital signs template"</div>
                            <div class="command-phrase">"Insert review of systems template"</div>
                          </div>
                        </v-list-item>
                        <v-list-item class="mt-2">
                          <template v-slot:prepend>
                            <v-icon icon="mdi-pill" size="small" color="primary" />
                          </template>
                          <div>
                            <div class="command-phrase">"Insert medication list template"</div>
                            <div class="command-phrase">"Insert allergy list template"</div>
                          </div>
                        </v-list-item>
                        <div class="text-caption text-medium-emphasis mt-2 ml-10">
                          All templates follow NHS PRSB standards with appropriate fields for dm+d medications, SNOMED CT coding, and NEWS2 scores.
                        </div>
                      </v-list>
                    </v-expansion-panel-text>
                  </v-expansion-panel>

                  <!-- Control Commands -->
                  <v-expansion-panel>
                    <v-expansion-panel-title>
                      <v-icon icon="mdi-power" class="mr-2" color="error" />
                      Control
                    </v-expansion-panel-title>
                    <v-expansion-panel-text>
                      <v-list density="compact" class="command-list">
                        <v-list-item subtitle="Stop dictation">
                          <template v-slot:prepend>
                            <v-icon icon="mdi-stop" size="small" />
                          </template>
                          <strong>"Stop dictation"</strong> / <strong>"Pause dictation"</strong> / <strong>"Stop listening"</strong>
                        </v-list-item>
                      </v-list>
                    </v-expansion-panel-text>
                  </v-expansion-panel>
                </v-expansion-panels>
              </v-card-text>
            </v-card>
          </v-col>

          <!-- Feature Info -->
          <v-col cols="12">
            <v-card class="glass-card">
              <v-card-title class="text-h6">
                <v-icon icon="mdi-lightbulb-on" class="mr-2" />
                Features
              </v-card-title>
              <v-card-text>
                <v-row>
                  <v-col cols="12" sm="6">
                    <div class="feature-item">
                      <v-icon icon="mdi-format-text" color="primary" class="mr-2" />
                      <span>Automatic punctuation & formatting</span>
                    </div>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <div class="feature-item">
                      <v-icon icon="mdi-numeric" color="primary" class="mr-2" />
                      <span>Smart number & date formatting</span>
                    </div>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <div class="feature-item">
                      <v-icon icon="mdi-gesture-tap" color="primary" class="mr-2" />
                      <span>Voice commands support</span>
                    </div>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <div class="feature-item">
                      <v-icon icon="mdi-flash" color="primary" class="mr-2" />
                      <span>Low-latency real-time streaming</span>
                    </div>
                  </v-col>
                </v-row>
              </v-card-text>
            </v-card>
          </v-col>
        </v-row>
      </v-col>
    </v-row>

    <!-- Command Action Feedback -->
    <v-snackbar 
      v-model="showCommandAction" 
      color="secondary" 
      timeout="3000"
      location="top"
    >
      <v-icon icon="mdi-gesture-tap" class="mr-2" />
      {{ lastCommandAction }}
    </v-snackbar>

    <!-- Snackbars -->
    <v-snackbar v-model="showError" color="error" timeout="5000">
      {{ errorMessage }}
      <template v-slot:actions>
        <v-btn variant="text" @click="showError = false">Close</v-btn>
      </template>
    </v-snackbar>

    <v-snackbar v-model="showSuccess" color="success" timeout="3000">
      {{ successMessage }}
    </v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useDictation } from '@/composables/useDictation'
import { useAudioCapture } from '@/composables/useAudioCapture'

const {
  isConnecting,
  isStreaming,
  connectionStatus,
  transcript,
  commands,
  error,
  credits,
  startSession,
  stopSession,
  sendAudioData,
  flush,
  reset,
  onCommand,
} = useDictation()

const {
  microphones,
  selectedMicrophone,
  initAudioCapture,
  startCapture,
  stopCapture,
} = useAudioCapture()

const selectedLanguage = ref('en')
const showError = ref(false)
const showSuccess = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const lastCommandAction = ref('')
const showCommandAction = ref(false)

// Demo Guide state
const showDemoGuide = ref(true) // Show by default for first-time users
const demoStep = ref(1)
const demoSteps = [
  { title: 'Start', value: 1 },
  { title: 'Template', value: 2 },
  { title: 'Navigate', value: 3 },
  { title: 'Review', value: 4 },
]

// ========================================
// UNIFIED TEMPLATE SYSTEM
// ========================================

// Template mode and view state
const templateMode = ref(false)
const templateViewMode = ref<'edit' | 'preview'>('edit')
const activeTemplate = ref<'gp-letter-with-summary' | 'clinical-note' | null>(null)
const templateEditorRef = ref<HTMLTextAreaElement | null>(null)
const templateFullText = ref('')
const currentSectionName = ref('Introduction')

// Template display properties
const activeTemplateName = computed(() => {
  switch (activeTemplate.value) {
    case 'gp-letter-with-summary': return 'GP Letter with Summary'
    case 'clinical-note': return 'NHS Clinical Note'
    default: return 'Document'
  }
})

const activeTemplateIcon = computed(() => {
  switch (activeTemplate.value) {
    case 'gp-letter-with-summary': return 'mdi-email-edit'
    case 'clinical-note': return 'mdi-file-document-edit'
    default: return 'mdi-file'
  }
})

const activeTemplateColor = computed(() => {
  switch (activeTemplate.value) {
    case 'gp-letter-with-summary': return 'secondary'
    case 'clinical-note': return 'primary'
    default: return 'primary'
  }
})

// Exit template mode
const exitTemplateMode = () => {
  templateMode.value = false
  activeTemplate.value = null
  templateViewMode.value = 'edit'
}

// Copy template document
const copyTemplateDocument = async () => {
  await navigator.clipboard.writeText(templateFullText.value)
  showSuccess.value = true
  successMessage.value = `${activeTemplateName.value} copied to clipboard!`
}

// Format template for preview (convert markdown-style to HTML)
const formattedTemplatePreview = computed(() => {
  let html = templateFullText.value
    // Escape HTML
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    // Bold text
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    // Line breaks
    .replace(/\n/g, '<br>')
  return `<div class="preview-content">${html}</div>`
})

// ========================================
// TEMPLATE DEFINITIONS
// ========================================

// GP Letter Template (formal letter format)
const gpLetterTemplateText = `[GP recipient details]
(Insert the full name, clinic name, and address of the referring GP)

Dear [clinician name or title],

Re: [patient name], DOB: [date of birth]

[Introductory statement acknowledging the referral]
Thank you for your referral regarding this patient.

[Brief overview of the patient's demographics and presenting complaint]


[Detailed description of the presenting complaint and relevant history]


[Clinical findings on examination]


[Summary of clinical reasoning and discussion with the patient]


**Diagnoses:**
1. 
2. 

**Investigations:**
1. 
2. 

**Medications:**
1. 
2. 

Note: 

**Suggested Management Plan:**
1. 
2. 
3. ACTIONS FOR SECRETARY: 

[Detailed description of planned management approach and follow-up plan]


Yours sincerely,

[clinician name]
[clinician title]`

// Clinical Note Template (SOAP/NHS format)
const clinicalNoteTemplateText = `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
NHS CLINICAL NOTE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Date: [DATE]
NHS Number: [NHS NUMBER]
Patient Name: [PATIENT NAME]
DOB: [DOB]

**PRESENTING COMPLAINT:**
[Describe the main reason for the consultation]


**HISTORY OF PRESENTING COMPLAINT:**
[Onset, duration, severity, associated symptoms]


**PAST MEDICAL HISTORY:**
[Relevant medical/surgical history]


**CURRENT MEDICATIONS:**
1. 
2. 
3. 

**ALLERGIES:**
• NKDA / [Allergy details]

**SOCIAL HISTORY:**
• Smoking Status: 
• Alcohol: 
• Occupation: 

**EXAMINATION FINDINGS:**
• General Appearance: 
• Observations: BP    /    mmHg, HR     bpm, Temp     °C, RR    /min, SpO2    %

**CLINICAL IMPRESSION:**
[Assessment and differential diagnoses]


**MANAGEMENT PLAN:**
1. 
2. 
3. 

**SAFETY NETTING:**
[Advice given to patient about when to seek further help]


**FOLLOW-UP:**
[Arrangements for review]


━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Clinician: [NAME] | GMC/NMC: [REG NUMBER]
Practice: [GP PRACTICE NAME]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━`

// Section markers for navigation
const templateSectionMarkers: Record<string, Record<string, { start: string; label: string }>> = {
  'gp-letter-with-summary': {
    'recipient': { start: '[GP recipient details]', label: 'Recipient Details' },
    'reference': { start: 'Re:', label: 'Patient Reference' },
    'introduction': { start: '[Introductory statement', label: 'Introduction' },
    'overview': { start: '[Brief overview', label: 'Patient Overview' },
    'history': { start: '[Detailed description of the presenting', label: 'Presenting History' },
    'findings': { start: '[Clinical findings', label: 'Clinical Findings' },
    'reasoning': { start: '[Summary of clinical reasoning', label: 'Clinical Reasoning' },
    'diagnoses': { start: '**Diagnoses:**', label: 'Diagnoses' },
    'investigations': { start: '**Investigations:**', label: 'Investigations' },
    'medications': { start: '**Medications:**', label: 'Medications' },
    'management': { start: '**Suggested Management Plan:**', label: 'Management Plan' },
    'follow_up': { start: '[Detailed description of planned management', label: 'Follow-up' },
    'sign_off': { start: 'Yours sincerely,', label: 'Sign-off' }
  },
  'clinical-note': {
    'presenting': { start: '**PRESENTING COMPLAINT:**', label: 'Presenting Complaint' },
    'history': { start: '**HISTORY OF PRESENTING COMPLAINT:**', label: 'History' },
    'past_medical': { start: '**PAST MEDICAL HISTORY:**', label: 'Past Medical History' },
    'medications': { start: '**CURRENT MEDICATIONS:**', label: 'Medications' },
    'allergies': { start: '**ALLERGIES:**', label: 'Allergies' },
    'social': { start: '**SOCIAL HISTORY:**', label: 'Social History' },
    'examination': { start: '**EXAMINATION FINDINGS:**', label: 'Examination' },
    'impression': { start: '**CLINICAL IMPRESSION:**', label: 'Clinical Impression' },
    'plan': { start: '**MANAGEMENT PLAN:**', label: 'Management Plan' },
    'safety': { start: '**SAFETY NETTING:**', label: 'Safety Netting' },
    'follow_up': { start: '**FOLLOW-UP:**', label: 'Follow-up' }
  }
}

// ========================================
// EDITOR INTERACTION HANDLERS
// ========================================

// Handle editor click to detect current section
const handleEditorClick = () => {
  if (!templateEditorRef.value || !activeTemplate.value) return
  
  const cursorPos = templateEditorRef.value.selectionStart
  const textBeforeCursor = templateFullText.value.substring(0, cursorPos)
  
  // Find which section we're in based on cursor position
  const markers = templateSectionMarkers[activeTemplate.value] || {}
  let detectedSection = 'Start'
  for (const [, marker] of Object.entries(markers)) {
    if (textBeforeCursor.includes(marker.start)) {
      detectedSection = marker.label
    }
  }
  currentSectionName.value = detectedSection
}

// Navigate to a section in the editor
const navigateToSection = (sectionKey: string) => {
  if (!templateEditorRef.value || !activeTemplate.value) return
  
  // Ensure we're in edit mode
  templateViewMode.value = 'edit'
  
  const markers = templateSectionMarkers[activeTemplate.value] || {}
  const marker = markers[sectionKey]
  if (!marker) return
  
  const text = templateFullText.value
  const position = text.indexOf(marker.start)
  
  if (position !== -1) {
    // Find the best cursor position
    let cursorPos = position
    
    // For numbered sections (like **Diagnoses:**\n1. ), position after "1. "
    const lineEnd = text.indexOf('\n', position)
    if (lineEnd !== -1) {
      // Check if next line starts with "1. "
      const nextLineStart = lineEnd + 1
      const nextLineContent = text.substring(nextLineStart, nextLineStart + 10)
      
      if (nextLineContent.startsWith('1. ')) {
        // Position cursor after "1. "
        cursorPos = nextLineStart + 3
      } else if (nextLineContent.match(/^\d+\.\s/)) {
        // Any numbered line - position after the number and space
        const match = nextLineContent.match(/^(\d+\.\s)/)
        cursorPos = nextLineStart + (match ? match[1].length : 0)
      } else {
        // For non-numbered sections, position at start of next line
        cursorPos = nextLineStart
      }
    }
    
    // Need to wait for Vue to update the DOM if we switched modes
    setTimeout(() => {
      if (templateEditorRef.value) {
        templateEditorRef.value.focus()
        templateEditorRef.value.setSelectionRange(cursorPos, cursorPos)
        
        // Scroll to the position
        const lineHeight = 24 // approximate line height
        const linesBeforeCursor = text.substring(0, cursorPos).split('\n').length
        templateEditorRef.value.scrollTop = (linesBeforeCursor - 5) * lineHeight
      }
    }, 50)
    
    currentSectionName.value = marker.label
    showCommandFeedback(`Now editing: ${marker.label}`)
  }
}

// Insert a new numbered point at cursor position
const insertNewPoint = () => {
  if (!templateEditorRef.value) return
  
  const editor = templateEditorRef.value
  const text = templateFullText.value
  const cursorPos = editor.selectionStart
  
  // Find the current line to determine the next number
  const textBeforeCursor = text.substring(0, cursorPos)
  const lines = textBeforeCursor.split('\n')
  
  // Look for the last numbered line to get the next number
  let nextNumber = 1
  for (let i = lines.length - 1; i >= 0; i--) {
    const match = lines[i].match(/^(\d+)\.\s/)
    if (match) {
      nextNumber = parseInt(match[1]) + 1
      break
    }
  }
  
  // Insert new line with next number
  const insertText = `\n${nextNumber}. `
  const beforeText = text.substring(0, cursorPos)
  const afterText = text.substring(cursorPos)
  
  templateFullText.value = beforeText + insertText + afterText
  
  // Move cursor after the new number
  const newCursorPos = cursorPos + insertText.length
  
  setTimeout(() => {
    if (templateEditorRef.value) {
      templateEditorRef.value.focus()
      templateEditorRef.value.setSelectionRange(newCursorPos, newCursorPos)
    }
  }, 0)
  
  showCommandFeedback(`Added point ${nextNumber}`)
}

// Insert a new line/paragraph
const insertNewLine = () => {
  if (!templateEditorRef.value) return
  
  const editor = templateEditorRef.value
  const text = templateFullText.value
  const cursorPos = editor.selectionStart
  
  const insertText = '\n'
  const beforeText = text.substring(0, cursorPos)
  const afterText = text.substring(cursorPos)
  
  templateFullText.value = beforeText + insertText + afterText
  
  const newCursorPos = cursorPos + insertText.length
  
  setTimeout(() => {
    if (templateEditorRef.value) {
      templateEditorRef.value.focus()
      templateEditorRef.value.setSelectionRange(newCursorPos, newCursorPos)
    }
  }, 0)
}

// Insert a bullet point
const insertBulletPoint = () => {
  if (!templateEditorRef.value) return
  
  const editor = templateEditorRef.value
  const text = templateFullText.value
  const cursorPos = editor.selectionStart
  
  const insertText = '\n• '
  const beforeText = text.substring(0, cursorPos)
  const afterText = text.substring(cursorPos)
  
  templateFullText.value = beforeText + insertText + afterText
  
  const newCursorPos = cursorPos + insertText.length
  
  setTimeout(() => {
    if (templateEditorRef.value) {
      templateEditorRef.value.focus()
      templateEditorRef.value.setSelectionRange(newCursorPos, newCursorPos)
    }
  }, 0)
  
  showCommandFeedback('Added bullet point')
}

// Note: navigateToGPLetterSection, getSectionIcon, getGPLetterSectionIcon removed - unused

// NHS-Compliant Template definitions
// Based on PRSB (Professional Record Standards Body) standards
const templates: Record<string, string> = {
  // GP Consultation Note - NHS Primary Care
  'gp consultation': `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
GP CONSULTATION NOTE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Date: [DATE]
NHS Number: [NHS NUMBER]
Patient Name: [PATIENT NAME]
DOB: [DOB]

PRESENTING COMPLAINT:


HISTORY OF PRESENTING COMPLAINT:


PAST MEDICAL HISTORY:


CURRENT MEDICATIONS (dm+d):


ALLERGIES AND ADVERSE REACTIONS:
• NKDA / [ALLERGY - REACTION TYPE]

SOCIAL HISTORY:
• Smoking Status: 
• Alcohol: 
• Occupation: 

EXAMINATION FINDINGS:
• General Appearance: 
• Observations: BP /  HR  Temp  RR  SpO2 %

CLINICAL IMPRESSION:


MANAGEMENT PLAN:


SAFETY NETTING:


FOLLOW-UP:


━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Clinician: [NAME] | GMC/NMC: [REG NUMBER]
Practice: [GP PRACTICE NAME]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,

  // Clinical Note - Generic NHS template
  'clinical note': `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
CLINICAL NOTE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Date/Time: [DATE/TIME]
NHS Number: [NHS NUMBER]

PRESENTING COMPLAINT:


CLINICAL FINDINGS:


CLINICAL IMPRESSION:


MANAGEMENT PLAN:


━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Author: [NAME] | Role: [ROLE]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,

  // SOAP Note (kept for compatibility, with NHS additions)
  'soap note': `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
CLINICAL CONSULTATION (SOAP)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

NHS Number: [NHS NUMBER]
Date: [DATE]

SUBJECTIVE (Patient History):


OBJECTIVE (Examination & Observations):
• BP: /  mmHg
• HR:  bpm
• Temp:  °C
• RR:  /min
• SpO2:  % on air

ASSESSMENT (Clinical Impression):


PLAN (Management):


━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Clinician: [NAME]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,
  'soap': `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
CLINICAL CONSULTATION (SOAP)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

NHS Number: [NHS NUMBER]
Date: [DATE]

SUBJECTIVE (Patient History):


OBJECTIVE (Examination & Observations):
• BP: /  mmHg
• HR:  bpm
• Temp:  °C
• RR:  /min
• SpO2:  % on air

ASSESSMENT (Clinical Impression):


PLAN (Management):


━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Clinician: [NAME]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,

  // NHS Physical Examination
  'physical exam': `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
PHYSICAL EXAMINATION
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

GENERAL APPEARANCE:
• Alert and oriented to time, place and person
• No acute distress

VITAL SIGNS:
• Blood Pressure: /  mmHg
• Heart Rate:  bpm, regular
• Temperature:  °C
• Respiratory Rate:  /min
• Oxygen Saturation:  % on room air
• Weight:  kg
• Height:  cm
• BMI: 

CARDIOVASCULAR:
• Heart sounds: S1 S2 present, no murmurs
• JVP: Not elevated
• Peripheral pulses: Present and equal
• No peripheral oedema

RESPIRATORY:
• Chest expansion: Equal bilaterally
• Percussion: Resonant throughout
• Auscultation: Vesicular breath sounds, no added sounds
• No wheeze or crackles

ABDOMINAL:
• Soft, non-tender, non-distended
• No organomegaly
• Bowel sounds: Present and normal

NEUROLOGICAL:
• Cranial nerves: Intact
• Power: 5/5 all limbs
• Sensation: Intact
• Reflexes: Normal

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Examiner: [NAME]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,

  // NHS Normal Examination
  'normal exam': `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EXAMINATION FINDINGS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

General: Patient appears well, no acute distress
CVS: Heart sounds normal, no murmurs
RS: Clear chest, good air entry bilaterally
Abdo: Soft, non-tender, no masses
CNS: Alert, oriented, no focal neurology

Impression: Examination unremarkable

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,

  // NHS Vital Signs (with metric units)
  'vital signs': `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
OBSERVATIONS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Date/Time: [DATE/TIME]

• Blood Pressure: /  mmHg
• Heart Rate:  bpm
• Temperature:  °C
• Respiratory Rate:  /min
• Oxygen Saturation:  % on [air/O2  L/min]
• NEWS2 Score: 

Consciousness: Alert / Voice / Pain / Unresponsive

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Recorded by: [NAME]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,

  // NHS Medication List (dm+d compatible format)
  'medication list': `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
CURRENT MEDICATIONS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

NHS Number: [NHS NUMBER]
Date Reviewed: [DATE]

REGULAR MEDICATIONS:
1. [DRUG NAME] [STRENGTH] [FORM] - [DOSE] [FREQUENCY] [ROUTE]
2. 
3. 

PRN (AS REQUIRED):
1. 

RECENTLY STOPPED:
1. [DRUG] - Stopped [DATE] - Reason: 

ALLERGIES/ADVERSE REACTIONS:
• [DRUG/SUBSTANCE] - [REACTION TYPE]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Reviewed by: [NAME] | Date: [DATE]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,

  // NHS Allergy Record
  'allergy list': `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
ALLERGIES AND ADVERSE REACTIONS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

NHS Number: [NHS NUMBER]

☐ No Known Drug Allergies (NKDA)

DRUG ALLERGIES:
1. Causative Agent: 
   Reaction Type: [Allergy/Intolerance/Adverse Reaction]
   Manifestation: 
   Severity: [Mild/Moderate/Severe/Life-threatening]
   Date First Experienced: 
   Certainty: [Confirmed/Suspected]

2. Causative Agent: 
   Reaction Type: 
   Manifestation: 

NON-DRUG ALLERGIES:
1. 

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Recorded by: [NAME] | Date: [DATE]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,

  // NHS Review of Systems
  'review of systems': `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
REVIEW OF SYSTEMS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

CONSTITUTIONAL:
☐ Fever ☐ Night sweats ☐ Weight loss ☐ Fatigue

CARDIOVASCULAR:
☐ Chest pain ☐ Palpitations ☐ Orthopnoea ☐ PND ☐ Oedema

RESPIRATORY:
☐ Dyspnoea ☐ Cough ☐ Sputum ☐ Haemoptysis ☐ Wheeze

GASTROINTESTINAL:
☐ Nausea ☐ Vomiting ☐ Abdominal pain ☐ Diarrhoea ☐ Constipation
☐ PR bleeding ☐ Dysphagia ☐ Jaundice

GENITOURINARY:
☐ Dysuria ☐ Frequency ☐ Haematuria ☐ Incontinence

NEUROLOGICAL:
☐ Headache ☐ Dizziness ☐ Visual disturbance ☐ Weakness
☐ Numbness ☐ Speech difficulty ☐ Seizures

MUSCULOSKELETAL:
☐ Joint pain ☐ Swelling ☐ Stiffness ☐ Back pain

SKIN:
☐ Rash ☐ Itching ☐ Lesions

PSYCHIATRIC:
☐ Low mood ☐ Anxiety ☐ Sleep disturbance

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,

  // NHS Discharge Summary
  'discharge summary': `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
DISCHARGE SUMMARY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

NHS Number: [NHS NUMBER]
Patient Name: [NAME]
DOB: [DOB]
GP Practice: [GP PRACTICE]

ADMISSION DETAILS:
• Date of Admission: [DATE]
• Date of Discharge: [DATE]
• Admitting Consultant: [NAME]
• Ward: [WARD]

DIAGNOSIS:
Primary: 
Secondary: 

PRESENTING COMPLAINT:


CLINICAL SUMMARY:


INVESTIGATIONS:


PROCEDURES:


DISCHARGE MEDICATIONS:
[Medication changes highlighted]

ALLERGIES:


FOLLOW-UP REQUIRED:


GP ACTIONS REQUIRED:


PATIENT ADVICE GIVEN:


━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Discharging Doctor: [NAME] | GMC: [NUMBER]
Consultant: [NAME]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,

  // NHS A&E Triage Note
  'ae triage': `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
A&E TRIAGE ASSESSMENT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Date/Time: [DATE/TIME]
NHS Number: [NHS NUMBER]
Mode of Arrival: [Walk-in/Ambulance/Other]

PRESENTING COMPLAINT:


OBSERVATIONS:
• BP: /  mmHg
• HR:  bpm
• Temp:  °C
• RR:  /min
• SpO2:  %
• GCS: /15
• Pain Score: /10
• Blood Glucose: 

MANCHESTER TRIAGE CATEGORY:
☐ Red - Immediate
☐ Orange - Very Urgent (10 min)
☐ Yellow - Urgent (60 min)
☐ Green - Standard (120 min)
☐ Blue - Non-Urgent (240 min)

DISCRIMINATOR:


ALLERGIES:


INITIAL ACTIONS:


━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Triage Nurse: [NAME] | NMC: [NUMBER]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,

  // Progress Note (NHS format)
  'progress note': `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
PROGRESS NOTE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Date/Time: [DATE/TIME]
NHS Number: [NHS NUMBER]

SUBJECTIVE:
Patient reports: 

OBJECTIVE:
Observations: BP /  HR  Temp  RR  SpO2 %
Examination: 

ASSESSMENT:


PLAN:


━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Author: [NAME] | Role: [ROLE]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,
}

const languages = [
  { name: 'English', code: 'en' },
  { name: 'Spanish', code: 'es' },
  { name: 'German', code: 'de' },
  { name: 'French', code: 'fr' },
  { name: 'Italian', code: 'it' },
  { name: 'Portuguese', code: 'pt' },
  { name: 'Dutch', code: 'nl' },
  { name: 'Polish', code: 'pl' },
  { name: 'Danish', code: 'da' },
  { name: 'Swedish', code: 'sv' },
  { name: 'Norwegian', code: 'no' },
  { name: 'Finnish', code: 'fi' },
]

/**
 * Handle voice commands - execute actions based on command type
 * This is where the magic happens - Corti detects the command, we execute it!
 */
const handleVoiceCommand = (command: { id: string; variables?: Record<string, string> }) => {
  console.log('Executing command:', command.id, command.variables)
  
  switch (command.id) {
    case 'go_to_section':
      executeNavigationCommand(command.variables?.section_key)
      break
    
    case 'insert_template':
      executeInsertTemplateCommand(command.variables?.template_name)
      break
    
    case 'delete_range':
      executeDeleteCommand(command.variables?.delete_range)
      break
    
    case 'new_line':
      if (templateMode.value) {
        insertNewLine()
      } else {
        insertText('\n')
      }
      showCommandFeedback('New line inserted')
      break
    
    case 'new_paragraph':
      if (templateMode.value) {
        insertTextAtCursor('\n\n')
      } else {
        insertText('\n\n')
      }
      showCommandFeedback('New paragraph inserted')
      break
    
    case 'new_point':
    case 'next_number':
      if (templateMode.value) {
        insertNewPoint()
      } else {
        showCommandFeedback('Insert a template first to use numbered points')
      }
      break
    
    case 'bullet_point':
    case 'new_bullet':
      if (templateMode.value) {
        insertBulletPoint()
      } else {
        showCommandFeedback('Insert a template first to use bullet points')
      }
      break
    
    case 'clear_all':
      reset()
      showCommandFeedback('All text cleared')
      break
    
    case 'undo':
      // Remove last transcript segment
      if (transcript.value.length > 0) {
        transcript.value.pop()
        showCommandFeedback('Undo: Removed last segment')
      }
      break
    
    case 'stop_dictation':
      stopDictation()
      showCommandFeedback('Dictation stopped by voice command')
      break
    
    default:
      showCommandFeedback(`Command detected: ${command.id}`)
  }
}

const executeNavigationCommand = (section: string | undefined) => {
  if (!section) return
  
  const normalizedSection = section.toLowerCase().trim()
  
  // Template mode navigation - unified for all templates
  if (templateMode.value && activeTemplate.value) {
    // Section name mappings for all templates
    const sectionNameMaps: Record<string, Record<string, string>> = {
      'gp-letter-with-summary': {
        'recipient details': 'recipient',
        'recipient': 'recipient',
        'gp details': 'recipient',
        'address': 'recipient',
        'patient reference': 'reference',
        'reference': 'reference',
        'introduction': 'introduction',
        'intro': 'introduction',
        'patient overview': 'overview',
        'overview': 'overview',
        'demographics': 'overview',
        'presenting history': 'history',
        'history': 'history',
        'clinical findings': 'findings',
        'examination': 'findings',
        'findings': 'findings',
        'clinical reasoning': 'reasoning',
        'reasoning': 'reasoning',
        'discussion': 'reasoning',
        'diagnoses': 'diagnoses',
        'diagnosis': 'diagnoses',
        'investigations': 'investigations',
        'tests': 'investigations',
        'medications': 'medications',
        'medication': 'medications',
        'meds': 'medications',
        'drugs': 'medications',
        'management plan': 'management',
        'management': 'management',
        'plan': 'management',
        'follow up': 'follow_up',
        'follow-up': 'follow_up',
        'sign off': 'sign_off',
        'sign-off': 'sign_off',
        'signature': 'sign_off',
        'closing': 'sign_off',
      },
      'clinical-note': {
        'presenting complaint': 'presenting',
        'presenting': 'presenting',
        'complaint': 'presenting',
        'history of presenting complaint': 'history',
        'history': 'history',
        'past medical history': 'past_medical',
        'past medical': 'past_medical',
        'past history': 'past_medical',
        'medications': 'medications',
        'medication': 'medications',
        'meds': 'medications',
        'current medications': 'medications',
        'allergies': 'allergies',
        'allergy': 'allergies',
        'social history': 'social',
        'social': 'social',
        'smoking': 'social',
        'examination findings': 'examination',
        'examination': 'examination',
        'exam': 'examination',
        'findings': 'examination',
        'observations': 'examination',
        'clinical impression': 'impression',
        'impression': 'impression',
        'assessment': 'impression',
        'diagnosis': 'impression',
        'diagnoses': 'impression',
        'management plan': 'plan',
        'management': 'plan',
        'plan': 'plan',
        'safety netting': 'safety',
        'safety': 'safety',
        'red flags': 'safety',
        'follow up': 'follow_up',
        'follow-up': 'follow_up',
        'review': 'follow_up',
      }
    }
    
    const sectionNameMap = sectionNameMaps[activeTemplate.value] || {}
    const markers = templateSectionMarkers[activeTemplate.value] || {}
    const sectionKeys = Object.keys(markers)
    
    // Handle next/previous
    if (normalizedSection === 'next' || normalizedSection === 'next section') {
      const currentIdx = sectionKeys.findIndex(key => markers[key].label === currentSectionName.value)
      if (currentIdx < sectionKeys.length - 1) {
        navigateToSection(sectionKeys[currentIdx + 1])
      } else {
        showCommandFeedback('Already at last section')
      }
    } else if (normalizedSection === 'previous' || normalizedSection === 'previous section') {
      const currentIdx = sectionKeys.findIndex(key => markers[key].label === currentSectionName.value)
      if (currentIdx > 0) {
        navigateToSection(sectionKeys[currentIdx - 1])
      } else {
        showCommandFeedback('Already at first section')
      }
    } else {
      // Find matching section
      const mappedKey = sectionNameMap[normalizedSection]
      if (mappedKey) {
        navigateToSection(mappedKey)
      } else {
        // Try partial match
        for (const [name, key] of Object.entries(sectionNameMap)) {
          if (normalizedSection.includes(name) || name.includes(normalizedSection)) {
            navigateToSection(key)
            return
          }
        }
        showCommandFeedback(`Section not found: ${normalizedSection}`)
      }
    }
    return
  }
  
  // Not in template mode - ignore navigation commands
  showCommandFeedback('Insert a template first to use navigation')
}

const executeInsertTemplateCommand = (templateName: string | undefined) => {
  if (!templateName) return
  
  const normalizedName = templateName.toLowerCase()
  
  // GP Letter with Summary template (formal letter format)
  if (normalizedName === 'gp letter with summary' || normalizedName === 'gp letter' || normalizedName === 'gp referral letter') {
    templateMode.value = true
    activeTemplate.value = 'gp-letter-with-summary'
    templateViewMode.value = 'edit'
    templateFullText.value = gpLetterTemplateText
    currentSectionName.value = 'Introduction'
    // Clear raw transcript to avoid confusion
    transcript.value = []
    
    showCommandFeedback('GP Letter template loaded!')
    successMessage.value = 'Template loaded! Say "Go to [section]" to navigate. Example: "Go to diagnoses"'
    showSuccess.value = true
    
    // Focus the editor and navigate to introduction after a short delay
    setTimeout(() => {
      navigateToSection('introduction')
    }, 300)
    return
  }
  
  // Clinical Note / SOAP templates
  const clinicalNoteTemplates = [
    'soap note', 'soap', 'clinical note', 'gp consultation', 
    'discharge summary', 'ae triage', 'progress note'
  ]
  
  if (clinicalNoteTemplates.includes(normalizedName)) {
    templateMode.value = true
    activeTemplate.value = 'clinical-note'
    templateViewMode.value = 'edit'
    templateFullText.value = clinicalNoteTemplateText
    currentSectionName.value = 'Presenting Complaint'
    // Clear raw transcript to avoid confusion
    transcript.value = []
    
    showCommandFeedback('NHS Clinical Note template loaded!')
    successMessage.value = 'Template loaded! Say "Go to [section]" to navigate. Example: "Go to medications"'
    showSuccess.value = true
    
    // Focus the editor and navigate to presenting complaint after a short delay
    setTimeout(() => {
      navigateToSection('presenting')
    }, 300)
    return
  }
  
  // For other templates (physical exam, vital signs, etc.), insert the actual text
  const templateText = templates[normalizedName]
  
  if (templateText) {
    // In template mode, insert at cursor
    if (templateMode.value) {
      insertTextAtCursor(templateText)
    } else {
      insertText(templateText)
    }
    showCommandFeedback(`Inserted: ${templateName} template`)
  } else {
    showCommandFeedback(`Template not found: ${templateName}`)
  }
}

const executeDeleteCommand = (range: string | undefined) => {
  if (!range) return
  
  const normalizedRange = range.toLowerCase()
  
  switch (normalizedRange) {
    case 'everything':
    case 'all':
      transcript.value = []
      showCommandFeedback('Deleted all text')
      break
    
    case 'the last word':
    case 'last word':
      if (transcript.value.length > 0) {
        const lastSegment = transcript.value[transcript.value.length - 1]
        const words = lastSegment.text.trim().split(/\s+/)
        if (words.length > 1) {
          words.pop()
          lastSegment.text = words.join(' ') + ' '
        } else {
          transcript.value.pop()
        }
        showCommandFeedback('Deleted last word')
      }
      break
    
    case 'the last sentence':
    case 'last sentence':
    case 'that':
    case 'this':
      if (transcript.value.length > 0) {
        transcript.value.pop()
        showCommandFeedback('Deleted last sentence')
      }
      break
    
    default:
      showCommandFeedback(`Delete: ${range}`)
  }
}

const insertText = (text: string) => {
  transcript.value.push({
    text: text,
    start: 0,
    end: 0,
    isFinal: true,
    timestamp: Date.now(),
  })
}

const showCommandFeedback = (message: string) => {
  lastCommandAction.value = message
  showCommandAction.value = true
  
  // Auto-hide after 3 seconds
  setTimeout(() => {
    showCommandAction.value = false
  }, 3000)
}

/**
 * Copy the formatted NHS Clinical Note to clipboard
 */
// Note: copySoapNote and copyGPLetter removed - unused

/**
 * Track the last transcript length to detect new additions
 */
let lastTranscriptLength = 0

/**
 * Insert text at cursor position in the template editor
 */
const insertTextAtCursor = (text: string) => {
  if (!templateEditorRef.value) return
  
  const editor = templateEditorRef.value
  const start = editor.selectionStart
  const end = editor.selectionEnd
  const currentText = templateFullText.value
  
  // Add space before text if needed
  const beforeText = currentText.substring(0, start)
  const afterText = currentText.substring(end)
  const needsSpace = beforeText.length > 0 && !beforeText.endsWith(' ') && !beforeText.endsWith('\n')
  
  const insertedText = (needsSpace ? ' ' : '') + text.trim()
  templateFullText.value = beforeText + insertedText + afterText
  
  // Move cursor to end of inserted text
  const newCursorPos = start + insertedText.length
  
  // Need to wait for Vue to update the DOM
  setTimeout(() => {
    if (templateEditorRef.value) {
      templateEditorRef.value.focus()
      templateEditorRef.value.setSelectionRange(newCursorPos, newCursorPos)
    }
  }, 0)
}

/**
 * Watch transcript for new segments and insert into template editor
 */
watch(transcript, (newTranscript) => {
  // Check if new segments were added
  if (newTranscript.length > lastTranscriptLength) {
    // Get new segments
    const newSegments = newTranscript.slice(lastTranscriptLength)
    
    // Append final segments to template
    for (const segment of newSegments) {
      if (segment.isFinal && segment.text.trim()) {
        if (templateMode.value) {
          // Insert at cursor position in template editor
          insertTextAtCursor(segment.text.trim())
        }
      }
    }
  }
  
  lastTranscriptLength = newTranscript.length
}, { deep: true })

// Register command handler when component mounts
let unsubscribeCommand: (() => void) | null = null

const connectionStatusClass = computed(() => {
  switch (connectionStatus.value) {
    case 'ready':
    case 'connected':
      return 'connected'
    case 'connecting':
      return 'recording'
    case 'disconnected':
    default:
      return 'disconnected'
  }
})

const connectionStatusText = computed(() => {
  switch (connectionStatus.value) {
    case 'ready':
      return 'Ready to dictate'
    case 'connected':
      return 'Configuring...'
    case 'connecting':
      return 'Connecting...'
    case 'disconnected':
    default:
      return 'Disconnected'
  }
})

const startDictation = async () => {
  try {
    // Start dictation session
    await startSession(selectedLanguage.value)
    
    // Initialize and start audio capture
    await initAudioCapture(selectedMicrophone.value)
    await startCapture((audioData: ArrayBuffer) => {
      sendAudioData(audioData)
    })
  } catch (err) {
    showError.value = true
    errorMessage.value = error.value || 'Failed to start dictation'
  }
}

const stopDictation = async () => {
  stopCapture()
  stopSession()
  showSuccess.value = true
  successMessage.value = 'Dictation stopped'
}

const flushBuffer = () => {
  flush()
  showSuccess.value = true
  successMessage.value = 'Buffer flushed'
}

const copyTranscript = async () => {
  const text = transcript.value
    .filter(s => s.isFinal)
    .map(s => s.text)
    .join(' ')
  
  await navigator.clipboard.writeText(text)
  showSuccess.value = true
  successMessage.value = 'Transcript copied!'
}

const clearTranscript = () => {
  reset()
  showSuccess.value = true
  successMessage.value = 'Transcript cleared'
}

onMounted(async () => {
  // Register command handler
  unsubscribeCommand = onCommand(handleVoiceCommand)
  
  // Request microphone permissions and list devices
  try {
    await navigator.mediaDevices.getUserMedia({ audio: true })
    const devices = await navigator.mediaDevices.enumerateDevices()
    const audioInputs = devices
      .filter(d => d.kind === 'audioinput')
      .map(d => ({
        deviceId: d.deviceId,
        label: d.label || `Microphone ${d.deviceId.slice(0, 5)}`,
      }))
    
    if (audioInputs.length) {
      microphones.value = audioInputs
      selectedMicrophone.value = audioInputs[0].deviceId
    }
  } catch (err) {
    console.error('Failed to get microphones:', err)
  }
})

onUnmounted(() => {
  // Unsubscribe from command events
  if (unsubscribeCommand) {
    unsubscribeCommand()
  }
  
  if (isStreaming.value) {
    stopDictation()
  }
})
</script>

<style scoped>
.dictation-page {
  min-height: calc(100vh - 120px);
}

.transcript-container {
  line-height: 1.8;
  font-size: 1.1rem;
}

.transcript-segment {
  transition: all 0.2s ease;
}

.transcript-segment.final {
  color: inherit;
}

.transcript-segment.partial {
  color: #888;
  font-style: italic;
}

.feature-item {
  display: flex;
  align-items: center;
  padding: 8px 0;
}

.waveform-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 48px;
}

.waveform-bar {
  width: 4px;
  height: 20px;
  margin: 0 3px;
  background: linear-gradient(180deg, #7C5CFF, #00D9FF);
  border-radius: 2px;
  animation: waveform 0.5s ease-in-out infinite alternate;
}

.waveform-bar:nth-child(1) { animation-delay: 0s; }
.waveform-bar:nth-child(2) { animation-delay: 0.1s; }
.waveform-bar:nth-child(3) { animation-delay: 0.2s; }
.waveform-bar:nth-child(4) { animation-delay: 0.3s; }
.waveform-bar:nth-child(5) { animation-delay: 0.4s; }

@keyframes waveform {
  from { height: 10px; }
  to { height: 40px; }
}

.status-indicator {
  display: inline-flex;
  align-items: center;
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 0.875rem;
  font-weight: 500;
}

.status-indicator::before {
  content: '';
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 8px;
}

.status-indicator.connected {
  background: rgba(76, 175, 80, 0.1);
  color: #4CAF50;
}

.status-indicator.connected::before {
  background: #4CAF50;
}

.status-indicator.recording {
  background: rgba(255, 152, 0, 0.1);
  color: #FF9800;
}

.status-indicator.recording::before {
  background: #FF9800;
  animation: pulse 1s infinite;
}

.status-indicator.disconnected {
  background: rgba(158, 158, 158, 0.1);
  color: #9E9E9E;
}

.status-indicator.disconnected::before {
  background: #9E9E9E;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.command-list {
  background: transparent !important;
}

.command-list .v-list-item {
  padding: 8px 0;
}

.command-phrase {
  font-family: 'Consolas', 'Monaco', monospace;
  background: rgba(124, 92, 255, 0.1);
  padding: 4px 8px;
  border-radius: 4px;
  margin: 2px 0;
  display: inline-block;
  font-size: 0.9rem;
  color: #7C5CFF;
}

.section-nav {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.soap-sections {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.soap-section {
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: all 0.3s ease;
}

.soap-section.active {
  border-color: #7C5CFF;
  box-shadow: 0 0 12px rgba(124, 92, 255, 0.3);
}

.soap-section-header {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: rgba(124, 92, 255, 0.1);
  cursor: pointer;
  transition: background 0.2s ease;
}

.soap-section-header:hover {
  background: rgba(124, 92, 255, 0.2);
}

.soap-section.active .soap-section-header {
  background: rgba(124, 92, 255, 0.25);
}

.soap-textarea {
  margin: 0;
}

.soap-textarea.active-section :deep(.v-field) {
  border-color: #7C5CFF;
}

.pulse-icon {
  animation: pulse-recording 1s infinite;
}

@keyframes pulse-recording {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

/* Demo Guide Styles */
.demo-guide-expanded {
  border: 2px solid rgb(var(--v-theme-warning));
}

.demo-stepper {
  background: transparent !important;
}

.demo-step-card {
  background: var(--hover-overlay, rgba(255, 255, 255, 0.05));
  border-radius: 12px;
}

.say-this-alert {
  text-align: center;
  font-size: 1.1rem;
}

.demo-command-table {
  background: transparent !important;
}

.demo-command-table code {
  background: rgba(var(--v-theme-secondary), 0.15);
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 0.85rem;
  white-space: nowrap;
}

.demo-command-table .active-row {
  background: rgba(var(--v-theme-primary), 0.15);
}

.demo-command-table .active-row td {
  font-weight: 600;
}

.cursor-pointer {
  cursor: pointer;
}

.feature-item {
  display: flex;
  align-items: center;
  padding: 8px 0;
}

/* Template Editor Styles */
.template-editor-container {
  position: relative;
  border: 1px solid rgba(var(--v-theme-primary), 0.3);
  border-radius: 8px;
  overflow: hidden;
}

.template-editor {
  width: 100%;
  min-height: 60vh;
  max-height: 65vh;
  padding: 24px;
  border: none;
  outline: none;
  resize: none;
  font-family: 'Georgia', 'Times New Roman', serif;
  font-size: 15px;
  line-height: 1.8;
  background: rgb(var(--v-theme-surface));
  color: rgb(var(--v-theme-on-surface));
  overflow-y: auto;
}

.template-editor:focus {
  box-shadow: inset 0 0 0 2px rgba(var(--v-theme-primary), 0.3);
}

.template-editor::placeholder {
  color: rgba(var(--v-theme-on-surface), 0.4);
}

/* Template Preview Styles */
.template-preview-container {
  border: 1px solid rgba(var(--v-theme-primary), 0.2);
  border-radius: 8px;
  background: rgb(var(--v-theme-surface));
  min-height: 60vh;
  max-height: 65vh;
  overflow-y: auto;
}

.template-preview {
  padding: 24px;
  font-family: 'Georgia', 'Times New Roman', serif;
  font-size: 15px;
  line-height: 1.8;
  color: rgb(var(--v-theme-on-surface));
}

.template-preview .preview-content {
  white-space: pre-wrap;
  word-wrap: break-word;
}

/* Scrollbar for template editor/preview */
.template-editor::-webkit-scrollbar,
.template-preview-container::-webkit-scrollbar {
  width: 8px;
}

.template-editor::-webkit-scrollbar-track,
.template-preview-container::-webkit-scrollbar-track {
  background: rgba(0, 0, 0, 0.1);
}

.template-editor::-webkit-scrollbar-thumb,
.template-preview-container::-webkit-scrollbar-thumb {
  background: rgba(var(--v-theme-primary), 0.4);
  border-radius: 4px;
}

.template-editor::-webkit-scrollbar-thumb:hover,
.template-preview-container::-webkit-scrollbar-thumb:hover {
  background: rgba(var(--v-theme-primary), 0.6);
}
</style>

