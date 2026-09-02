import { describe, expect, it } from 'vitest'
import type { AiChatMessage, AiExtraction } from '@/composables/useAiAssistant'
import { buildReportDoc, defaultSelection } from './buildReport'
import { toMarkdown } from './markdown'
import { getTemplate } from './templates'

const extraction: AiExtraction = {
  summary: 'Two weeks of intermittent frontal headaches, worse in the afternoons.',
  chiefComplaint: 'Intermittent headaches for two weeks',
  history: ['No prior head injury'],
  allergies: [],
  medications: ['ibuprofen 400 mg as needed'],
  symptoms: ['frontal headache', 'photophobia'],
  diagnosis: ['Tension-type headache'],
  differentialDiagnosis: ['Migraine without aura'],
  treatment: ['Continue ibuprofen 400 mg PRN'],
  labTests: [],
  procedures: [],
  followUp: ['Review in four weeks if unresolved'],
  riskFactors: [],
  medicalTerms: ['photophobia'],
  actionItems: ['Patient to keep a headache diary'],
}

const chat: AiChatMessage[] = [
  { role: 'user', content: 'What allergies were mentioned?' },
  { role: 'assistant', content: 'The extracted record does not include any allergies.' },
]

/** Fixed timestamps keep the snapshot deterministic. */
const meta = {
  title: 'Ambient Session — 30 Jul 2026',
  patientName: 'Jane Doe',
  clinician: 'Dr A. Smith',
  consultationDate: '30 July 2026, 14:05',
  sessionId: 'sess-abc123',
  generatedAt: '30 July 2026, 14:22',
}

const render = (templateKey: string, withChat = false) => {
  const template = getTemplate(templateKey)
  return toMarkdown(
    buildReportDoc({
      extraction,
      templateKey,
      selectedKeys: defaultSelection(template, extraction),
      meta,
      chat: withChat ? chat : [],
    })
  )
}

describe('toMarkdown', () => {
  it('renders the full clinical report', () => {
    expect(render('full')).toMatchSnapshot()
  })

  it('renders the SOAP-style note with group headings', () => {
    expect(render('soap-style')).toMatchSnapshot()
  })

  it('renders the brief summary with the Q&A appendix', () => {
    expect(render('brief', true)).toMatchSnapshot()
  })

  it('puts sections at h2 when the group has no super-heading', () => {
    expect(render('full')).toContain('\n## Summary\n')
  })

  it('demotes sections to h3 beneath a group super-heading', () => {
    const md = render('soap-style')
    expect(md).toContain('\n## Subjective\n')
    expect(md).toContain('\n### Chief Complaint\n')
  })

  it('italicises the placeholder for an empty always-render section', () => {
    expect(render('full')).toContain('## Allergies\n\n_None mentioned_')
  })

  it('omits sections the extraction left empty', () => {
    expect(render('full')).not.toContain('Lab Tests')
    expect(render('full')).not.toContain('Risk Factors')
  })

  it('hard-breaks every metadata line except the last', () => {
    const md = render('full')
    expect(md).toContain('**Patient:** Jane Doe  \n')
    expect(md).toContain('**Generated:** 30 July 2026, 14:22\n')
    expect(md).not.toContain('**Generated:** 30 July 2026, 14:22  \n')
  })

  it('labels Q&A turns as Q and A', () => {
    const md = render('brief', true)
    expect(md).toContain('## Follow-up Questions')
    expect(md).toContain('**Q:** What allergies were mentioned?')
    expect(md).toContain('**A:** The extracted record does not include any allergies.')
  })

  it('omits the Q&A heading when there is no thread', () => {
    expect(render('brief')).not.toContain('Follow-up Questions')
  })

  it('ends with the disclaimer', () => {
    expect(render('full').trimEnd()).toMatch(/_AI-generated documentation aid\..*Verify clinically.*_$/)
  })
})
