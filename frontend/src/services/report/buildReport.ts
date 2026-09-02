import type { AiChatMessage, AiExtraction } from '@/composables/useAiAssistant'
import { getTemplate, sectionDef, type ReportTemplate } from './templates'
import type {
  ExtractionKey,
  ReportBlock,
  ReportDoc,
  ReportGroup,
  ReportMeta,
  ReportQaTurn,
} from './types'

/** Printed instead of content for always-render sections that found nothing. */
export const EMPTY_PLACEHOLDER = 'None mentioned'

export const REPORT_DISCLAIMER =
  'AI-generated documentation aid. Every section is derived from an automated extraction ' +
  'of this consultation, not from clinician dictation. Verify clinically before use or filing.'

const paragraphText = (extraction: AiExtraction, key: ExtractionKey): string => {
  const value = extraction[key]
  return typeof value === 'string' ? value.trim() : ''
}

const listItems = (extraction: AiExtraction, key: ExtractionKey): string[] => {
  const value = extraction[key]
  return Array.isArray(value) ? value.map(item => item.trim()).filter(Boolean) : []
}

/** True when the extraction holds nothing for this section. */
export function isSectionEmpty(extraction: AiExtraction, key: ExtractionKey): boolean {
  return sectionDef(key).kind === 'paragraph'
    ? !paragraphText(extraction, key)
    : listItems(extraction, key).length === 0
}

/** One row of the dialog's section checklist. */
export interface SectionChoice {
  key: ExtractionKey
  label: string
  /** The extraction found nothing for this section. */
  empty: boolean
  /**
   * False for empty sections that carry no clinical meaning — they are shown
   * greyed out rather than hidden, so it is visible that the report adapts to
   * what the consultation actually contained.
   */
  selectable: boolean
}

export function sectionChoices(template: ReportTemplate, extraction: AiExtraction): SectionChoice[] {
  return template.groups.flatMap(group =>
    group.keys.map(key => {
      const def = sectionDef(key)
      const empty = isSectionEmpty(extraction, key)
      return { key, label: def.label, empty, selectable: !empty || def.alwaysRender === true }
    })
  )
}

/** Everything the template offers that has content, plus the always-render sections. */
export function defaultSelection(template: ReportTemplate, extraction: AiExtraction): ExtractionKey[] {
  return sectionChoices(template, extraction)
    .filter(choice => choice.selectable)
    .map(choice => choice.key)
}

export interface ReportMetaInput {
  title: string
  patientName?: string
  clinician?: string
  /** Pre-formatted by the caller, so the builder stays pure and locale-free. */
  consultationDate?: string
  sessionId?: string
  /** Pre-formatted by the caller. Defaults to an ISO timestamp. */
  generatedAt?: string
}

export interface BuildReportInput {
  extraction: AiExtraction
  templateKey: string
  selectedKeys: ExtractionKey[]
  meta: ReportMetaInput
  chat?: AiChatMessage[]
  /** Defaults to true when the chat thread has turns. */
  includeQa?: boolean
}

const trimToUndefined = (value?: string): string | undefined => {
  const trimmed = value?.trim()
  return trimmed ? trimmed : undefined
}

function buildBlock(extraction: AiExtraction, key: ExtractionKey): ReportBlock | null {
  const def = sectionDef(key)

  if (def.kind === 'paragraph') {
    const text = paragraphText(extraction, key)
    if (!text && !def.alwaysRender) return null
    return { key, heading: def.label, kind: 'paragraph', text: text || EMPTY_PLACEHOLDER, empty: !text }
  }

  const items = listItems(extraction, key)
  if (!items.length && !def.alwaysRender) return null
  return { key, heading: def.label, kind: 'list', items, empty: items.length === 0 }
}

/**
 * Assemble the report's intermediate representation. Deterministic and
 * side-effect free: no LLM call, no network, nothing the clinician has not
 * already seen on screen.
 *
 * Groups that end up with no rendered sections are dropped, so a SOAP-style
 * report never prints an "Objective" heading with nothing beneath it.
 */
export function buildReportDoc(input: BuildReportInput): ReportDoc {
  const template = getTemplate(input.templateKey)
  const selected = new Set(input.selectedKeys)

  const groups: ReportGroup[] = []
  for (const templateGroup of template.groups) {
    const blocks: ReportBlock[] = []
    for (const key of templateGroup.keys) {
      if (!selected.has(key)) continue
      const block = buildBlock(input.extraction, key)
      if (block) blocks.push(block)
    }
    if (blocks.length) groups.push({ heading: templateGroup.heading, blocks })
  }

  const chat = input.chat ?? []
  const includeQa = input.includeQa ?? chat.length > 0
  const qa: ReportQaTurn[] = includeQa
    ? chat.map(message => ({ role: message.role, content: message.content.trim() }))
    : []

  const meta: ReportMeta = {
    title: input.meta.title.trim() || 'Consultation Report',
    templateName: template.name,
    patientName: trimToUndefined(input.meta.patientName),
    clinician: trimToUndefined(input.meta.clinician),
    consultationDate: trimToUndefined(input.meta.consultationDate),
    sessionId: trimToUndefined(input.meta.sessionId),
    generatedAt: trimToUndefined(input.meta.generatedAt) ?? new Date().toISOString(),
  }

  return { meta, groups, qa, disclaimer: REPORT_DISCLAIMER }
}

/** Header rows, in print order, with blank fields already dropped. */
export function metaRows(meta: ReportMeta): { label: string; value: string }[] {
  const rows: { label: string; value: string }[] = [{ label: 'Template', value: meta.templateName }]
  if (meta.patientName) rows.push({ label: 'Patient', value: meta.patientName })
  if (meta.clinician) rows.push({ label: 'Clinician', value: meta.clinician })
  if (meta.consultationDate) rows.push({ label: 'Consultation date', value: meta.consultationDate })
  if (meta.sessionId) rows.push({ label: 'Session ID', value: meta.sessionId })
  rows.push({ label: 'Generated', value: meta.generatedAt })
  return rows
}

/** Heading depth for a section: one deeper when the group has a super-heading. */
export const sectionDepth = (group: ReportGroup): 1 | 2 => (group.heading ? 2 : 1)

export const QA_HEADING = 'Follow-up Questions'
