import type { Content, DynamicContent, TDocumentDefinitions, TVirtualFileSystem } from 'pdfmake/interfaces'
import { EMPTY_PLACEHOLDER, QA_HEADING, metaRows, sectionDepth } from './buildReport'
import { downloadBlob } from './download'
import type { ReportDoc, ReportGroup } from './types'

type PdfMake = typeof import('pdfmake/build/pdfmake')

/** A4 width (595.28pt) less the left and right page margins. */
const CONTENT_WIDTH = 515

let pdfMakePromise: Promise<PdfMake> | null = null

/**
 * pdfmake and its embedded Roboto font data are several hundred KB gzipped, so
 * they are fetched on first export rather than shipped in the initial bundle.
 */
function loadPdfMake(): Promise<PdfMake> {
  if (!pdfMakePromise) {
    pdfMakePromise = (async () => {
      const [pdfMakeModule, vfsModule] = await Promise.all([
        import('pdfmake/build/pdfmake'),
        import('pdfmake/build/vfs_fonts'),
      ])
      // Both are UMD bundles: depending on how CJS interop is applied the
      // instance is either the namespace itself or sits on `.default`, so
      // accept whichever shape arrives.
      const pdfMake = (pdfMakeModule as unknown as { default?: PdfMake }).default ?? pdfMakeModule
      const vfs =
        (vfsModule as unknown as { default?: TVirtualFileSystem }).default ??
        (vfsModule as unknown as TVirtualFileSystem)
      pdfMake.addVirtualFileSystem(vfs)
      return pdfMake
    })().catch(err => {
      // Do not cache a rejection — a failed first export must not disable the
      // button for the rest of the session.
      pdfMakePromise = null
      throw err
    })
  }
  return pdfMakePromise
}

function groupContent(group: ReportGroup): Content[] {
  const out: Content[] = []
  if (group.heading) out.push({ text: group.heading, style: 'groupHeading' })

  const headingStyle = sectionDepth(group) === 1 ? 'sectionHeading' : 'subSectionHeading'
  for (const block of group.blocks) {
    out.push({ text: block.heading, style: headingStyle })
    if (block.empty) {
      out.push({ text: EMPTY_PLACEHOLDER, style: 'placeholder' })
    } else if (block.kind === 'paragraph') {
      out.push({ text: block.text ?? '', style: 'body' })
    } else {
      out.push({ ul: block.items ?? [], style: 'body' })
    }
  }

  return out
}

export function toDocDefinition(doc: ReportDoc): TDocumentDefinitions {
  const content: Content[] = [
    { text: doc.meta.title, style: 'title' },
    {
      style: 'meta',
      layout: 'noBorders',
      margin: [0, 0, 0, 10],
      table: {
        widths: ['auto', '*'],
        body: metaRows(doc.meta).map(row => [
          { text: `${row.label}:`, bold: true },
          { text: row.value },
        ]),
      },
    },
    {
      margin: [0, 0, 0, 4],
      canvas: [
        { type: 'line', x1: 0, y1: 0, x2: CONTENT_WIDTH, y2: 0, lineWidth: 0.5, lineColor: '#cccccc' },
      ],
    },
  ]

  for (const group of doc.groups) content.push(...groupContent(group))

  if (doc.qa.length) {
    content.push({ text: QA_HEADING, style: 'groupHeading' })
    for (const turn of doc.qa) {
      content.push({
        style: 'body',
        margin: [0, 0, 0, 6],
        text: [{ text: turn.role === 'user' ? 'Q: ' : 'A: ', bold: true }, turn.content],
      })
    }
  }

  const footer: DynamicContent = (currentPage, pageCount) => ({
    margin: [40, 12, 40, 0],
    columns: [
      { text: doc.disclaimer, style: 'footer', width: '*' },
      { text: `${currentPage} / ${pageCount}`, style: 'footer', width: 'auto', alignment: 'right' },
    ],
  })

  return {
    pageSize: 'A4',
    pageMargins: [40, 40, 40, 64],
    info: { title: doc.meta.title, subject: doc.meta.templateName },
    content,
    footer,
    defaultStyle: { font: 'Roboto', fontSize: 10, lineHeight: 1.25 },
    styles: {
      title: { fontSize: 18, bold: true, margin: [0, 0, 0, 8] },
      meta: { fontSize: 9, color: '#555555' },
      groupHeading: { fontSize: 13, bold: true, margin: [0, 14, 0, 4] },
      sectionHeading: { fontSize: 11, bold: true, margin: [0, 10, 0, 3] },
      subSectionHeading: { fontSize: 10, bold: true, margin: [0, 8, 0, 2] },
      body: { fontSize: 10, margin: [0, 0, 0, 2] },
      placeholder: { fontSize: 10, italics: true, color: '#777777', margin: [0, 0, 0, 2] },
      footer: { fontSize: 7, color: '#777777' },
    },
  }
}

/** Split out from downloadPdf so PDF generation can be exercised without a DOM click. */
export async function renderPdfBlob(doc: ReportDoc): Promise<Blob> {
  const pdfMake = await loadPdfMake()
  return pdfMake.createPdf(toDocDefinition(doc)).getBlob()
}

export async function downloadPdf(doc: ReportDoc, filename: string): Promise<void> {
  downloadBlob(await renderPdfBlob(doc), filename)
}
