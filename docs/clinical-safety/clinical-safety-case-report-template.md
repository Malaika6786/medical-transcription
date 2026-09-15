# Clinical Safety Case Report (CSCR) — Template

**This is a template, not a completed safety case.** Per DCB0129, this
report must be authored/owned by a named, GMC/NMC-registered Clinical
Safety Officer — see `docs/regulatory/PREREQUISITES.md`.

## 1. Scope

_Which version/deployment of the system does this safety case cover? What
is explicitly out of scope?_

## 2. System Description

_Reference `SYSTMONE_INTEGRATION_REPORT.md` §2 (Project Overview) as the
starting technical description — the CSO should confirm it's still
accurate at the time this CSCR is written, since the codebase moves faster
than this document will._

## 3. Clinical Risk Management Process

_Describe the process used to identify, analyse, and control hazards —
normally: hazard identification → hazard log (see
`DCB0129-hazard-log-template.md`) → risk evaluation → mitigation → residual
risk acceptance._

## 4. Hazard Log Summary

_Reference the completed hazard log. Summarize: total hazards identified,
how many remain open, how many have accepted residual risk, and who
accepted them._

## 5. Test Evidence

_What testing was performed against each mitigation? For AI-generated
clinical content specifically, this should include the red-teaming
described in `docs/regulatory/medical-device-classification-worksheet.md`
(does `diagnosis`/`differentialDiagnosis`/`treatment` ever get populated
when the clinician didn't actually state one?)._

## 6. Training and Deployment Considerations

_What must a deploying organisation (a GP practice/PCN) do or know before
using this system safely? This feeds their own DCB0160 assessment — see
SYSTMONE_INTEGRATION_REPORT.md §5._

## 7. Residual Risks and Sign-off

_List every hazard whose residual risk was accepted rather than
eliminated, with the accepting authority's name and role. No CSCR is
complete without this — silence here is not the same as "no residual
risk."_

## 8. Clinical Safety Officer Declaration

_Name, GMC/NMC registration number, and signed declaration that this
report reflects their independent clinical judgement._

---

**Do not present a filled-in version of this template as a real CSCR
unless every section above was genuinely authored by a qualified CSO who
reviewed the live system, not just this codebase.**
