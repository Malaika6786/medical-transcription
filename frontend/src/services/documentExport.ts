// Turns the free-form, user-edited HTML that RichTextEditor produces (the
// primary "Generate Document" flow's output — Ambient AI, File
// Transcription, and Saved Sessions all edit a document this way) into the
// same ReportDoc shape the AI Assistant's "Generate Report" dialog already
// uses, so it can go through the exact same tested PDF/DOCX/Markdown
// exporters (services/report) instead of a second, parallel implementation.
import {
  REPORT_DISCLAIMER,
  REPORT_FORMATS,
  exportReport,
  reportFilename,
  type ExtractionKey,
  type ReportBlock,
  type ReportDoc,
  type ReportFormat,
} from './report'

export { REPORT_FORMATS }
export type { ReportFormat }

// block.key only identifies a ReportBlock for Vue's v-for; none of the PDF,
// DOCX, or Markdown writers read it. A single placeholder is safe here
// since these blocks don't come from an AiExtraction field.
const PLACEHOLDER_KEY = 'summary' as ExtractionKey

/**
 * Walks the top-level elements of edited document HTML, grouping each
 * heading (h1-h4) with the paragraph/list content that follows it into one
 * ReportBlock. Headingless leading content becomes a single untitled block.
 */
function parseHtmlToBlocks(html: string): ReportBlock[] {
  const container = document.createElement('div')
  container.innerHTML = html

  const blocks: ReportBlock[] = []
  let heading = ''
  let paragraphs: string[] = []
  let listItems: string[] = []

  const flush = () => {
    if (!heading && paragraphs.length === 0 && listItems.length === 0) return
    if (listItems.length > 0 && paragraphs.length === 0) {
      blocks.push({ key: PLACEHOLDER_KEY, heading, kind: 'list', items: listItems, empty: false })
    } else {
      const text = [...paragraphs, ...listItems.map(i => `• ${i}`)].join('\n\n')
      blocks.push({ key: PLACEHOLDER_KEY, heading, kind: 'paragraph', text, empty: text.length === 0 })
    }
    heading = ''
    paragraphs = []
    listItems = []
  }

  Array.from(container.children).forEach(el => {
    const tag = el.tagName.toLowerCase()
    if (/^h[1-4]$/.test(tag)) {
      flush()
      heading = el.textContent?.trim() || ''
    } else if (tag === 'ul' || tag === 'ol') {
      el.querySelectorAll('li').forEach(li => {
        const text = li.textContent?.trim()
        if (text) listItems.push(text)
      })
    } else {
      const text = el.textContent?.trim()
      if (text) paragraphs.push(text)
    }
  })
  flush()

  if (blocks.length === 0) {
    const text = container.textContent?.trim() || ''
    blocks.push({ key: PLACEHOLDER_KEY, heading: '', kind: 'paragraph', text, empty: !text })
  }
  return blocks
}

export interface DocumentExportMeta {
  title: string
  templateName?: string
  patientName?: string
  clinician?: string
  consultationDate?: string
}

function buildExportDoc(html: string, meta: DocumentExportMeta): ReportDoc {
  return {
    meta: {
      title: meta.title || 'Generated Document',
      templateName: meta.templateName || 'Clinical Document',
      patientName: meta.patientName,
      clinician: meta.clinician,
      consultationDate: meta.consultationDate,
      generatedAt: new Date().toISOString(),
    },
    groups: [{ blocks: parseHtmlToBlocks(html) }],
    qa: [],
    disclaimer: REPORT_DISCLAIMER,
  }
}

/**
 * Exports the given edited document HTML as a real PDF/DOCX/Markdown file —
 * same writers, same look, as the AI Assistant's report export.
 */
export async function exportGeneratedDocument(
  html: string,
  format: ReportFormat,
  meta: DocumentExportMeta
): Promise<string> {
  if (!html || !html.replace(/<[^>]*>/g, '').trim()) {
    throw new Error('Nothing to export — the document is empty.')
  }
  const doc = buildExportDoc(html, meta)
  return exportReport(doc, format)
}

export { reportFilename }
