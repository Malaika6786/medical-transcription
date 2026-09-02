# xstek Medical Transcription
## Complete Documentation Guide

---

**Version:** 1.0  
**Last Updated:** January 2026  
**Platform:** xstek Medical Transcription Demo

---

## Table of Contents

1. [Overview](#overview)
2. [Understanding Transcription Modes](#understanding-transcription-modes)
3. [SOAP Note Components](#soap-note-components)
4. [Glossary of Terms](#glossary-of-terms)
5. [Features](#features)
   - [File Transcription (Async)](#file-transcription-async)
   - [Ambient AI (Real-Time)](#ambient-ai-real-time)
   - [Dictation (Voice-Controlled)](#dictation-voice-controlled)
6. [Best Practices](#best-practices)
7. [Troubleshooting](#troubleshooting)
8. [Sample Conversations](#sample-conversations)
9. [Technical Architecture](#technical-architecture)

---

## Overview

**xstek Medical Transcription** is an AI-powered platform that transforms clinical conversations into accurate, structured documentation. The platform uses advanced speech recognition technology specifically trained for healthcare terminology.

### Three Ways to Transcribe:

| Method | Description | Use Case |
|--------|-------------|----------|
| **File Transcription** | Upload pre-recorded audio files | Recorded consultations, meetings |
| **Ambient AI** | Real-time conversation capture with fact extraction | Live consultations, clinical encounters |
| **Dictation** | Voice-controlled note-taking with templates | Quick notes, template filling |

---

## Understanding Transcription Modes

There are different ways to convert speech to text in this platform. Here's what each term means:

### Transcribe (Upload & Process)
Upload a pre-recorded audio file for batch processing. The file is sent to the server, processed, and returns the complete text.

**Example:**
```
Upload "consultation.mp3" → Wait 10 seconds → Get full text
API: POST /api/transcribe/upload
```

**When to Use:**
- Recorded consultations
- Meeting recordings
- Pre-recorded dictations

---

### Stream (Real-Time Audio)
Send audio in real-time via WebSocket. Text appears as you speak. Used by Ambient AI.

**Example:**
```
Speak → See words appear instantly → Continue talking
API: WebSocket /api/ambient/ws
```

**When to Use:**
- Live consultations
- Real-time note-taking
- Clinical fact extraction

---

### Dictation (Stateless Voice-to-Text)
Real-time transcription without storing session state. Supports voice commands for editing and navigation.

**Example:**
```
"Patient presents with cough period new paragraph" → Formatted text
API: WebSocket /api/dictation/ws
```

**When to Use:**
- Quick notes
- Template filling
- Voice commands

---

### Transcript (The Output)
The resulting text document from any transcription method. Contains the words spoken in the audio.

**Example:**
```
Input: Audio → Output: "Patient reports headache for 2 weeks..."
```

**When to Use:**
- Copy to EHR
- Generate documents
- Review/edit

---

### Key Difference Summary

| Term | What it is |
|------|------------|
| **Transcribe** | The action (upload file) |
| **Stream** | The method (real-time) |
| **Dictation** | The mode (voice commands) |
| **Transcript** | The result (the text) |

---

## SOAP Note Components

For accurate SOAP notes, the AI needs to extract specific clinical data. Here's what each component means:

### Vital Signs (Objective Measurements)

Measurable physiological parameters that indicate a patient's basic body functions.

**Includes:**
- **Blood Pressure** - e.g., "140/90 mmHg"
- **Heart Rate** - e.g., "88 beats per minute"
- **Temperature** - e.g., "37.5°C"
- **Respiratory Rate** - e.g., "16 breaths per minute"
- **Oxygen Saturation** - e.g., "98% on room air"
- **Weight/Height/BMI**

**How to Say It:**
> "Blood pressure is 140 over 90, heart rate 88, temperature 37.5"

---

### Medications (Current & Prescribed Drugs)

Drugs the patient is currently taking or being prescribed, including dosage and frequency.

**Include:**
- **Drug Name** - Generic or brand name
- **Dose** - e.g., "500 milligrams"
- **Frequency** - e.g., "twice daily"
- **Route** - oral, IV, topical, etc.
- **Duration** - "for 7 days"

**How to Say It:**
> "Patient takes Metformin 500 milligrams twice daily and Lisinopril 10 milligrams once daily"

---

### Allergies (Adverse Reactions)

Substances that cause allergic reactions in the patient, including the type of reaction.

**Include:**
- **Allergen** - Drug, food, or environmental
- **Reaction Type** - Rash, anaphylaxis, nausea
- **Severity** - Mild, moderate, severe
- **NKDA** - "No known drug allergies"

**How to Say It:**
> "Patient is allergic to Penicillin which causes a skin rash"  
> or  
> "No known drug allergies"

---

### Diagnoses (Clinical Conclusions)

Medical conditions identified based on symptoms, history, and examination findings.

**Include:**
- **Primary Diagnosis** - Main condition
- **Secondary Diagnoses** - Related conditions
- **Differential Diagnoses** - Possibilities to rule out
- **Chronic Conditions** - Existing history

**How to Say It:**
> "Assessment shows tension headache. Patient also has history of Type 2 Diabetes and Hypertension"

---

**💡 Pro Tip:** For complete SOAP notes, always mention all four components during the consultation. The AI extracts what it hears — if you don't say it, it won't appear in the note.

---

## Glossary of Terms

### Clinical Facts
**Category:** Core Concept

Structured pieces of clinical information extracted from conversations, such as medications, allergies, vital signs, diagnoses, symptoms, and medical history. Facts are validated by AI before being used for documentation.

**Example:** From "I take Metformin 500mg twice daily" → Medication fact: "Metformin 500mg BID"

**Note:** Facts are the foundation of accurate clinical documentation. The more explicit you are, the better the extraction.

---

### FactsR™
**Category:** Technology

The AI engine that powers clinical fact extraction. FactsR™ uses a recursive fact-first reasoning loop to identify, validate, and structure clinical knowledge in real-time. It processes audio approximately every 60 seconds.

**Example:** Listens to conversation → Identifies clinical entities → Validates accuracy → Outputs structured facts

**Note:** FactsR™ is designed to reduce "note bloat" and improve documentation accuracy compared to simple transcription.

---

### Ambient AI
**Category:** Feature

A real-time AI assistant that "listens" to doctor-patient conversations, transcribes speech, extracts clinical facts, and generates documentation automatically. No manual note-taking required during the encounter.

**Example:** Start recording → Have conversation → Stop → Get SOAP note automatically

**Note:** Best results with 2+ minutes of recording and clear clinical content.

---

### SOAP Note
**Category:** Document Type

**S**ubjective, **O**bjective, **A**ssessment, **P**lan - A standardized format for medical documentation.

| Section | Description |
|---------|-------------|
| **Subjective** | Patient's symptoms, complaints, history |
| **Objective** | Exam findings, vital signs, test results |
| **Assessment** | Diagnosis, clinical impression |
| **Plan** | Treatment, medications, follow-up |

**Example:**
```
S: Headache for 2 weeks
O: BP 140/90, HR 88
A: Tension headache
P: Ibuprofen PRN
```

**Note:** SOAP notes are the most common format for clinical documentation worldwide.

---

### Speaker Diarization
**Category:** Technology

The process of identifying and separating different speakers in an audio recording. Allows the system to distinguish between doctor and patient voices.

**Example:** 
```
Doctor: "What brings you in?" → Speaker 0
Patient: "Headache" → Speaker 1
```

**Note:** Works best with multi-channel audio (separate microphones). Single mic has limitations.

---

### Stateless Dictation
**Category:** Feature

A transcription mode where each session is independent with no persistent state. Audio is converted to text in real-time without creating an "interaction" or storing session history.

**Example:** Start dictating → See text appear → Copy note → Done (no session saved)

**Note:** Ideal for quick notes and voice commands. Does not extract clinical facts.

---

### Voice Commands
**Category:** Feature

Spoken instructions that control the application instead of being transcribed. Used for navigation, editing, and template insertion.

**Available Commands:**
- `"Insert GP consultation template"` - Loads structured note template
- `"Go to subjective section"` - Navigate to a section
- `"Go to objective section"`
- `"Go to assessment section"`
- `"Go to plan section"`
- `"Go to next section"` / `"Go to previous section"`
- `"New paragraph"`
- `"Delete last sentence"`
- `"Undo"`

**Note:** Commands must be spoken clearly and match defined phrases.

---

### NHS-Compliant Templates
**Category:** Document Type

Clinical note templates designed to meet UK National Health Service documentation standards, including PRSB (Professional Record Standards Body) standards, SNOMED CT coding, and dm+d medication references.

**Available Templates:**
1. GP Consultation
2. Clinical Note
3. Discharge Summary
4. A&E Triage
5. Progress Note
6. Physical Exam
7. Vital Signs
8. History & Physical
9. Emergency Note
10. Referral Letter

---

### WebSocket Streaming
**Category:** Technology

A protocol that enables real-time, bidirectional communication between the browser and server. Used for sending audio and receiving transcriptions without page refreshes.

**Flow:**
```
Browser sends audio chunks → Server sends back transcript updates → Both happen simultaneously
```

**Note:** Enables the real-time experience of seeing transcripts appear as you speak.

---

### Interaction
**Category:** Core Concept

A session container in the API that holds all data for a clinical encounter: transcripts, facts, and generated documents. Created when starting Ambient AI or uploading a file.

**Example:** Start session → Interaction ID created → All data stored under that ID

**Note:** Used for tracking, billing, and retrieving session data.

---

### Document Generation
**Category:** Feature

The process of creating structured clinical documents (like SOAP notes) from transcripts and extracted facts. The AI summarizes and organizes information into the requested format.

**Flow:**
```
Facts + Transcript → AI Processing → Formatted SOAP Note
```

**Note:** Quality depends on the facts extracted; empty sections mean missing clinical content.

---

## Features

### File Transcription (Async)

Upload pre-recorded audio files for batch processing. Ideal for transcribing recorded consultations, meetings, or dictated notes.

**Supported Formats:**
- WAV
- MP3
- M4A
- FLAC
- OGG
- WebM

**Capabilities:**
- Speaker diarization (identify different speakers)
- Multi-language detection
- Timestamp generation
- Document generation

---

### Ambient AI (Real-Time)

Capture live conversations between clinicians and patients. The AI extracts clinical facts in real-time and automatically generates documentation.

**How it Works:**
1. Start recording → Audio streams to AI
2. Real-time transcription appears as you speak
3. Clinical facts (medications, vitals, allergies) are extracted
4. Stop recording → SOAP note generated automatically

**What Gets Extracted:**
- Medications
- Allergies
- Vital Signs
- Diagnoses
- Symptoms
- Medical History

**Document Templates Available:**
- SOAP Note
- History & Physical
- Emergency Note
- Discharge Summary
- + 5 more templates

---

### Dictation (Voice-Controlled)

Stateless real-time speech-to-text with automatic punctuation, number formatting, and voice commands for hands-free note creation.

**NHS Templates:**
- GP Consultation
- Clinical Note
- Discharge Summary
- A&E Triage
- Progress Note
- + 5 more templates

**Auto Formatting:**
- Automatic punctuation
- Numbers as digits
- Date/time formatting
- Medical abbreviations

**Voice Commands:**
| Command | Action |
|---------|--------|
| "Insert GP consultation template" | Load structured template |
| "Go to subjective section" | Navigate to Subjective |
| "Go to objective section" | Navigate to Objective |
| "Go to assessment section" | Navigate to Assessment |
| "Go to plan section" | Navigate to Plan |
| "New paragraph" | Insert paragraph break |
| "Delete last sentence" | Remove previous sentence |
| "Undo" | Undo last action |

---

## Best Practices

### ✅ DO

- Use a quality external microphone
- Speak clearly at moderate pace
- Mention specific numbers: *"Blood pressure 140 over 90"*
- State medication doses: *"Metformin 500 milligrams twice daily"*
- Record for at least 2 minutes for best fact extraction
- Wait 2-3 seconds before stopping recording
- Use a quiet environment

### ❌ DON'T

- Use vague terms: *"BP is normal"* (say the numbers instead)
- Talk over each other
- Stop recording immediately after speaking
- Record in noisy environments
- Use abbreviations verbally: say "blood pressure" not "B-P"
- Rush through clinical information
- Expect facts from very short sessions (<1 min)

### For Complete SOAP Notes, Always Mention:

| Component | Example |
|-----------|---------|
| **Vital Signs** | "Blood pressure 140 over 90, heart rate 88" |
| **Medications** | "Metformin 500 milligrams twice daily" |
| **Allergies** | "Allergic to Penicillin, causes rash" |
| **Diagnoses** | "Assessment: tension headache" |

---

## Troubleshooting

| Issue | Cause | Solution |
|-------|-------|----------|
| No clinical facts appearing | Recording too short or no clinical content spoken | Record 2+ minutes; mention medications, vitals, diagnoses explicitly |
| Empty Objective section in SOAP | No vital signs or exam findings mentioned | Say "Blood pressure 140 over 90, heart rate 88" explicitly |
| Medication name misheard | Speech recognition limitation | Speak medication names slowly and clearly; spell if needed |
| Speaker labels wrong | Single microphone limitation | Use separate microphones for doctor/patient in production |
| Voice commands not working | Phrase not matching defined commands | Use exact phrases: "Insert GP consultation template" |
| Transcription stops working | WebSocket connection lost | Stop and restart the session; check internet connection |
| Document generation fails | Not enough content in transcript | Ensure at least 2 minutes of clinical conversation |

---

## Sample Conversations

### Sample 1: Basic Consultation (For Testing)

Use this sample for testing Ambient AI:

```
Doctor: "Good morning. What brings you in today?"

Patient: "I've been having severe headaches for the past two weeks."

Doctor: "Any other symptoms?"

Patient: "Yes, some nausea in the mornings and my vision gets blurry sometimes."

Doctor: "What's your medical history?"

Patient: "I have Type 2 Diabetes. I take Metformin 500 milligrams twice daily."

Doctor: "Any allergies?"

Patient: "I'm allergic to Penicillin. It causes a rash."

Doctor: "Let me check your vitals. Blood pressure 140 over 90, heart rate 88 beats per minute."

Doctor: "Based on my assessment, you have tension headaches. I'm prescribing Ibuprofen 400 milligrams as needed."
```

### Sample 2: Dictation Demo Script

Use this script to test the Dictation feature:

**Step 1:** Start Dictation

**Step 2:** Say: *"Insert GP consultation template"*

**Step 3:** Dictate into each section:

| Section | Say This |
|---------|----------|
| **Presenting Complaint** | "55-year-old male presents with chest pain for 3 days, located on the left side" |
| **Navigate** | "Go to objective section" |
| **Clinical Findings** | "Blood pressure 140 over 90, heart rate 72 regular, oxygen saturation 98 percent" |
| **Navigate** | "Go to assessment section" |
| **Clinical Impression** | "Stable angina, differential diagnosis includes musculoskeletal chest pain" |
| **Navigate** | "Go to plan section" |
| **Management Plan** | "Refer to cardiology for exercise stress test. Continue current medications. Follow up in 2 weeks" |

**Step 4:** Click "Copy Note" to get formatted output

---

### Sample 3: Complex Consultation (Comprehensive)

```
Doctor: "Hello Mrs. Johnson. I see you're here about your diabetes follow-up. How have you been feeling?"

Patient: "Not great, doctor. I've been having increased thirst and I need to use the bathroom more often."

Doctor: "I see. And how's your blood sugar control been?"

Patient: "My home readings have been around 180 to 200 in the mornings."

Doctor: "That's elevated. Let me take your vitals. Blood pressure is 145 over 95, heart rate 76 beats per minute, temperature 36.8 degrees, oxygen saturation 99 percent on room air."

Patient: "Is that high blood pressure?"

Doctor: "It's slightly elevated. Can you remind me what medications you're taking?"

Patient: "I take Metformin 1000 milligrams twice daily, Lisinopril 20 milligrams once daily, and Atorvastatin 40 milligrams at night."

Doctor: "Any allergies?"

Patient: "Yes, I'm allergic to Sulfa drugs. They cause a severe rash and swelling."

Doctor: "Based on my examination and your symptoms, your diabetes is not well controlled. Your HbA1c from last week was 8.5 percent. I'm going to add Glipizide 5 milligrams once daily before breakfast. We'll also increase your Lisinopril to 40 milligrams for better blood pressure control."

Patient: "Should I be worried?"

Doctor: "With these medication adjustments and lifestyle modifications, we should see improvement. I want you to monitor your blood sugar twice daily, reduce carbohydrate intake, and walk for 30 minutes daily. Let's follow up in 6 weeks to recheck your blood pressure and HbA1c."
```

---

## Technical Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                        FRONTEND                             │
│  Vue 3 + Vuetify 3 + TypeScript                            │
│  - WebSocket Client                                         │
│  - MediaRecorder API                                        │
│  - Real-time UI Updates                                     │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                        BACKEND                              │
│  Go + Fiber Framework                                       │
│  - REST API Endpoints                                       │
│  - WebSocket Proxy                                          │
│  - OAuth2 Authentication                                    │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                      CORTI AI ENGINE                        │
│  FactsR™ Technology                                         │
│  - Speech-to-Text                                           │
│  - Clinical Fact Extraction                                 │
│  - Document Generation                                      │
└─────────────────────────────────────────────────────────────┘
```

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/transcribe/upload` | POST | Upload audio file for async transcription |
| `/api/ambient/ws` | WebSocket | Real-time ambient AI streaming |
| `/api/dictation/ws` | WebSocket | Real-time dictation streaming |
| `/api/health` | GET | Health check endpoint |

### Data Flow

1. **File Transcription:**
   ```
   Upload Audio → Backend → Corti API → Process → Return Transcript
   ```

2. **Ambient AI:**
   ```
   Record Audio → WebSocket → Stream to Corti → Real-time Transcript + Facts → Generate Document
   ```

3. **Dictation:**
   ```
   Record Audio → WebSocket → Stream to Corti → Real-time Transcript → Voice Commands → Formatted Note
   ```

---

## Contact & Support

For technical support or questions about xstek Medical Transcription:

- **Documentation:** [http://localhost:5173/docs](http://localhost:5173/docs)
- **Demo Platform:** [http://localhost:5173](http://localhost:5173)

---

*This documentation is generated for xstek Medical Transcription Demo. All clinical examples are for demonstration purposes only.*



