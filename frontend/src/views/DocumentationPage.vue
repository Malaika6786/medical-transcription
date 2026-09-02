<template>
  <div class="documentation-page">
    <v-row>
      <v-col cols="12">
        <h1 class="text-h4 font-weight-bold mb-2">
          <v-icon icon="mdi-book-open-page-variant" class="mr-2" color="primary" />
          Documentation
        </h1>
        <p class="text-body-1 text-medium-emphasis mb-6">
          Learn about XStek AI Medical Transcription features, terminology, and best practices.
        </p>
      </v-col>
    </v-row>

    <v-row>
      <!-- Side Navigation -->
      <v-col cols="12" md="3">
        <v-card class="glass-card sticky-nav">
          <v-card-title class="text-subtitle-1 font-weight-bold">
            <v-icon icon="mdi-format-list-bulleted" class="mr-2" size="20" />
            Contents
          </v-card-title>
          <v-list density="compact" nav class="bg-transparent">
            <v-list-item
              v-for="section in sections"
              :key="section.id"
              :prepend-icon="section.icon"
              :title="section.title"
              @click="scrollToSection(section.id)"
              :class="{ 'v-list-item--active': activeSection === section.id }"
            />
          </v-list>

          <v-divider class="my-2" />

          <v-card-text>
            <p class="text-caption text-medium-emphasis mb-2">Quick Links</p>
            <v-btn
              block
              variant="tonal"
              color="primary"
              size="small"
              class="mb-2"
              to="/ambient-session"
            >
              <v-icon icon="mdi-broadcast" class="mr-2" size="16" />
              Try Ambient AI
            </v-btn>
            <v-btn
              block
              variant="tonal"
              color="secondary"
              size="small"
              class="mb-2"
              to="/dictation"
            >
              <v-icon icon="mdi-microphone-message" class="mr-2" size="16" />
              Try Dictation
            </v-btn>
            <v-btn
              block
              variant="tonal"
              color="info"
              size="small"
              to="/async-transcription"
            >
              <v-icon icon="mdi-file-upload" class="mr-2" size="16" />
              Upload File
            </v-btn>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Main Content -->
      <v-col cols="12" md="9">
        <!-- Overview Section -->
        <v-card id="overview" class="glass-card mb-6">
          <v-card-title class="text-h5">
            <v-icon icon="mdi-information" class="mr-2" color="primary" />
            Overview
          </v-card-title>
          <v-card-text>
            <p class="text-body-1 mb-4">
              <strong>XStek AI Medical Transcription</strong> is an AI-powered platform that transforms 
              clinical conversations into accurate, structured documentation. The platform uses 
              advanced speech recognition technology specifically trained for healthcare terminology.
            </p>
            
            <v-alert type="info" variant="tonal" class="mb-4">
              <strong>Three Ways to Transcribe:</strong>
              <ol class="mt-2 mb-0">
                <li><strong>File Transcription</strong> - Upload pre-recorded audio files</li>
                <li><strong>Ambient AI</strong> - Real-time conversation capture with fact extraction</li>
                <li><strong>Dictation</strong> - Voice-controlled note-taking with templates</li>
              </ol>
            </v-alert>
          </v-card-text>
        </v-card>

        <!-- Transcription Modes Section -->
        <v-card id="modes" class="glass-card mb-6">
          <v-card-title class="text-h5">
            <v-icon icon="mdi-swap-horizontal" class="mr-2" color="info" />
            Understanding Transcription Modes
          </v-card-title>
          <v-card-text>
            <p class="text-body-1 mb-4">
              There are different ways to convert speech to text in this platform. Here's what each term means:
            </p>

            <v-table class="mb-4">
              <thead>
                <tr>
                  <th style="width: 15%">Term</th>
                  <th style="width: 25%">What It Does</th>
                  <th style="width: 30%">Example</th>
                  <th style="width: 30%">When to Use</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td>
                    <v-chip color="primary" size="small">Transcribe</v-chip>
                  </td>
                  <td>
                    <strong>Upload & Process</strong><br>
                    Upload a pre-recorded audio file for batch processing. The file is sent to the server, processed, and returns the complete text.
                  </td>
                  <td class="text-body-2">
                    <em>Upload "consultation.mp3" → Wait 10 seconds → Get full text</em><br>
                    <code class="text-caption">POST /api/transcribe/upload</code>
                  </td>
                  <td class="text-body-2">
                    • Recorded consultations<br>
                    • Meeting recordings<br>
                    • Pre-recorded dictations
                  </td>
                </tr>
                <tr>
                  <td>
                    <v-chip color="secondary" size="small">Stream</v-chip>
                  </td>
                  <td>
                    <strong>Real-Time Audio</strong><br>
                    Send audio in real-time via WebSocket. Text appears as you speak. Used by Ambient AI.
                  </td>
                  <td class="text-body-2">
                    <em>Speak → See words appear instantly → Continue talking</em><br>
                    <code class="text-caption">WebSocket /api/ambient/ws</code>
                  </td>
                  <td class="text-body-2">
                    • Live consultations<br>
                    • Real-time note-taking<br>
                    • Clinical fact extraction
                  </td>
                </tr>
                <tr>
                  <td>
                    <v-chip color="success" size="small">Dictation</v-chip>
                  </td>
                  <td>
                    <strong>Stateless Voice-to-Text</strong><br>
                    Real-time transcription without storing session state. Supports voice commands for editing.
                  </td>
                  <td class="text-body-2">
                    <em>"Patient presents with cough period new paragraph" → Formatted text</em><br>
                    <code class="text-caption">WebSocket /api/dictation/ws</code>
                  </td>
                  <td class="text-body-2">
                    • Quick notes<br>
                    • Template filling<br>
                    • Voice commands
                  </td>
                </tr>
                <tr>
                  <td>
                    <v-chip color="warning" size="small">Transcript</v-chip>
                  </td>
                  <td>
                    <strong>The Output Text</strong><br>
                    The resulting text document from any transcription method. Contains the words spoken in the audio.
                  </td>
                  <td class="text-body-2">
                    <em>Input: Audio → Output: "Patient reports headache for 2 weeks..."</em>
                  </td>
                  <td class="text-body-2">
                    • Copy to EHR<br>
                    • Generate documents<br>
                    • Review/edit
                  </td>
                </tr>
              </tbody>
            </v-table>

            <v-alert type="info" variant="tonal">
              <strong>Key Difference:</strong> 
              <span class="ml-1">
                <strong>Transcribe</strong> = the action (upload file), 
                <strong>Stream</strong> = the method (real-time), 
                <strong>Dictation</strong> = the mode (voice commands), 
                <strong>Transcript</strong> = the result (the text).
              </span>
            </v-alert>
          </v-card-text>
        </v-card>

        <!-- SOAP Note Components Section -->
        <v-card id="soap" class="glass-card mb-6">
          <v-card-title class="text-h5">
            <v-icon icon="mdi-clipboard-pulse" class="mr-2" color="success" />
            SOAP Note Components
          </v-card-title>
          <v-card-text>
            <p class="text-body-1 mb-4">
              For accurate SOAP notes, the AI needs to extract specific clinical data. Here's what each component means:
            </p>

            <v-row>
              <!-- Vital Signs -->
              <v-col cols="12" md="6">
                <v-card variant="outlined" class="pa-4 h-100">
                  <div class="d-flex align-center mb-3">
                    <v-avatar color="success" size="40" class="mr-3">
                      <v-icon icon="mdi-heart-pulse" color="white" />
                    </v-avatar>
                    <div>
                      <h4 class="text-subtitle-1 font-weight-bold mb-0">Vital Signs</h4>
                      <p class="text-caption text-medium-emphasis mb-0">Objective measurements</p>
                    </div>
                  </div>
                  <p class="text-body-2 mb-3">
                    Measurable physiological parameters that indicate a patient's basic body functions.
                  </p>
                  <v-divider class="mb-3" />
                  <p class="text-subtitle-2 font-weight-medium mb-2">Includes:</p>
                  <ul class="text-body-2 mb-3">
                    <li><strong>Blood Pressure</strong> - e.g., "140/90 mmHg"</li>
                    <li><strong>Heart Rate</strong> - e.g., "88 beats per minute"</li>
                    <li><strong>Temperature</strong> - e.g., "37.5°C"</li>
                    <li><strong>Respiratory Rate</strong> - e.g., "16 breaths per minute"</li>
                    <li><strong>Oxygen Saturation</strong> - e.g., "98% on room air"</li>
                    <li><strong>Weight/Height/BMI</strong></li>
                  </ul>
                  <v-alert type="success" variant="tonal" density="compact">
                    <strong>Say:</strong> "Blood pressure is 140 over 90, heart rate 88, temperature 37.5"
                  </v-alert>
                </v-card>
              </v-col>

              <!-- Medications -->
              <v-col cols="12" md="6">
                <v-card variant="outlined" class="pa-4 h-100">
                  <div class="d-flex align-center mb-3">
                    <v-avatar color="primary" size="40" class="mr-3">
                      <v-icon icon="mdi-pill" color="white" />
                    </v-avatar>
                    <div>
                      <h4 class="text-subtitle-1 font-weight-bold mb-0">Medications</h4>
                      <p class="text-caption text-medium-emphasis mb-0">Current & prescribed drugs</p>
                    </div>
                  </div>
                  <p class="text-body-2 mb-3">
                    Drugs the patient is currently taking or being prescribed, including dosage and frequency.
                  </p>
                  <v-divider class="mb-3" />
                  <p class="text-subtitle-2 font-weight-medium mb-2">Include:</p>
                  <ul class="text-body-2 mb-3">
                    <li><strong>Drug Name</strong> - Generic or brand name</li>
                    <li><strong>Dose</strong> - e.g., "500 milligrams"</li>
                    <li><strong>Frequency</strong> - e.g., "twice daily"</li>
                    <li><strong>Route</strong> - oral, IV, topical, etc.</li>
                    <li><strong>Duration</strong> - "for 7 days"</li>
                  </ul>
                  <v-alert type="success" variant="tonal" density="compact">
                    <strong>Say:</strong> "Patient takes Metformin 500 milligrams twice daily and Lisinopril 10 milligrams once daily"
                  </v-alert>
                </v-card>
              </v-col>

              <!-- Allergies -->
              <v-col cols="12" md="6">
                <v-card variant="outlined" class="pa-4 h-100">
                  <div class="d-flex align-center mb-3">
                    <v-avatar color="error" size="40" class="mr-3">
                      <v-icon icon="mdi-alert-circle" color="white" />
                    </v-avatar>
                    <div>
                      <h4 class="text-subtitle-1 font-weight-bold mb-0">Allergies</h4>
                      <p class="text-caption text-medium-emphasis mb-0">Adverse reactions</p>
                    </div>
                  </div>
                  <p class="text-body-2 mb-3">
                    Substances that cause allergic reactions in the patient, including the type of reaction.
                  </p>
                  <v-divider class="mb-3" />
                  <p class="text-subtitle-2 font-weight-medium mb-2">Include:</p>
                  <ul class="text-body-2 mb-3">
                    <li><strong>Allergen</strong> - Drug, food, or environmental</li>
                    <li><strong>Reaction Type</strong> - Rash, anaphylaxis, nausea</li>
                    <li><strong>Severity</strong> - Mild, moderate, severe</li>
                    <li><strong>NKDA</strong> - "No known drug allergies"</li>
                  </ul>
                  <v-alert type="success" variant="tonal" density="compact">
                    <strong>Say:</strong> "Patient is allergic to Penicillin which causes a skin rash" or "No known drug allergies"
                  </v-alert>
                </v-card>
              </v-col>

              <!-- Diagnoses -->
              <v-col cols="12" md="6">
                <v-card variant="outlined" class="pa-4 h-100">
                  <div class="d-flex align-center mb-3">
                    <v-avatar color="warning" size="40" class="mr-3">
                      <v-icon icon="mdi-stethoscope" color="white" />
                    </v-avatar>
                    <div>
                      <h4 class="text-subtitle-1 font-weight-bold mb-0">Diagnoses</h4>
                      <p class="text-caption text-medium-emphasis mb-0">Clinical conclusions</p>
                    </div>
                  </div>
                  <p class="text-body-2 mb-3">
                    Medical conditions identified based on symptoms, history, and examination findings.
                  </p>
                  <v-divider class="mb-3" />
                  <p class="text-subtitle-2 font-weight-medium mb-2">Include:</p>
                  <ul class="text-body-2 mb-3">
                    <li><strong>Primary Diagnosis</strong> - Main condition</li>
                    <li><strong>Secondary Diagnoses</strong> - Related conditions</li>
                    <li><strong>Differential Diagnoses</strong> - Possibilities to rule out</li>
                    <li><strong>Chronic Conditions</strong> - Existing history</li>
                  </ul>
                  <v-alert type="success" variant="tonal" density="compact">
                    <strong>Say:</strong> "Assessment shows tension headache. Patient also has history of Type 2 Diabetes and Hypertension"
                  </v-alert>
                </v-card>
              </v-col>
            </v-row>

            <v-alert type="warning" variant="tonal" class="mt-4">
              <v-icon icon="mdi-lightbulb" class="mr-2" />
              <strong>Pro Tip:</strong> For complete SOAP notes, always mention all four components during the consultation. 
              The AI extracts what it hears — if you don't say it, it won't appear in the note.
            </v-alert>
          </v-card-text>
        </v-card>

        <!-- Glossary Section -->
        <v-card id="glossary" class="glass-card mb-6">
          <v-card-title class="text-h5">
            <v-icon icon="mdi-book-alphabet" class="mr-2" color="secondary" />
            Glossary of Terms
          </v-card-title>
          <v-card-text>
            <v-expansion-panels variant="accordion">
              <v-expansion-panel
                v-for="term in glossaryTerms"
                :key="term.term"
              >
                <v-expansion-panel-title>
                  <div class="d-flex align-center">
                    <v-chip :color="term.color" size="small" class="mr-3">
                      {{ term.category }}
                    </v-chip>
                    <strong>{{ term.term }}</strong>
                  </div>
                </v-expansion-panel-title>
                <v-expansion-panel-text>
                  <p class="text-body-1 mb-3">{{ term.definition }}</p>
                  <v-alert v-if="term.example" type="success" variant="tonal" density="compact">
                    <strong>Example:</strong> {{ term.example }}
                  </v-alert>
                  <p v-if="term.note" class="text-caption text-medium-emphasis mt-2 mb-0">
                    <v-icon icon="mdi-lightbulb" size="14" class="mr-1" />
                    {{ term.note }}
                  </p>
                </v-expansion-panel-text>
              </v-expansion-panel>
            </v-expansion-panels>
          </v-card-text>
        </v-card>

        <!-- Features Section -->
        <v-card id="features" class="glass-card mb-6">
          <v-card-title class="text-h5">
            <v-icon icon="mdi-star" class="mr-2" color="warning" />
            Features
          </v-card-title>
          <v-card-text>
            <v-tabs v-model="featureTab" color="primary" class="mb-4">
              <v-tab value="file">File Transcription</v-tab>
              <v-tab value="ambient">Ambient AI</v-tab>
              <v-tab value="dictation">Dictation</v-tab>
            </v-tabs>

            <v-window v-model="featureTab">
              <!-- File Transcription -->
              <v-window-item value="file">
                <div class="pa-4">
                  <h3 class="text-h6 mb-3">
                    <v-icon icon="mdi-file-upload" class="mr-2" color="primary" />
                    File Transcription (Async)
                  </h3>
                  <p class="text-body-1 mb-4">
                    Upload pre-recorded audio files for batch processing. Ideal for transcribing 
                    recorded consultations, meetings, or dictated notes.
                  </p>
                  
                  <v-row>
                    <v-col cols="12" sm="6">
                      <v-card variant="outlined" class="pa-3">
                        <h4 class="text-subtitle-1 font-weight-bold mb-2">
                          <v-icon icon="mdi-check-circle" color="success" class="mr-1" />
                          Supported Formats
                        </h4>
                        <v-chip-group>
                          <v-chip size="small">WAV</v-chip>
                          <v-chip size="small">MP3</v-chip>
                          <v-chip size="small">M4A</v-chip>
                          <v-chip size="small">FLAC</v-chip>
                          <v-chip size="small">OGG</v-chip>
                          <v-chip size="small">WebM</v-chip>
                        </v-chip-group>
                      </v-card>
                    </v-col>
                    <v-col cols="12" sm="6">
                      <v-card variant="outlined" class="pa-3">
                        <h4 class="text-subtitle-1 font-weight-bold mb-2">
                          <v-icon icon="mdi-cog" color="info" class="mr-1" />
                          Capabilities
                        </h4>
                        <ul class="text-body-2">
                          <li>Speaker diarization</li>
                          <li>Multi-language detection</li>
                          <li>Timestamp generation</li>
                          <li>Document generation</li>
                        </ul>
                      </v-card>
                    </v-col>
                  </v-row>

                  <v-btn color="primary" class="mt-4" to="/async-transcription">
                    <v-icon icon="mdi-arrow-right" class="mr-2" />
                    Go to File Transcription
                  </v-btn>
                </div>
              </v-window-item>

              <!-- Ambient AI -->
              <v-window-item value="ambient">
                <div class="pa-4">
                  <h3 class="text-h6 mb-3">
                    <v-icon icon="mdi-broadcast" class="mr-2" color="secondary" />
                    Ambient AI (Real-Time)
                  </h3>
                  <p class="text-body-1 mb-4">
                    Capture live conversations between clinicians and patients. The AI extracts 
                    clinical facts in real-time and automatically generates documentation.
                  </p>
                  
                  <v-alert type="info" variant="tonal" class="mb-4">
                    <strong>How it works:</strong>
                    <ol class="mt-2 mb-0">
                      <li>Click "Start New Recording" → Audio streams to AI</li>
                      <li>Real-time transcription appears as you speak</li>
                      <li>Clinical facts (medications, vitals, allergies) are extracted</li>
                      <li>Stop recording → Select template → Generate document</li>
                      <li>Transcript auto-collapses, report expands for easy review</li>
                    </ol>
                  </v-alert>

                  <v-row>
                    <v-col cols="12" sm="6">
                      <v-card variant="outlined" class="pa-3">
                        <h4 class="text-subtitle-1 font-weight-bold mb-2">
                          <v-icon icon="mdi-check-circle" color="success" class="mr-1" />
                          What Gets Extracted
                        </h4>
                        <v-chip-group>
                          <v-chip size="small" color="primary">Medications</v-chip>
                          <v-chip size="small" color="secondary">Allergies</v-chip>
                          <v-chip size="small" color="success">Vital Signs</v-chip>
                          <v-chip size="small" color="warning">Diagnoses</v-chip>
                          <v-chip size="small" color="info">Symptoms</v-chip>
                          <v-chip size="small" color="error">Medical History</v-chip>
                        </v-chip-group>
                      </v-card>
                    </v-col>
                    <v-col cols="12" sm="6">
                      <v-card variant="outlined" class="pa-3">
                        <h4 class="text-subtitle-1 font-weight-bold mb-2">
                          <v-icon icon="mdi-file-document" color="info" class="mr-1" />
                          Document Templates
                        </h4>
                        <ul class="text-body-2">
                          <li>SOAP Note</li>
                          <li>History & Physical</li>
                          <li>Emergency Note</li>
                          <li>Discharge Summary</li>
                          <li>+ 5 more templates</li>
                        </ul>
                      </v-card>
                    </v-col>
                  </v-row>

                  <v-btn color="secondary" class="mt-4" to="/ambient-session">
                    <v-icon icon="mdi-arrow-right" class="mr-2" />
                    Go to Ambient AI
                  </v-btn>
                </div>
              </v-window-item>

              <!-- Dictation -->
              <v-window-item value="dictation">
                <div class="pa-4">
                  <h3 class="text-h6 mb-3">
                    <v-icon icon="mdi-microphone-message" class="mr-2" color="info" />
                    Dictation (Voice-Controlled)
                  </h3>
                  <p class="text-body-1 mb-4">
                    Stateless real-time speech-to-text with automatic punctuation, number formatting, 
                    and voice commands for hands-free note creation.
                  </p>
                  
                  <v-alert type="success" variant="tonal" class="mb-4">
                    <strong>Voice Commands Available:</strong>
                    <div class="mt-2">
                      <v-chip size="small" class="mr-1 mb-1">"Insert GP consultation template"</v-chip>
                      <v-chip size="small" class="mr-1 mb-1">"Go to clinical findings"</v-chip>
                      <v-chip size="small" class="mr-1 mb-1">"New paragraph"</v-chip>
                      <v-chip size="small" class="mr-1 mb-1">"Delete last sentence"</v-chip>
                      <v-chip size="small" class="mr-1 mb-1">"Undo"</v-chip>
                    </div>
                  </v-alert>

                  <v-row>
                    <v-col cols="12" sm="6">
                      <v-card variant="outlined" class="pa-3">
                        <h4 class="text-subtitle-1 font-weight-bold mb-2">
                          <v-icon icon="mdi-hospital-building" color="success" class="mr-1" />
                          NHS Templates
                        </h4>
                        <ul class="text-body-2">
                          <li>GP Consultation</li>
                          <li>Clinical Note</li>
                          <li>Discharge Summary</li>
                          <li>A&E Triage</li>
                          <li>Progress Note</li>
                          <li>+ 5 more templates</li>
                        </ul>
                      </v-card>
                    </v-col>
                    <v-col cols="12" sm="6">
                      <v-card variant="outlined" class="pa-3">
                        <h4 class="text-subtitle-1 font-weight-bold mb-2">
                          <v-icon icon="mdi-format-text" color="info" class="mr-1" />
                          Auto Formatting
                        </h4>
                        <ul class="text-body-2">
                          <li>Automatic punctuation</li>
                          <li>Numbers as digits</li>
                          <li>Date/time formatting</li>
                          <li>Medical abbreviations</li>
                        </ul>
                      </v-card>
                    </v-col>
                  </v-row>

                  <v-btn color="info" class="mt-4" to="/dictation">
                    <v-icon icon="mdi-arrow-right" class="mr-2" />
                    Go to Dictation
                  </v-btn>
                </div>
              </v-window-item>
            </v-window>
          </v-card-text>
        </v-card>

        <!-- Best Practices Section -->
        <v-card id="best-practices" class="glass-card mb-6">
          <v-card-title class="text-h5">
            <v-icon icon="mdi-lightbulb" class="mr-2" color="warning" />
            Best Practices
          </v-card-title>
          <v-card-text>
            <v-row>
              <v-col cols="12" md="6">
                <v-card variant="tonal" color="success" class="pa-4 h-100">
                  <h4 class="text-subtitle-1 font-weight-bold mb-3">
                    <v-icon icon="mdi-check" class="mr-1" />
                    DO
                  </h4>
                  <ul class="text-body-2">
                    <li class="mb-2">Use a quality external microphone</li>
                    <li class="mb-2">Speak clearly at moderate pace</li>
                    <li class="mb-2">Mention specific numbers: <em>"Blood pressure 140 over 90"</em></li>
                    <li class="mb-2">State medication doses: <em>"Metformin 500 milligrams twice daily"</em></li>
                    <li class="mb-2">Record for at least 2 minutes for best fact extraction</li>
                    <li class="mb-2">Wait 2-3 seconds before stopping recording</li>
                    <li class="mb-2">Use quiet environment</li>
                    <li class="mb-2">Save sessions and manage them before reaching the 5 session limit</li>
                  </ul>
                </v-card>
              </v-col>
              <v-col cols="12" md="6">
                <v-card variant="tonal" color="error" class="pa-4 h-100">
                  <h4 class="text-subtitle-1 font-weight-bold mb-3">
                    <v-icon icon="mdi-close" class="mr-1" />
                    DON'T
                  </h4>
                  <ul class="text-body-2">
                    <li class="mb-2">Use vague terms: <em>"BP is normal"</em> (say the numbers)</li>
                    <li class="mb-2">Talk over each other</li>
                    <li class="mb-2">Stop recording immediately after speaking</li>
                    <li class="mb-2">Record in noisy environments</li>
                    <li class="mb-2">Use abbreviations verbally: say "blood pressure" not "B-P"</li>
                    <li class="mb-2">Rush through clinical information</li>
                    <li class="mb-2">Expect facts from very short sessions (&lt;1 min)</li>
                  </ul>
                </v-card>
              </v-col>
            </v-row>

            <v-alert type="info" variant="tonal" class="mt-4">
              <strong>For Complete SOAP Notes, Always Mention:</strong>
              <v-row class="mt-2">
                <v-col cols="6" sm="3">
                  <v-chip color="primary" size="small" block>Vital Signs</v-chip>
                </v-col>
                <v-col cols="6" sm="3">
                  <v-chip color="secondary" size="small" block>Medications</v-chip>
                </v-col>
                <v-col cols="6" sm="3">
                  <v-chip color="warning" size="small" block>Allergies</v-chip>
                </v-col>
                <v-col cols="6" sm="3">
                  <v-chip color="success" size="small" block>Diagnoses</v-chip>
                </v-col>
              </v-row>
            </v-alert>
          </v-card-text>
        </v-card>

        <!-- Troubleshooting Section -->
        <v-card id="troubleshooting" class="glass-card mb-6">
          <v-card-title class="text-h5">
            <v-icon icon="mdi-wrench" class="mr-2" color="error" />
            Troubleshooting
          </v-card-title>
          <v-card-text>
            <v-table>
              <thead>
                <tr>
                  <th>Issue</th>
                  <th>Cause</th>
                  <th>Solution</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="issue in troubleshootingItems" :key="issue.issue">
                  <td><strong>{{ issue.issue }}</strong></td>
                  <td>{{ issue.cause }}</td>
                  <td>{{ issue.solution }}</td>
                </tr>
              </tbody>
            </v-table>
          </v-card-text>
        </v-card>

        <!-- Sample Conversation Section -->
        <v-card id="samples" class="glass-card mb-6">
          <v-card-title class="text-h5">
            <v-icon icon="mdi-message-text" class="mr-2" color="info" />
            Sample Conversation
          </v-card-title>
          <v-card-text>
            <v-alert type="success" variant="tonal" class="mb-4">
              <strong>Use this sample for testing Ambient AI:</strong>
            </v-alert>
            
            <v-card variant="outlined" class="pa-4 conversation-sample">
              <div class="conversation-line doctor">
                <v-chip size="small" color="primary" class="mr-2">Doctor</v-chip>
                "Good morning. What brings you in today?"
              </div>
              <div class="conversation-line patient">
                <v-chip size="small" color="secondary" class="mr-2">Patient</v-chip>
                "I've been having severe headaches for the past two weeks."
              </div>
              <div class="conversation-line doctor">
                <v-chip size="small" color="primary" class="mr-2">Doctor</v-chip>
                "Any other symptoms?"
              </div>
              <div class="conversation-line patient">
                <v-chip size="small" color="secondary" class="mr-2">Patient</v-chip>
                "Yes, some nausea in the mornings and my vision gets blurry sometimes."
              </div>
              <div class="conversation-line doctor">
                <v-chip size="small" color="primary" class="mr-2">Doctor</v-chip>
                "What's your medical history?"
              </div>
              <div class="conversation-line patient">
                <v-chip size="small" color="secondary" class="mr-2">Patient</v-chip>
                "I have Type 2 Diabetes. I take Metformin 500 milligrams twice daily."
              </div>
              <div class="conversation-line doctor">
                <v-chip size="small" color="primary" class="mr-2">Doctor</v-chip>
                "Any allergies?"
              </div>
              <div class="conversation-line patient">
                <v-chip size="small" color="secondary" class="mr-2">Patient</v-chip>
                "I'm allergic to Penicillin. It causes a rash."
              </div>
              <div class="conversation-line doctor">
                <v-chip size="small" color="primary" class="mr-2">Doctor</v-chip>
                "Let me check your vitals. Blood pressure 140 over 90, heart rate 88 beats per minute."
              </div>
              <div class="conversation-line doctor">
                <v-chip size="small" color="primary" class="mr-2">Doctor</v-chip>
                "Based on my assessment, you have tension headaches. I'm prescribing Ibuprofen 400 milligrams as needed."
              </div>
            </v-card>

            <v-btn 
              color="primary" 
              variant="tonal" 
              class="mt-4"
              @click="copySampleConversation"
            >
              <v-icon icon="mdi-content-copy" class="mr-2" />
              Copy Sample Text
            </v-btn>
          </v-card-text>
        </v-card>

        <!-- Architecture Section -->
        <v-card id="architecture" class="glass-card mb-6">
          <v-card-title class="text-h5">
            <v-icon icon="mdi-sitemap" class="mr-2" color="primary" />
            Technical Architecture
          </v-card-title>
          <v-card-text>
            <v-row>
              <v-col cols="12" md="4">
                <v-card variant="outlined" class="pa-4 text-center h-100">
                  <v-icon icon="mdi-vuejs" size="48" color="success" class="mb-2" />
                  <h4 class="text-subtitle-1 font-weight-bold">Frontend</h4>
                  <p class="text-body-2 text-medium-emphasis">Vue 3 + Vuetify 3</p>
                  <v-chip-group class="justify-center">
                    <v-chip size="x-small">TypeScript</v-chip>
                    <v-chip size="x-small">WebSocket</v-chip>
                    <v-chip size="x-small">MediaRecorder</v-chip>
                  </v-chip-group>
                </v-card>
              </v-col>
              <v-col cols="12" md="4">
                <v-card variant="outlined" class="pa-4 text-center h-100">
                  <v-icon icon="mdi-language-go" size="48" color="info" class="mb-2" />
                  <h4 class="text-subtitle-1 font-weight-bold">Backend</h4>
                  <p class="text-body-2 text-medium-emphasis">Go + Fiber</p>
                  <v-chip-group class="justify-center">
                    <v-chip size="x-small">REST API</v-chip>
                    <v-chip size="x-small">WebSocket Proxy</v-chip>
                    <v-chip size="x-small">OAuth2</v-chip>
                  </v-chip-group>
                </v-card>
              </v-col>
              <v-col cols="12" md="4">
                <v-card variant="outlined" class="pa-4 text-center h-100">
                  <v-icon icon="mdi-brain" size="48" color="secondary" class="mb-2" />
                  <h4 class="text-subtitle-1 font-weight-bold">AI Engine</h4>
                  <p class="text-body-2 text-medium-emphasis">Corti FactsR™</p>
                  <v-chip-group class="justify-center">
                    <v-chip size="x-small">Speech-to-Text</v-chip>
                    <v-chip size="x-small">Fact Extraction</v-chip>
                    <v-chip size="x-small">Doc Generation</v-chip>
                  </v-chip-group>
                </v-card>
              </v-col>
            </v-row>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Copy Snackbar -->
    <v-snackbar v-model="showCopied" color="success" timeout="2000">
      Copied to clipboard!
    </v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const activeSection = ref('overview')
