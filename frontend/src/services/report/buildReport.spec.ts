import { describe, expect, it } from 'vitest'
import type { AiChatMessage, AiExtraction } from '@/composables/useAiAssistant'
import {
  EMPTY_PLACEHOLDER,
  buildReportDoc,
  defaultSelection,
  isSectionEmpty,
  sectionChoices,
} from './buildReport'
import { reportFilename } from './download'
import { SECTIONS, getTemplate } from './templates'

/**
 * A realistic extraction: labTests, procedures and riskFactors found nothing,
 * and allergies is empty but clinically meaningful (always renders).
 */
const extraction: AiExtraction = {
  summary: 'Two weeks of intermittent frontal headaches, worse in the afternoons.',
  chiefComplaint: 'Intermittent headaches for two weeks',
  history: ['No prior head injury', 'Similar episodes three years ago'],
  allergies: [],
  medications: ['ibuprofen 400 mg as needed'],
  symptoms: ['frontal headache', 'photophobia'],
  diagnosis: ['Tension-type headache'],
  differentialDiagnosis: ['Migraine without aura'],
  treatment: ['Continue ibuprofen 400 mg PRN', 'Sleep hygiene advice'],
  labTests: [],
  procedures: [],
  followUp: ['Review in four weeks if unresolved'],
  riskFactors: [],
  medicalTerms: ['photophobia'],
  actionItems: ['Patient to keep a headache diary'],
}

const meta = {
  title: 'Ambient Session — 30 Jul 2026',
  patientName: 'Jane Doe',
  clinician: 'Dr A. Smith',
  consultationDate: '30 July 2026, 14:05',
  sessionId: 'sess-abc123',
  generatedAt: '30 July 2026, 14:22',
}

const chat: AiChatMessage[] = [
  { role: 'user', content: 'What allergies were mentioned?' },
  { role: 'assistant', content: 'The extracted record does not include any allergies.' },
]

const build = (templateKey: string, overrides: Partial<Parameters<typeof buildReportDoc>[0]> = {}) => {
  const template = getTemplate(templateKey)
  return buildReportDoc({
    extraction,
    templateKey,
    selectedKeys: defaultSelection(template, extraction),
    meta,
    ...overrides,
  })
}

const headings = (templateKey: string) =>
  build(templateKey).groups.flatMap(group => group.blocks.map(block => block.heading))

describe('isSectionEmpty', () => {
  it('treats a blank paragraph and an empty list as empty', () => {
    expect(isSectionEmpty({ ...extraction, summary: '   ' }, 'summary')).toBe(true)
    expect(isSectionEmpty(extraction, 'labTests')).toBe(true)
  })

  it('treats a list of only whitespace entries as empty', () => {
    expect(isSectionEmpty({ ...extraction, symptoms: ['  ', ''] }, 'symptoms')).toBe(true)
  })

  it('reports populated sections as non-empty', () => {
    expect(isSectionEmpty(extraction, 'summary')).toBe(false)
    expect(isSectionEmpty(extraction, 'diagnosis')).toBe(false)
  })
})

describe('sectionChoices', () => {
  it('offers every field of the full template, in panel order', () => {
    const choices = sectionChoices(getTemplate('full'), extraction)
    expect(choices.map(c => c.key)).toEqual(SECTIONS.map(s => s.key))
  })

  it('marks empty sections unselectable so they cannot be ticked', () => {
    const byKey = new Map(sectionChoices(getTemplate('full'), extraction).map(c => [c.key, c]))
    expect(byKey.get('labTests')).toMatchObject({ empty: true, selectable: false })
    expect(byKey.get('procedures')).toMatchObject({ empty: true, selectable: false })
  })

  it('keeps empty always-render sections selectable', () => {
    const byKey = new Map(sectionChoices(getTemplate('full'), extraction).map(c => [c.key, c]))
    expect(byKey.get('allergies')).toMatchObject({ empty: true, selectable: true })
  })
})

describe('defaultSelection', () => {
  it('ticks everything with content plus the always-render sections', () => {
    const selected = defaultSelection(getTemplate('full'), extraction)
    expect(selected).toContain('allergies')
    expect(selected).toContain('diagnosis')
    expect(selected).not.toContain('labTests')
    expect(selected).not.toContain('procedures')
    expect(selected).not.toContain('riskFactors')
  })
})

describe('buildReportDoc — full template', () => {
  it('renders one unheaded group', () => {
    const doc = build('full')
    expect(doc.groups).toHaveLength(1)
    expect(doc.groups[0].heading).toBeUndefined()
  })

  it('omits empty sections and keeps the always-render ones', () => {
    expect(headings('full')).toEqual([
      'Summary',
      'Chief Complaint',
      'Diagnosis',
      'Differential Diagnosis',
      'Allergies',
      'Medications',
      'Symptoms',
      'Medical Terms',
      'History',
      'Treatment',
      'Follow-up',
      'Action Items',
    ])
  })

  it('marks an empty always-render section rather than dropping its content silently', () => {
    const allergies = build('full').groups[0].blocks.find(b => b.key === 'allergies')
    expect(allergies).toMatchObject({ kind: 'list', empty: true, items: [] })
  })

  it('trims whitespace-only list entries', () => {
    const doc = buildReportDoc({
      extraction: { ...extraction, symptoms: ['  frontal headache ', '   ', ''] },
      templateKey: 'full',
      selectedKeys: ['symptoms'],
      meta,
    })
    expect(doc.groups[0].blocks[0].items).toEqual(['frontal headache'])
  })

  it('drops a section the user unticked', () => {
    const doc = buildReportDoc({
      extraction,
      templateKey: 'full',
      selectedKeys: defaultSelection(getTemplate('full'), extraction).filter(k => k !== 'medications'),
      meta,
    })
    expect(headings('full')).toContain('Medications')
    expect(doc.groups[0].blocks.map(b => b.key)).not.toContain('medications')
  })

  it('produces no groups at all when nothing is selected', () => {
    const doc = buildReportDoc({ extraction, templateKey: 'full', selectedKeys: [], meta })
    expect(doc.groups).toEqual([])
  })
})

