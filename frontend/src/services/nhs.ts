// Thin wrappers around the NHS/SystmOne integration backend routes
// (internal/handlers/patient_handler.go, nhs_handler.go, audit_handler.go)
// — see SYSTMONE_INTEGRATION_REPORT.md and docs/nhs/README.md. Every call
// here 404s until the backend is booted with FIELD_ENCRYPTION_KEY set (see
// main.go) — callers should treat a 404 on these routes as "NHS integration
// isn't configured on this deployment", not a generic error.
import api from './api'

export interface Patient {
  id: string
  nhsNumber?: string
  name: string
  dateOfBirth?: string
  sex: 'male' | 'female' | 'other' | 'unknown'
  pdsVerifiedAt?: string
  createdBy: string
  createdAt: string
  updatedAt: string
}

export interface PatientPayload {
  nhsNumber?: string
  name: string
  dateOfBirth?: string
  sex?: string
}

export interface FHIRPatientRef {
  NHSNumber: string
  Name: string
  BirthDate: string
}

export interface AuditEvent {
  id: number
  occurredAt: string
  actorId: string
  actorName: string
  action: string
  resourceType: string
  resourceId: string
  patientId?: string
  ipAddress?: string
  detail?: Record<string, unknown>
}

export async function listPatients(): Promise<Patient[]> {
  const res = await api.get('/patients')
  return res.data || []
}

export async function getPatient(id: string): Promise<Patient> {
  const res = await api.get(`/patients/${id}`)
  return res.data
}

export async function createPatient(payload: PatientPayload): Promise<Patient> {
  const res = await api.post('/patients', payload)
  return res.data
}

export async function updatePatient(id: string, payload: PatientPayload): Promise<Patient> {
  const res = await api.put(`/patients/${id}`, payload)
  return res.data
}

export async function deletePatient(id: string): Promise<void> {
  await api.delete(`/patients/${id}`)
}

export async function verifyPatientPDS(id: string): Promise<Patient> {
  const res = await api.post(`/patients/${id}/verify-pds`)
  return res.data
}

export async function pdsTrace(nhsNumber: string): Promise<FHIRPatientRef> {
  const res = await api.post('/nhs/pds/trace', { nhsNumber })
  return res.data
}

export async function pdsSearch(params: {
  familyName: string
  givenName?: string
  birthDate: string
  postcode?: string
}): Promise<FHIRPatientRef[]> {
  const res = await api.post('/nhs/pds/search', params)
  return res.data?.candidates || []
}

export async function linkSessionToPatient(sessionId: string, patientId: string): Promise<void> {
  await api.put(`/sessions/${sessionId}/patient`, { patientId })
}

export async function sendSessionToGP(sessionId: string, recipientMailboxId: string): Promise<{ meshMessageId: string }> {
  const res = await api.post(`/nhs/sessions/${sessionId}/send-to-gp`, { recipientMailboxId })
  return res.data
}

export async function listAuditEvents(filter?: { patientId?: string; actorId?: string; limit?: number }): Promise<AuditEvent[]> {
  const res = await api.get('/audit', { params: filter })
  return res.data || []
}
