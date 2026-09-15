# Prerequisites Beyond This Codebase

Closing every technical gap in
[`SYSTMONE_INTEGRATION_REPORT.md`](../../../SYSTMONE_INTEGRATION_REPORT.md)
and [`docs/nhs/README.md`](../nhs/README.md) makes this project
**integration-ready**. It does not, by itself, get it connected to a real
NHS system — three things outside the code are equally necessary, and none
of them can be satisfied by writing more software:

## 1. A registered legal organisation

DSPT, DTAC, MESH mailbox issuance, and GP Connect/SystmOne onboarding are
all granted to a registered company/organisation with a named accountable
person — not an individual developer or a demo repository. If this project
is to go further than its current demo status, this is usually the first
real blocker to resolve.

## 2. A willing NHS-side pilot partner

You cannot integrate into "NHS systems" in the abstract. You need a
specific GP practice or Primary Care Network (PCN) that agrees to pilot
with you — MESH/PDS test credentials, and the eventual live connection, are
both tied to that receiving organisation's own registration.

## 3. A named Clinical Safety Officer

DCB0129's clinical safety case must be signed off by a clinician registered
with the GMC or NMC, with training in clinical risk management. This
typically means bringing in a clinician as a co-founder, advisor, or paid
consultant — it is not something a development team can self-certify.
`docs/clinical-safety/` in this repo has the templates a real CSO would use
to do this work; it does not do the work for you.

## Suggested order of operations

1. Decide on the organisational vehicle (company registration, or partner
   with an existing DSPT-registered health-tech organisation).
2. Identify and formally engage a Clinical Safety Officer.
3. In parallel, approach candidate pilot practices/PCNs — this
   conversation alone can take months and should start early.
4. Complete DSPT and begin DTAC once the organisation exists.
5. Only once 1-3 are underway does the technical integration work in
   `internal/nhs/` have a real destination to connect to.

None of this is unusual — it's the standard path every NHS-connected
health-tech supplier goes through. It's a parallel, multi-month
organisational effort alongside the engineering, not a follow-on step after
the code is "done."