const featureTab = ref('file')
const showCopied = ref(false)

const sections = [
  { id: 'overview', title: 'Overview', icon: 'mdi-information' },
  { id: 'modes', title: 'Transcription Modes', icon: 'mdi-swap-horizontal' },
  { id: 'soap', title: 'SOAP Note Components', icon: 'mdi-clipboard-pulse' },
  { id: 'glossary', title: 'Glossary', icon: 'mdi-book-alphabet' },
  { id: 'features', title: 'Features', icon: 'mdi-star' },
  { id: 'best-practices', title: 'Best Practices', icon: 'mdi-lightbulb' },
  { id: 'troubleshooting', title: 'Troubleshooting', icon: 'mdi-wrench' },
  { id: 'samples', title: 'Sample Conversation', icon: 'mdi-message-text' },
  { id: 'architecture', title: 'Architecture', icon: 'mdi-sitemap' },
]

const glossaryTerms = [
  {
    term: 'Clinical Facts',
    category: 'Core Concept',
    color: 'primary',
    definition: 'Structured pieces of clinical information extracted from conversations, such as medications, allergies, vital signs, diagnoses, symptoms, and medical history. Facts are validated by AI before being used for documentation.',
    example: 'From "I take Metformin 500mg twice daily" → Medication fact: "Metformin 500mg BID"',
    note: 'Facts are the foundation of accurate clinical documentation. The more explicit you are, the better the extraction.'
  },
  {
    term: 'FactsR™',
    category: 'Technology',
    color: 'secondary',
    definition: 'The AI engine that powers clinical fact extraction. FactsR™ uses a recursive fact-first reasoning loop to identify, validate, and structure clinical knowledge in real-time. It processes audio approximately every 60 seconds.',
    example: 'Listens to conversation → Identifies clinical entities → Validates accuracy → Outputs structured facts',
    note: 'FactsR™ is designed to reduce "note bloat" and improve documentation accuracy compared to simple transcription.'
  },
  {
    term: 'Ambient AI',
    category: 'Feature',
    color: 'success',
    definition: 'A real-time AI assistant that "listens" to doctor-patient conversations, transcribes speech, extracts clinical facts, and generates documentation automatically. No manual note-taking required during the encounter.',
    example: 'Start recording → Have conversation → Stop → Select template → Generate document',
    note: 'Best results with 2+ minutes of recording and clear clinical content. Max 5 saved sessions per user.'
  },
  {
    term: 'SOAP Note',
    category: 'Document Type',
    color: 'warning',
    definition: 'Subjective, Objective, Assessment, Plan - A standardized format for medical documentation. Subjective = patient\'s symptoms; Objective = exam findings; Assessment = diagnosis; Plan = treatment.',
    example: 'S: Headache for 2 weeks | O: BP 140/90 | A: Tension headache | P: Ibuprofen PRN',
    note: 'SOAP notes are the most common format for clinical documentation worldwide.'
  },
  {
    term: 'Speaker Diarization',
    category: 'Technology',
    color: 'info',
    definition: 'The process of identifying and separating different speakers in an audio recording. Allows the system to distinguish between doctor and patient voices.',
    example: 'Doctor: "What brings you in?" → Speaker 0 | Patient: "Headache" → Speaker 1',
    note: 'Works best with multi-channel audio (separate microphones). Single mic has limitations.'
  },
  {
    term: 'Transcription',
    category: 'Core Concept',
    color: 'primary',
    definition: 'The conversion of spoken audio into written text. Can be real-time (streaming) or asynchronous (file upload).',
    example: 'Audio: "Blood pressure is 140 over 90" → Text: "Blood pressure is 140 over 90"',
    note: 'Transcription is the first step; fact extraction provides the clinical intelligence.'
  },
  {
    term: 'Stateless Dictation',
    category: 'Feature',
    color: 'secondary',
    definition: 'A transcription mode where each session is independent with no persistent state. Audio is converted to text in real-time without creating an "interaction" or storing session history.',
    example: 'Start dictating → See text appear → Copy note → Done (no session saved)',
    note: 'Ideal for quick notes and voice commands. Does not extract clinical facts.'
  },
  {
    term: 'Voice Commands',
    category: 'Feature',
    color: 'success',
    definition: 'Spoken instructions that control the application instead of being transcribed. Used for navigation, editing, and template insertion.',
    example: '"Go to presenting complaint" → Cursor moves to that section',
    note: 'Commands must be spoken clearly and match defined phrases.'
  },
  {
    term: 'NHS-Compliant Templates',
    category: 'Document Type',
    color: 'warning',
    definition: 'Clinical note templates designed to meet UK National Health Service documentation standards, including PRSB (Professional Record Standards Body) standards, SNOMED CT coding, and dm+d medication references.',
    example: 'GP Consultation template with proper NHS structure and terminology',
    note: 'Includes 10 templates: GP Consultation, Clinical Note, Discharge Summary, A&E Triage, etc.'
  },
  {
    term: 'WebSocket Streaming',
    category: 'Technology',
    color: 'info',
    definition: 'A protocol that enables real-time, bidirectional communication between the browser and server. Used for sending audio and receiving transcriptions without page refreshes.',
    example: 'Browser sends audio chunks → Server sends back transcript updates → Both happen simultaneously',
    note: 'Enables the real-time experience of seeing transcripts appear as you speak.'
  },
  {
    term: 'Interaction',
    category: 'Core Concept',
    color: 'primary',
    definition: 'A session container in the API that holds all data for a clinical encounter: transcripts, facts, and generated documents. Created when starting Ambient AI or uploading a file.',
    example: 'Start session → Interaction ID created → All data stored under that ID',
    note: 'Used for tracking, billing, and retrieving session data.'
  },
  {
    term: 'Document Generation',
    category: 'Feature',
    color: 'secondary',
    definition: 'The process of creating structured clinical documents (like SOAP notes) from transcripts and extracted facts. The AI summarizes and organizes information into the requested format. You can also change templates and regenerate from saved sessions.',
    example: 'Facts + Transcript → AI Processing → Formatted SOAP Note',
    note: 'Quality depends on the facts extracted; empty sections mean missing clinical content. Templates can be changed in saved sessions.'
  },
]

