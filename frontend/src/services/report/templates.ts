import type { BlockKind, ExtractionKey } from './types'

export interface SectionDef {
  key: ExtractionKey
  label: string
  kind: BlockKind
  /**
   * Render even when the extraction found nothing. The absence of a diagnosis
   * or of a drug allergy is itself clinically meaningful, so those two print
   * "None mentioned" instead of vanishing — matching how AiAssistantPanel
   * displays them on screen.
   */
  alwaysRender?: boolean
}

/** All 15 extraction fields, in the order AiAssistantPanel displays them. */
export const SECTIONS: SectionDef[] = [
  { key: 'summary', label: 'Summary', kind: 'paragraph' },
  { key: 'chiefComplaint', label: 'Chief Complaint', kind: 'paragraph' },
  { key: 'diagnosis', label: 'Diagnosis', kind: 'list', alwaysRender: true },
  { key: 'differentialDiagnosis', label: 'Differential Diagnosis', kind: 'list' },
  { key: 'allergies', label: 'Allergies', kind: 'list', alwaysRender: true },
  { key: 'medications', label: 'Medications', kind: 'list' },
  { key: 'symptoms', label: 'Symptoms', kind: 'list' },
  { key: 'labTests', label: 'Lab Tests', kind: 'list' },
  { key: 'procedures', label: 'Procedures', kind: 'list' },
  { key: 'riskFactors', label: 'Risk Factors', kind: 'list' },
  { key: 'medicalTerms', label: 'Medical Terms', kind: 'list' },
  { key: 'history', label: 'History', kind: 'list' },
  { key: 'treatment', label: 'Treatment', kind: 'list' },
  { key: 'followUp', label: 'Follow-up', kind: 'list' },
  { key: 'actionItems', label: 'Action Items', kind: 'list' },
]

const byKey = new Map(SECTIONS.map(s => [s.key, s]))

export function sectionDef(key: ExtractionKey): SectionDef {
  const def = byKey.get(key)
  if (!def) throw new Error(`No section definition for extraction key "${key}"`)
  return def
}

/** An ordered run of sections, optionally under a super-heading. */
export interface TemplateGroup {
  heading?: string
  keys: ExtractionKey[]
}

export interface ReportTemplate {
  key: string
  name: string
  description: string
  groups: TemplateGroup[]
}

/**
 * Templates are pure regroupings and reorderings of the extraction fields —
 * report generation never re-invokes the model, so what is exported is exactly
 * what the clinician reviewed on screen.
 *
 * Note that "SOAP-style" is a mapping of extracted fields onto S/O/A/P
 * headings, NOT model-written SOAP prose. It is not equivalent to the
 * Corti-generated SOAP note on the file-transcription page.
 */
export const REPORT_TEMPLATES: ReportTemplate[] = [
  {
    key: 'full',
    name: 'Full Clinical Report',
    description: 'Every extracted section, in the order the panel shows them',
    groups: [{ keys: SECTIONS.map(s => s.key) }],
  },
  {
    key: 'soap-style',
    name: 'SOAP-style Note',
    description: 'Extracted fields grouped under Subjective, Objective, Assessment, Plan',
    groups: [
      { keys: ['summary'] },
      { heading: 'Subjective', keys: ['chiefComplaint', 'history', 'symptoms', 'allergies', 'riskFactors'] },
      { heading: 'Objective', keys: ['labTests', 'procedures'] },
      { heading: 'Assessment', keys: ['diagnosis', 'differentialDiagnosis'] },
      { heading: 'Plan', keys: ['treatment', 'medications', 'followUp', 'actionItems'] },
      { heading: 'Reference', keys: ['medicalTerms'] },
    ],
  },
  {
    key: 'brief',
    name: 'Brief Summary',
    description: 'Presenting problem, conclusion and what happens next',
    groups: [
      { keys: ['summary', 'chiefComplaint', 'diagnosis', 'medications', 'followUp', 'actionItems'] },
    ],
  },
]

export const DEFAULT_TEMPLATE_KEY = 'full'

export function getTemplate(key: string): ReportTemplate {
  return REPORT_TEMPLATES.find(t => t.key === key) ?? REPORT_TEMPLATES[0]
}
