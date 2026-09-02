# xstek Medical Transcription - Internal Demo Guide

**Date:** January 2026  
**Presenter:** [Your Name]  
**Product:** xstek Medical Transcription Platform  
**Status:** Development Demo

## 🌍 Portable live-demo setup

The AI Assistance panel uses the hosted OpenAI-compatible provider profile in
`backend/.env.example`, so the demo does not depend on the presenter's laptop,
local Ollama, or a private GPU VM.

```bash
cd medical-transcription/backend
cp .env.example .env
```

Set `AI_API_KEY` in that ignored `.env` (or inject it as a process secret). Keep
`AI_MODEL=deepseek-v4-flash`; select the nearest `AI_BASE_URL` from the regional
table in [SETUP.md](SETUP.md#6-step-3--backend-configuration) if the instructor
is outside mainland China. Never commit the real key.

---

## 📋 Executive Summary

xstek Medical Transcription is a production-grade AI-powered platform that transforms clinical conversations into accurate, structured documentation. The platform integrates advanced speech recognition with medical-specific AI to deliver:

- **Real-time transcription** with clinical fact extraction
- **Automated clinical document generation** (SOAP notes, discharge summaries, etc.)
- **Voice-controlled dictation** with NHS-compliant templates
- **Multi-language support** for global healthcare settings

---

## 🎯 Demo Objectives

By the end of this demo, colleagues will understand:

1. ✅ What features are fully implemented
2. ⚠️ What features need improvement
3. 🚀 What the roadmap looks like
4. 💡 How the platform adds value to clinical workflows

---

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    xstek Medical Transcription                   │
├─────────────────────────────────────────────────────────────────┤
│  FRONTEND (Vue 3 + Vuetify)                                     │
│  ├── File Transcription Page                                    │
│  ├── Ambient AI Session Page                                    │
│  └── Dictation Page                                             │
├─────────────────────────────────────────────────────────────────┤
│  BACKEND (Go + Fiber)                                           │
│  ├── REST API Handlers                                          │
│  ├── WebSocket Proxy (Ambient + Dictation)                      │
│  ├── OAuth2 Token Manager                                       │
│  └── Audio Processing                                           │
├─────────────────────────────────────────────────────────────────┤
│  AI ENGINE (Corti API Integration)                              │
│  ├── Speech-to-Text (FactsR™)                                   │
│  ├── Clinical Fact Extraction                                   │
│  └── Document Generation                                        │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🎬 Demo Script

### Demo 1: File Transcription (Async)

**Route:** `http://localhost:5173/async-transcription`

**Duration:** ~3 minutes

#### Steps:
1. Navigate to **File Transcription** from the navigation bar
2. Drag and drop a sample audio file (WAV, MP3, M4A, etc.)
3. Select language (English US)
4. Click **Upload & Transcribe**
5. Wait for processing (~30-60 seconds for a 2-minute file)
6. Show the structured transcript with:
   - Speaker diarization (Doctor vs Patient)
   - Timestamps
   - Formatted text
7. Select a template (e.g., SOAP Note)
8. Click **Generate Document**
9. Show the generated clinical document

#### Talking Points:
> "This feature allows clinicians to upload pre-recorded consultations and automatically generate clinical documentation. The AI identifies different speakers and extracts key clinical information."

#### What's Working:
- ✅ Audio file upload (multiple formats)
- ✅ Transcription with speaker diarization
- ✅ Multiple document templates
- ✅ Document generation

#### What Needs Improvement:
- ⚠️ Large file upload progress indicator
- ⚠️ Batch file processing

---

### Demo 2: Ambient AI (Real-Time)

**Route:** `http://localhost:5173/ambient-session`

**Duration:** ~5 minutes

#### Preparation:
- Use a quiet room
- Position microphone between speakers
- Have a sample conversation ready

#### Steps:
1. Navigate to **Ambient AI** from the navigation bar
2. Select language and microphone
3. Click **Start Recording**
4. Perform a sample doctor-patient conversation:

**Sample Script:**
> **Doctor:** "Good morning. What brings you in today?"
> 
> **Patient:** "I've been having severe headaches for two weeks."
> 
> **Doctor:** "Any other symptoms?"
> 
> **Patient:** "Yes, some nausea and blurry vision."
> 
> **Doctor:** "What's your medical history?"
> 
> **Patient:** "I have Type 2 Diabetes. I take Metformin 500 milligrams twice daily."
> 
> **Doctor:** "Any allergies?"
> 
> **Patient:** "I'm allergic to Penicillin."
> 
> **Doctor:** "Let me check your vitals. Blood pressure 140 over 90, heart rate 88."
> 
> **Doctor:** "Based on my assessment, you have tension headaches. I'm prescribing Ibuprofen 400 milligrams as needed."

5. Observe:
   - Real-time transcript appearing
   - Clinical facts being extracted (medications, allergies, vitals)
   - Facts status indicator
6. Click **Stop Recording**
7. Show automatic SOAP note generation
8. Demonstrate manual template selection

#### Talking Points:
> "Ambient AI listens to the entire consultation in real-time, extracts clinical facts like medications, allergies, and vital signs, and automatically generates documentation. This eliminates manual note-taking during patient encounters."

#### What's Working:
- ✅ Real-time transcription
- ✅ WebSocket streaming
- ✅ Clinical fact extraction (FactsR™)
- ✅ Automatic SOAP note generation
- ✅ Facts status indicator
- ✅ Multiple template support

#### What Needs Improvement:
- ⚠️ Facts extraction consistency (sometimes delayed)
- ⚠️ Speaker diarization accuracy (single mic limitation)
- ⚠️ Waiting period before document generation

---

### Demo 3: Real-Time Dictation

**Route:** `http://localhost:5173/dictation`

**Duration:** ~4 minutes

#### Steps:
1. Navigate to **Dictation** from the navigation bar
2. Select language and microphone
3. Click **Start Dictation**
4. Demonstrate basic dictation:
   > "The patient is a 55-year-old female presenting with chest pain. Blood pressure 130 over 85."

5. Show automatic punctuation and number formatting
6. Demonstrate voice commands:
   - Say: **"Insert GP consultation template"**
   - Say: **"Go to presenting complaint section"**
   - Dictate content for that section
   - Say: **"Go to clinical findings section"**
   - Say: **"New paragraph"**
7. Show the structured NHS clinical note
8. Click **Copy NHS Note** to show formatted output
9. Click **Stop Dictation**

#### Voice Commands to Demo:
| Say This | What Happens |
|----------|--------------|
| "Insert GP consultation template" | Inserts NHS template |
| "Go to presenting complaint" | Moves cursor to that section |
| "Go to clinical findings" | Moves to next section |
| "New line" | Adds line break |
| "New paragraph" | Adds paragraph break |
| "Delete last sentence" | Removes last sentence |
| "Undo" | Undoes last action |

#### Talking Points:
> "The Dictation feature provides stateless, low-latency transcription with voice commands. Clinicians can navigate through structured templates using voice alone, making documentation hands-free."

#### What's Working:
- ✅ Real-time transcription
- ✅ Automatic punctuation
- ✅ Number formatting (140 over 90 → 140/90)
- ✅ Voice command detection
- ✅ NHS-compliant templates (10 templates)
- ✅ Section navigation
- ✅ Copy formatted output

#### What Needs Improvement:
- ⚠️ Medical term accuracy ("lisinopril" sometimes misheard)
- ⚠️ Command phrases need to be spoken clearly
- ⚠️ Template editing (manual typing in sections)

---

## 📊 Feature Comparison Matrix

| Feature | File Transcription | Ambient AI | Dictation |
|---------|-------------------|------------|-----------|
| Audio Input | Pre-recorded file | Live microphone | Live microphone |
| Processing | Async (batch) | Real-time stream | Real-time stream |
| Speaker Diarization | ✅ Yes | ✅ Yes | ❌ No |
| Fact Extraction | Via transcript | ✅ Real-time | ❌ No |
| Document Generation | ✅ Yes | ✅ Yes | Templates only |
| Voice Commands | ❌ No | ❌ No | ✅ Yes |
| NHS Templates | ✅ Yes | ✅ Yes | ✅ Yes |
| Session State | Stateful | Stateful | Stateless |

---

## 📈 Technical Achievements

### Backend (Go + Fiber)
- OAuth2 client credentials authentication
- Automatic token refresh (5-minute expiry handling)
- WebSocket proxy for real-time streaming
- Concurrent session management
- Error handling and recovery

### Frontend (Vue 3 + Vuetify)
- Modern responsive UI with dark/light themes
- Real-time WebSocket communication
- Audio capture and streaming
- Dynamic component rendering
- State management with composables

### API Integration
- REST APIs for file upload and document generation
- WebSocket for Ambient AI streaming
- WebSocket for Dictation streaming
- Proper error handling and retry logic

---

## ⚠️ Known Limitations & Areas for Improvement

### 1. Fact Extraction Consistency
**Issue:** Clinical facts sometimes don't appear or are delayed  
**Cause:** AI processes facts in ~60-second intervals; short sessions may miss facts  
**Workaround:** Record for 2+ minutes; speak clinical content clearly  
**Future Fix:** Add buffering and fact confirmation before document generation

### 2. Speaker Diarization
**Issue:** Single microphone cannot reliably distinguish speakers  
**Cause:** Current setup uses mono audio from one mic  
**Recommended:** Multi-channel setup (separate mics for doctor/patient)  
**Vendor Update:** Improved diarization expected Q1 2026

### 3. Medical Term Recognition
**Issue:** Some medications misheard (e.g., "lisinopril" → "at least in April")  
**Cause:** Speech recognition model limitations  
**Workaround:** Speak medication names slowly and clearly  
**Future Fix:** Await vendor model improvements

### 4. Document Objective Section
**Issue:** Sometimes empty in SOAP notes  
**Cause:** Requires explicit mention of vitals and exam findings  
**Workaround:** Always verbalize vital signs with numbers

---

## 🗺️ Roadmap

### Completed ✅
- [x] File transcription with diarization
- [x] Real-time ambient AI streaming
- [x] Clinical fact extraction
- [x] Document generation (9 templates)
- [x] Real-time dictation
- [x] Voice commands (navigation, templates, editing)
- [x] NHS-compliant templates (10 templates)
- [x] Multi-language support
- [x] Dark/light theme

### In Progress 🔄
- [ ] Improve fact extraction reliability
- [ ] Add fact confirmation before document generation
- [ ] Better error handling and user feedback

### Planned 📋
- [ ] Multi-channel audio support (dual microphone)
- [ ] EHR/EMR integration
- [ ] Custom template builder
- [ ] Medical coding (ICD-10, SNOMED CT)
- [ ] Batch processing for multiple files
- [ ] User authentication and session history
- [ ] Mobile-responsive improvements

---

## 🔧 Technical Setup (For Reference)

### Running the Demo

**Backend:**
```bash
cd backend
go run cmd/server/main.go
# Starts on http://localhost:8080
```

**Frontend:**
```bash
cd frontend
npm run dev
# Starts on http://localhost:5173
```

### Environment Variables
```env
CLIENT_ID=your_api_client_id
CLIENT_SECRET=your_api_client_secret
TENANT_NAME=your_tenant_name
ENVIRONMENT=eu  # or us
```

---

## 🎤 Demo Tips

### Before the Demo:
- [ ] Test audio on the demo machine
- [ ] Ensure quiet environment
- [ ] Have sample audio file ready
- [ ] Practice the sample conversation
- [ ] Check internet connectivity

### During the Demo:
- Speak clearly and at moderate pace
- Pause between speakers in conversation
- Mention specific clinical data (numbers, medication names)
- Show both working features and known limitations
- Be transparent about what's still in development

### Handling Questions:
| Question | Answer |
|----------|--------|
| "Why are facts sometimes missing?" | AI processes in 60-second intervals; need 2+ minutes of recording with specific clinical content |
| "Can it integrate with our EHR?" | Planned for future; requires custom integration work |
| "How accurate is it?" | Very high for clear audio; some limitations with medical terms |
| "Is it HIPAA/GDPR compliant?" | Data flows through Corti's certified infrastructure; needs review for production |

---

## 📞 Q&A Preparation

**Expected Questions:**

1. **"What's the accuracy rate?"**
   > Speech recognition accuracy is typically 95%+ for clear audio. Medical term accuracy varies; ongoing improvements from the AI vendor.

2. **"How does it handle accents?"**
   > Supports multiple languages and accents. Works best with clear pronunciation.

3. **"What about data security?"**
   > Audio is streamed securely (WSS), tokens are server-side only, no credentials exposed to frontend.

4. **"Can we customize templates?"**
   > Currently using predefined templates. Custom template builder is on the roadmap.

5. **"What's the cost model?"**
   > Based on API usage (per minute of audio processed). Need to finalize pricing with vendor.

---

## 📝 Feedback Collection

After the demo, collect feedback on:

1. Which features are most valuable for clinical workflows?
2. What additional features would be helpful?
3. Any concerns about accuracy or usability?
4. Integration requirements with existing systems?

---

## 📎 Appendix

### Sample Audio Files for Demo
If you don't have sample files, you can:
1. Record a sample using the Ambient AI feature
2. Use the Dictation feature to create test content
3. Download sample medical audio from test repositories

### Supported Audio Formats
- WAV (recommended)
- MP3
- M4A
- FLAC
- OGG
- WebM

### Document Templates Available
1. SOAP Note (corti-soap)
2. Brief Clinical Note
3. Outpatient Visit Note
4. Emergency Note
5. Nursing Note
6. Emergency Response Note
7. History & Physical
8. Referral Letter
9. Patient Summary

### NHS-Compliant Dictation Templates
1. GP Consultation
2. Clinical Note
3. Discharge Summary
4. A&E Triage
5. Progress Note
6. Physical Exam
7. Vital Signs
8. Medication List
9. Allergy List
10. Review of Systems

---

**Demo Duration:** ~15-20 minutes  
**Q&A Time:** ~10 minutes  
**Total Session:** ~30 minutes

---

*Prepared by xstek Development Team*