const troubleshootingItems = [
  {
    issue: 'No clinical facts appearing',
    cause: 'Recording too short or no clinical content spoken',
    solution: 'Record 2+ minutes; mention medications, vitals, diagnoses explicitly'
  },
  {
    issue: 'Empty Objective section in SOAP',
    cause: 'No vital signs or exam findings mentioned',
    solution: 'Say "Blood pressure 140 over 90, heart rate 88" explicitly'
  },
  {
    issue: 'Medication name misheard',
    cause: 'Speech recognition limitation',
    solution: 'Speak medication names slowly and clearly; spell if needed'
  },
  {
    issue: 'Speaker labels wrong',
    cause: 'Single microphone limitation',
    solution: 'Use separate microphones for doctor/patient in production'
  },
  {
    issue: 'Voice commands not working',
    cause: 'Phrase not matching defined commands',
    solution: 'Use exact phrases: "Insert GP consultation template"'
  },
  {
    issue: 'Transcription stops working',
    cause: 'WebSocket connection lost',
    solution: 'Stop and restart the session; check internet connection'
  },
  {
    issue: 'Document generation fails',
    cause: 'Not enough content in transcript',
    solution: 'Ensure at least 2 minutes of clinical conversation'
  },
  {
    issue: 'Cannot start new recording',
    cause: 'Maximum 5 saved sessions reached',
    solution: 'Go to Saved Sessions and delete older sessions to continue'
  },
]

