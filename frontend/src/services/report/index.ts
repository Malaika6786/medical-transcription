import { downloadBlob, reportFilename } from './download'
import { toMarkdown } from './markdown'
import type { ReportDoc } from './types'

export type ReportFormat = 'pdf' | 'docx' | 'md'

export interface ReportFormatOption {
  value: ReportFormat
  label: string
  ext: string
  icon: string
}

export const REPORT_FORMATS: ReportFormatOption[] = [
  { value: 'pdf', label: 'PDF', ext: 'pdf', icon: 'mdi-file-pdf-box' },
  { value: 'docx', label: 'Word (.docx)', ext: 'docx', icon: 'mdi-file-word-box' },
  { value: 'md', label: 'Markdown', ext: 'md', icon: 'mdi-language-markdown' },
]

/**
 * Serialize the report and hand it to the browser as a download. The PDF and
 * DOCX writers pull their libraries in on demand, so choosing Markdown costs
 * nothing extra.
 */
export async function exportReport(doc: ReportDoc, format: ReportFormat): Promise<string> {
  const option = REPORT_FORMATS.find(f => f.value === format)
  if (!option) throw new Error(`Unsupported report format "${format}"`)

  const filename = reportFilename(doc.meta.templateName, option.ext)

  switch (format) {
    case 'pdf': {
      const { downloadPdf } = await import('./pdf')
      await downloadPdf(doc, filename)
      break
    }
    case 'docx': {
      const { downloadDocx } = await import('./docx')
      await downloadDocx(doc, filename)
      break
    }
    case 'md':
      downloadBlob(new Blob([toMarkdown(doc)], { type: 'text/markdown;charset=utf-8' }), filename)
      break
  }

  return filename
}

export {
  buildReportDoc,
  defaultSelection,
  isSectionEmpty,
  metaRows,
  sectionChoices,
  REPORT_DISCLAIMER,
  EMPTY_PLACEHOLDER,
} from './buildReport'
export type { BuildReportInput, ReportMetaInput, SectionChoice } from './buildReport'
export { toMarkdown } from './markdown'
export { reportFilename } from './download'
export { DEFAULT_TEMPLATE_KEY, REPORT_TEMPLATES, getTemplate, sectionDef, SECTIONS } from './templates'
export type { ReportTemplate, SectionDef } from './templates'
export type {
  ExtractionKey,
  ReportBlock,
  ReportDoc,
  ReportGroup,
  ReportMeta,
  ReportMetaSource,
  ReportQaTurn,
} from './types'