describe('buildReportDoc — SOAP-style template', () => {
  it('groups fields under the S/O/A/P headings', () => {
    const doc = build('soap-style')
    expect(doc.groups.map(g => g.heading)).toEqual([
      undefined,
      'Subjective',
      'Assessment',
      'Plan',
      'Reference',
    ])
  })

  it('drops the Objective group entirely, since both its sections are empty', () => {
    expect(build('soap-style').groups.map(g => g.heading)).not.toContain('Objective')
  })

  it('places each field under its mapped heading', () => {
    const doc = build('soap-style')
    const group = (heading: string) => doc.groups.find(g => g.heading === heading)
    expect(group('Subjective')?.blocks.map(b => b.key)).toEqual([
      'chiefComplaint',
      'history',
      'symptoms',
      'allergies',
    ])
    expect(group('Assessment')?.blocks.map(b => b.key)).toEqual(['diagnosis', 'differentialDiagnosis'])
    expect(group('Plan')?.blocks.map(b => b.key)).toEqual([
      'treatment',
      'medications',
      'followUp',
      'actionItems',
    ])
  })

  it('keeps the summary as an unheaded preamble', () => {
    const doc = build('soap-style')
    expect(doc.groups[0].blocks.map(b => b.key)).toEqual(['summary'])
  })
})

describe('buildReportDoc — brief template', () => {
  it('includes only the presenting problem, conclusion and next steps', () => {
    expect(headings('brief')).toEqual([
      'Summary',
      'Chief Complaint',
      'Diagnosis',
      'Medications',
      'Follow-up',
      'Action Items',
    ])
  })
})

describe('buildReportDoc — Q&A appendix', () => {
  it('is included by default when the thread has turns', () => {
    expect(build('full', { chat }).qa).toHaveLength(2)
  })

  it('is empty when the thread is empty', () => {
    expect(build('full', { chat: [] }).qa).toEqual([])
    expect(build('full').qa).toEqual([])
  })

  it('is omitted when the user opts out', () => {
    expect(build('full', { chat, includeQa: false }).qa).toEqual([])
  })

  it('preserves turn order and roles', () => {
    expect(build('full', { chat }).qa.map(t => t.role)).toEqual(['user', 'assistant'])
  })
})

describe('buildReportDoc — metadata', () => {
  it('names the template it was built from', () => {
    expect(build('soap-style').meta.templateName).toBe('SOAP-style Note')
  })

  it('drops blank optional fields instead of printing empty labels', () => {
    const doc = buildReportDoc({
      extraction,
      templateKey: 'brief',
      selectedKeys: ['summary'],
      meta: { title: 'Untitled', patientName: '   ', clinician: '' },
    })
    expect(doc.meta.patientName).toBeUndefined()
    expect(doc.meta.clinician).toBeUndefined()
    expect(doc.meta.generatedAt).toBeTruthy()
  })

  it('falls back to a generic title when none is given', () => {
    const doc = buildReportDoc({ extraction, templateKey: 'brief', selectedKeys: [], meta: { title: '  ' } })
    expect(doc.meta.title).toBe('Consultation Report')
  })

  it('always carries the verify-clinically disclaimer', () => {
    expect(build('full').disclaimer).toMatch(/Verify clinically/)
  })
})

describe('buildReportDoc — empty sections', () => {
  it('drops an empty non-always-render section even when explicitly selected', () => {
    const doc = buildReportDoc({
      extraction: { ...extraction, chiefComplaint: '' },
      templateKey: 'full',
      selectedKeys: ['chiefComplaint'],
      meta,
    })
    expect(doc.groups).toEqual([])
  })

  it('exposes the placeholder the serializers print', () => {
    expect(EMPTY_PLACEHOLDER).toBe('None mentioned')
  })
})

describe('reportFilename', () => {
  it('slugs the template name and stamps the time', () => {
    expect(reportFilename('SOAP-style Note', 'pdf', new Date('2026-07-30T14:22:01.500Z'))).toBe(
      'soap-style-note-2026-07-30T14-22-01.pdf'
    )
  })

  it('collapses punctuation and never leaves a leading or trailing dash', () => {
    expect(reportFilename('Word (.docx) — Report!', 'docx', new Date('2026-07-30T00:00:00Z'))).toBe(
      'word-docx-report-2026-07-30T00-00-00.docx'
    )
  })
})
