# Medical Device Classification Worksheet

**Status: NOT DETERMINED.** This worksheet structures the question for a
qualified regulatory reviewer (ideally in consultation with the MHRA or a
regulatory consultant) — it does not answer it. Do not treat any
conclusion drawn from this document alone as a real classification
decision.

## Why this matters

Per NHS England/MHRA guidance on AI-enabled ambient scribing (2026), a tool
that only produces a verbatim transcript or a draft letter for clinician
review is **not** a medical device. A tool that provides "generated
insights" suggesting diagnosis, treatment, or follow-up **is** likely to be
one, requiring UKCA/CE marking as Software as a Medical Device (SaMD)
before any clinical use.

## This project's actual output, mapped against that line

The AI Assistance module (`internal/ai/`, `POST /api/ai/summarize`)
produces a 15-field `ExtractionResult`. Each field needs its own
determination — they are not all equivalent:

| Field | Likely closer to... | Reasoning worth checking |
|---|---|---|
| `summary` | Non-device | A condensed version of what was said, not a new clinical judgement |
| `chiefComplaint` | Non-device | Restates what the patient said |
| `history` | Non-device | Restates what was said |
| `allergies` | Non-device | Restates what was said |
| `medications` | Non-device | Restates what was said |
| `symptoms` | Non-device | Restates what was said |
| **`diagnosis`** | **Needs review** | If this is "what the clinician said aloud, transcribed" it's likely non-device; if the model is inferring a diagnosis not explicitly stated, that's a generated insight |
| **`differentialDiagnosis`** | **Needs review** | Generating a differential the clinician didn't state out loud is the clearest example MHRA's guidance flags as device-like |
| **`treatment`** | **Needs review** | Same reasoning as differential diagnosis |
| `labTests` | Non-device (if restating what was ordered) / needs review (if suggesting new tests) | Depends on prompt behaviour — check `internal/ai/prompts.go` |
| `procedures` | Non-device | Restates what was said |
| `followUp` | Needs review | Depends on whether it's restating a stated plan or generating a new one |
| `riskFactors` | Needs review | Depends on whether it's identifying risk factors not explicitly discussed |
| `medicalTerms` | Non-device | A glossary/definition aid |
| `actionItems` | Non-device | Restates what was said |

## What was actually found in `internal/ai/prompts.go`

The current extraction prompt instructs the model, for the three fields
flagged above, to extract:

- `diagnosis`: "each diagnosis or clinical impression **the doctor states**"
- `differentialDiagnosis`: "alternative or suspected diagnoses **the doctor
  is still considering**"
- `treatment`: "each treatment or intervention **advised or performed**"

This wording is restate-oriented (transcribing what the clinician already
said), not generate-oriented (the model forming its own independent
diagnosis) — which is a meaningfully better starting position than a
prompt that asked the model to diagnose on its own. This is still not a
formal determination: whether an LLM can be relied upon to *only* restate
and never leak its own inference into these fields in practice (rather
than in the prompt's stated intent) is exactly the kind of question a real
regulatory/clinical safety review needs to test empirically — e.g. by
red-teaming the prompt with transcripts that don't contain an explicit
diagnosis, and checking whether the model still fills the field in anyway.

## What to actually do with this

1. Red-team the current prompt (see above) with a regulatory/clinical
   reviewer: feed it transcripts where the clinician never states a
   diagnosis, and check whether `diagnosis`/`differentialDiagnosis` come
   back empty (expected) or the model infers one anyway (would push this
   toward device classification despite the prompt's wording).
2. Get a written determination — even an informal one — from someone with
   MHRA/regulatory expertise, citing the specific prompt behaviour.
3. If any field is determined to be device-like, either (a) remove/disable
   it, (b) change the prompt so it only restates rather than infers, or (c)
   proceed down the full UKCA-marking SaMD pathway for those fields
   specifically — a materially larger undertaking than anything else in
   this report.
4. Whatever the outcome, register on the NHS England AVT Supplier Registry
   regardless — the ambient/dictation capture itself, independent of the
   extraction fields, is squarely what that registry governs.

## Needs verification

- The exact MHRA guidance wording and worked examples (only summarized here
  from public commentary, not the primary MHRA guidance document itself).
- Whether "restates vs. infers" is actually the right test MHRA applies, or
  a simplification — confirm with a qualified reviewer.
