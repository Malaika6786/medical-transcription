import type { AiExtraction } from '@/composables/useAiAssistant'

/** Any of the 15 fields on the extraction record. */
export type ExtractionKey = keyof AiExtraction

/** How a section's content is laid out, in every output format. */
export type BlockKind = 'paragraph' | 'list'

/** One rendered section of the report. */
export interface ReportBlock {
  key: ExtractionKey
  heading: string
  kind: BlockKind
  /** Set when kind is 'paragraph'. */
  text?: string
  /** Set when kind is 'list'. Empty only for always-render sections. */
  items?: string[]
  /** True when the extraction found nothing and a placeholder is printed instead. */
  empty: boolean
}

/** A run of sections, optionally under a super-heading (SOAP's S/O/A/P). */
export interface ReportGroup {
  heading?: string
  blocks: ReportBlock[]
}

/** Header metadata. Every optional field is omitted from the header when blank. */
export interface ReportMeta {
  title: string
  templateName: string
  patientName?: string
  clinician?: string
  consultationDate?: string
  sessionId?: string
  generatedAt: string
}

/**
 * Loosely-typed metadata as the host page has it: dates unformatted, absent
 * fields either missing or null. The report dialog normalizes this into the
 * strings ReportMeta requires.
 */
export interface ReportMetaSource {
  title?: string
  patientName?: string
  clinician?: string
  consultationDate?: Date | string | null
  sessionId?: string | null
}

/** One turn of the follow-up Q&A appendix. */
export interface ReportQaTurn {
  role: 'user' | 'assistant'
  content: string
}

/**
 * The single intermediate representation of a report.
 *
 * buildReportDoc() produces it once from the extraction, the chosen template
 * and the user's section selection. The markdown, PDF and DOCX serializers are
 * thin walks over this structure — which is the whole point: three independent
 * format-specific builders would drift apart within a week.
 */
export interface ReportDoc {
  meta: ReportMeta
  groups: ReportGroup[]
  /** Empty unless the user opted in and the chat thread had turns. */
  qa: ReportQaTurn[]
  disclaimer: string
}
