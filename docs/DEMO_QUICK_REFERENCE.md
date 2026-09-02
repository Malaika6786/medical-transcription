# xstek Medical Transcription - Quick Reference Card

## 🔐 Before the demo

1. On the instructor's machine, copy `backend/.env.example` to `backend/.env`.
2. Set the supplied provider key in `AI_API_KEY`; keep `AI_MODEL=deepseek-v4-flash`.
3. Use the nearest regional `AI_BASE_URL` from [SETUP.md](SETUP.md#6-step-3--backend-configuration).
4. Do not put real keys or patient-identifying transcripts in GitHub.

## 🌐 URLs
| Page | URL |
|------|-----|
| Home | http://localhost:5173 |
| File Transcription | http://localhost:5173/async-transcription |
| Ambient AI | http://localhost:5173/ambient-session |
| Dictation | http://localhost:5173/dictation |

---

## 🎬 Demo 1: File Transcription
```
1. Go to File Transcription
2. Drag & drop audio file
3. Select language → Upload
4. Wait for transcript
5. Select template → Generate Document
```

---

## 🎙️ Demo 2: Ambient AI

### Sample Conversation:
```
Doctor: "What brings you in today?"
Patient: "Severe headaches for two weeks."
Doctor: "What medications do you take?"
Patient: "Metformin 500 milligrams twice daily."
Doctor: "Any allergies?"
Patient: "Allergic to Penicillin."
Doctor: "Blood pressure 140 over 90, heart rate 88."
Doctor: "Diagnosis: tension headaches. Prescribing Ibuprofen 400 mg."
```

### Key Points to Show:
- ✅ Real-time transcript
- ✅ Clinical facts appearing
- ✅ Auto-generated SOAP note

---

## 🎤 Demo 3: Dictation

### Voice Commands to Demo:
| Say This | Action |
|----------|--------|
| "Insert GP consultation template" | Inserts NHS template |
| "Go to presenting complaint" | Navigate section |
| "Go to clinical findings" | Navigate section |
| "New paragraph" | Add paragraph |
| "Delete last sentence" | Delete text |
| "Undo" | Undo action |

### Sample Dictation:
```
"The patient is a 55-year-old female presenting with 
chest pain. Blood pressure 130 over 85. Heart rate 
72 beats per minute."
```

---

## ✅ What Works

| Feature | Status |
|---------|--------|
| File upload & transcription | ✅ |
| Speaker diarization | ✅ |
| Real-time streaming | ✅ |
| Clinical fact extraction | ✅ |
| Document generation | ✅ |
| Voice commands | ✅ |
| NHS templates | ✅ |
| Multi-language | ✅ |

---

## ⚠️ Known Limitations

| Issue | Workaround |
|-------|------------|
| Facts delayed | Record 2+ minutes |
| Speaker mix-up | Multi-mic planned |
| Med terms misheard | Speak slowly |
| Empty Objective | Say vitals explicitly |

---

## 💡 Demo Tips

1. **Quiet room** - minimize background noise
2. **Clear speech** - moderate pace
3. **Be specific** - "140 over 90" not "elevated"
4. **Wait 2 sec** - before stopping recording
5. **Show limitations** - be transparent

---

## 🔢 Key Numbers

| Metric | Value |
|--------|-------|
| Supported languages | 15+ |
| Document templates | 9 |
| NHS templates | 10 |
| Voice commands | 11 |
| Audio formats | 6 |

---

**Demo Time: ~20 min | Q&A: ~10 min**
