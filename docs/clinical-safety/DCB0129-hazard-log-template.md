# DCB0129 Hazard Log — Template

**This is a template, not a completed hazard log.** A real hazard log must
be created, owned, and maintained by a named Clinical Safety Officer (CSO)
— see `docs/regulatory/PREREQUISITES.md`. Filling in rows below without a
qualified CSO's involvement does not constitute DCB0129 compliance.

## Instructions

For every distinct way this product's output could plausibly mislead or
harm a patient, add one row. Start from the code paths that touch clinical
content — this list is a starting point drawn from the actual codebase, not
an exhaustive analysis:

| ID | Hazard | Cause | Effect | Initial Risk (Likelihood × Severity) | Existing Controls | Mitigation | Residual Risk | Owner | Status |
|---|---|---|---|---|---|---|---|---|---|
| H-01 | Transcript omits or mishears clinically significant content | Speech recognition error (Corti STT), background noise, accent/dialect mismatch | Clinician's record is missing or contains incorrect information | TBD | Clinician reviews transcript before saving (workflow assumption — verify this is actually enforced in the UI, not just assumed) | TBD | TBD | CSO | Open |
| H-02 | AI extraction (`diagnosis`/`differentialDiagnosis`/`treatment`) states something the clinician did not actually say | LLM hallucination despite restate-oriented prompt (see `docs/regulatory/medical-device-classification-worksheet.md`) | Clinician or downstream reader (e.g. a GP receiving a sent document) relies on an incorrect clinical statement | TBD | None automated today — no confidence scoring or source-highlighting was found in `internal/ai/` | Add source-span highlighting so every extracted claim can be traced to transcript text; require clinician sign-off before a document is sent to a GP practice | TBD | CSO | Open |
| H-03 | Document sent to the wrong patient's GP record | PDS matching error, clinician selects wrong patient in UI, or a stale/incorrect NHS number on the patient record | Serious safety incident — clinical information appears in the wrong patient's record | TBD | `internal/nhs`'s Send Document flow requires PDS verification (`pds_verified_at`) before sending — see `internal/handlers/nhs_handler.go` | Confirm the UI surfaces the PDS-confirmed name/DOB for a final clinician check immediately before send, not just at verification time | TBD | CSO | Open |
| H-04 | Session/document data breach exposes patient identity + clinical content | Application vulnerability, credential compromise, infrastructure misconfiguration | Confidentiality breach, regulatory/reputational harm, potential patient distress | TBD | Field-level encryption for patient PII (`internal/cryptofield`), audit trail (`internal/pgstore/audit_store.go`), RBAC | Penetration test before go-live; confirm TLS termination and infrastructure hardening in the actual deployment (not just this repo's Dockerfiles) | TBD | CSO + Security lead | Open |
| H-05 | Field encryption key (`FIELD_ENCRYPTION_KEY`) is lost | Operational error, inadequate secrets management | All existing patient records become permanently undecryptable | TBD | None — there is no key-rotation/backup scheme in `internal/cryptofield` today (documented as a known limitation in that package) | Establish a secrets-management and key-backup procedure before any real patient data is stored | TBD | Ops lead | Open |

## Notes

- "Initial Risk"/"Residual Risk" should use whatever likelihood/severity
  matrix your organisation's clinical risk management process defines
  (DCB0129 does not mandate a specific matrix) — left blank here
  deliberately rather than inventing one.
- This table should grow as the CSO reviews the actual running system, not
  just the code — several of the "Existing Controls" cells above are
  claims about what the code appears to do and need independent
  verification against the live behaviour.
