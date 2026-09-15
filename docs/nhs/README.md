# NHS / SystmOne Integration Module (`internal/nhs`, `internal/cryptofield`, `internal/pdfgen`)

This is the engineering half of closing the gaps identified in
[`SYSTMONE_INTEGRATION_REPORT.md`](../../../SYSTMONE_INTEGRATION_REPORT.md).
It is real, compiled, unit-tested Go code — but it **cannot reach a real
NHS system** without organisational and credential prerequisites this
codebase alone cannot satisfy (see
[`docs/regulatory/PREREQUISITES.md`](../regulatory/PREREQUISITES.md)). This
document is the honest map of what's live today vs. what's still blocked on
those prerequisites.

## What's real and working today

| Piece | Location | Status |
|---|---|---|
| NHS number validation (modulus-11 check digit) | `internal/nhs/nhsnumber.go` | ✅ Fully working, unit-tested (`nhsnumber_test.go`) |
| Structured patient identity model | `internal/nhs/patient.go`, `internal/pgstore/patient_store.go` | ✅ Full CRUD, encrypted at rest |
| Field-level encryption (AES-256-GCM) | `internal/cryptofield/` | ✅ Fully working, unit-tested |
| Clinical audit trail | `internal/pgstore/audit_store.go` | ✅ Append-only, wired into every patient/NHS action |
| PDF generation for the clinical letter | `internal/pdfgen/` | ✅ Dependency-free, unit-tested, produces valid PDF |
| FHIR Bundle construction for GP Connect: Send Document | `internal/nhs/fhir.go` | ⚠️ Structurally correct FHIR JSON; the exact coding/profile constraints need confirming against NHS England's live Send Document profile |
| PDS client (trace by NHS number, demographic search) | `internal/nhs/pds.go` | ⚠️ Defaults to NHS's public sandbox (synthetic patients, no real credentials needed) — code path is real, the *live* endpoint/auth needs verification |
| MESH client (message transport) | `internal/nhs/mesh.go` | ⚠️ Implements the publicly documented request/auth shape — **cannot send anywhere without a real MESH mailbox ID + password + shared key** |
| NHS CIS2 (OIDC) scaffold | `internal/nhs/cis2.go` | ⚠️ Standard, generically-correct OIDC Authorization Code + PKCE — not wired into any route; only needed if a future NHS API call requires clinician-identity-bound tokens (GP Connect: Send Document and MESH do not) |
| Data-residency boot guard | `utils.Config.ValidateDataResidency` | ✅ Refuses to start with a non-UK/EU Corti region or AI endpoint, unless explicitly overridden |
| Production containers/IaC | `backend/Dockerfile`, `frontend/Dockerfile`, `docker-compose.prod.yml` | ✅ Working multi-stage builds + TLS via Caddy |

## What still needs NHS-issued credentials/registration (not solvable in code)

- **MESH mailbox** (ID, password, shared key) — issued when your
  organisation registers as a supplier/sender.
- **PDS production access** — the sandbox needs no credentials; real
  patient lookups do, and the exact current auth scheme (API key vs. CIS2
  bearer token) needs confirming against NHS Digital's live PDS FHIR API
  catalogue entry at integration time.
- **ODS code** — your organisation's NHS-wide identifier, assigned on
  registration, used as the FHIR message author (`NHS_ORG_ODS_CODE`).
- **GP Connect: Send Document profile conformance** — the Composition
  coding values in `fhir.go` are placeholders; NHS England's actual
  published profile (or a conformance testing session with them) will say
  exactly what's mandatory.
- **A receiving practice's MESH mailbox ID** for any real send — you need a
  specific GP practice/PCN willing to pilot with you.

See `docs/regulatory/PREREQUISITES.md` for the organisational side of this
(legal entity, Clinical Safety Officer, pilot partner) — none of that is
something code can provide.

## How to try what's here right now, safely

1. `FIELD_ENCRYPTION_KEY=$(openssl rand -hex 32)` in `backend/.env` — this
   turns on the `/api/patients` and `/api/nhs/*` routes.
2. Create a patient via `POST /api/patients` with a real-format NHS number
   that passes the modulus-11 check (e.g. `9434765919`, a well-known valid
   test number — not a real patient).
3. `POST /api/nhs/pds/trace` against the default sandbox `PDS_BASE_URL` —
   this exercises the whole PDS client code path against NHS's own public
   synthetic-patient sandbox, no real credentials required. Confirmed
   working live (2026-09-08) against NHS numbers `9000000009` ("Jane
   Smith") and `9000000025` ("Janet Smythe") — `9000000017` and most other
   checksum-valid numbers 404, since the sandbox only recognises its own
   specific pre-loaded synthetic patients, not every valid-format number.
4. `POST /api/patients/:id/verify-pds` to see the full verify-and-stamp
   flow.
5. `POST /api/nhs/sessions/:id/send-to-gp` will correctly refuse with "MESH
   is not configured" until real `MESH_MAILBOX_ID`/`MESH_MAILBOX_PASSWORD`/
   `MESH_SHARED_KEY` values exist — that refusal is the intended behaviour,
   not a bug.

## Data residency

`DATA_RESIDENCY_UK_ONLY` (default `true`; the shipped demo `.env.example`
explicitly sets it `false` for its own known-synthetic-data posture — see
that file's comments) makes the server refuse to boot if `CORTI_ENVIRONMENT`
or `AI_BASE_URL` fall outside a UK/EU/local allowlist. Flip it on, and point
those two settings at UK/EU-hosted or self-hosted endpoints, before this
deployment ever touches a real patient's data.