const scrollToSection = (sectionId: string) => {
  const element = document.getElementById(sectionId)
  if (element) {
    element.scrollIntoView({ behavior: 'smooth', block: 'start' })
    activeSection.value = sectionId
  }
}

const sampleConversation = `Doctor: "Good morning. What brings you in today?"
Patient: "I've been having severe headaches for the past two weeks."
Doctor: "Any other symptoms?"
Patient: "Yes, some nausea in the mornings and my vision gets blurry sometimes."
Doctor: "What's your medical history?"
Patient: "I have Type 2 Diabetes. I take Metformin 500 milligrams twice daily."
Doctor: "Any allergies?"
Patient: "I'm allergic to Penicillin. It causes a rash."
Doctor: "Let me check your vitals. Blood pressure 140 over 90, heart rate 88 beats per minute."
Doctor: "Based on my assessment, you have tension headaches. I'm prescribing Ibuprofen 400 milligrams as needed."`

const copySampleConversation = async () => {
  try {
    await navigator.clipboard.writeText(sampleConversation)
    showCopied.value = true
  } catch (err) {
    console.error('Failed to copy:', err)
  }
}

// Track active section on scroll
const handleScroll = () => {
  const sectionElements = sections.map(s => ({
    id: s.id,
    element: document.getElementById(s.id)
  }))

  for (const section of sectionElements.reverse()) {
    if (section.element) {
      const rect = section.element.getBoundingClientRect()
      if (rect.top <= 150) {
        activeSection.value = section.id
        break
      }
    }
  }
}

onMounted(() => {
  window.addEventListener('scroll', handleScroll)
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
})
</script>

<style scoped>
.documentation-page {
  min-height: calc(100vh - 120px);
  padding-top: 1rem;
}

.sticky-nav {
  position: sticky;
  top: 80px;
}

.conversation-sample {
  background: rgba(var(--v-theme-surface-variant), 0.3);
  font-family: 'Space Grotesk', sans-serif;
}

.conversation-line {
  padding: 8px 0;
  border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.1);
}

.conversation-line:last-child {
  border-bottom: none;
}

.conversation-line.doctor {
  padding-left: 0;
}

.conversation-line.patient {
  padding-left: 20px;
}
</style>

