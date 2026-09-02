import type { Paragraph as DocxParagraph } from 'docx'
import { EMPTY_PLACEHOLDER, QA_HEADING, metaRows, sectionDepth } from './buildReport'
import { downloadBlob } from './download'
import type { ReportDoc } from './types'

/** docx measures font size in half-points: 14 = 7pt, 18 = 9pt. */
const META_SIZE = 18
const FOOTER_SIZE = 14
const MUTED = '777777'

/**
 * Serialize to a real OOXML .docx. The library is imported on first export so
 * it stays out of the initial bundle.
 *
 * Split out from downloadDocx so generation can be exercised without a DOM click.
 */
export async function renderDocxBlob(doc: ReportDoc): Promise<Blob> {
  const { AlignmentType, Document, Footer, HeadingLevel, Packer, PageNumber, Paragraph, TextRun } =
    await import('docx')

  const children: DocxParagraph[] = [
    new Paragraph({ text: doc.meta.title, heading: HeadingLevel.TITLE }),
  ]

  for (const row of metaRows(doc.meta)) {
    children.push(
      new Paragraph({
        children: [
          new TextRun({ text: `${row.label}: `, bold: true, size: META_SIZE }),
          new TextRun({ text: row.value, size: META_SIZE }),
        ],
      })
    )
  }

  for (const group of doc.groups) {
    if (group.heading) {
      children.push(new Paragraph({ text: group.heading, heading: HeadingLevel.HEADING_1 }))
    }
    const headingLevel = sectionDepth(group) === 1 ? HeadingLevel.HEADING_1 : HeadingLevel.HEADING_2

    for (const block of group.blocks) {
      children.push(new Paragraph({ text: block.heading, heading: headingLevel }))

      if (block.empty) {
        children.push(
          new Paragraph({
            children: [new TextRun({ text: EMPTY_PLACEHOLDER, italics: true, color: MUTED })],
          })
        )
      } else if (block.kind === 'paragraph') {
        children.push(new Paragraph({ text: block.text ?? '' }))
      } else {
        for (const item of block.items ?? []) {
          children.push(new Paragraph({ text: item, bullet: { level: 0 } }))
        }
      }
    }
  }

  if (doc.qa.length) {
    children.push(new Paragraph({ text: QA_HEADING, heading: HeadingLevel.HEADING_1 }))
    for (const turn of doc.qa) {
      children.push(
        new Paragraph({
          children: [
            new TextRun({ text: turn.role === 'user' ? 'Q: ' : 'A: ', bold: true }),
            new TextRun({ text: turn.content }),
          ],
        })
      )
    }
  }

  const footer = new Footer({
    children: [
      new Paragraph({
        children: [
          new TextRun({ text: doc.disclaimer, size: FOOTER_SIZE, color: MUTED, italics: true }),
        ],
      }),
      new Paragraph({
        alignment: AlignmentType.RIGHT,
        children: [
          new TextRun({ text: 'Page ', size: FOOTER_SIZE, color: MUTED }),
          new TextRun({ children: [PageNumber.CURRENT], size: FOOTER_SIZE, color: MUTED }),
          new TextRun({ text: ' of ', size: FOOTER_SIZE, color: MUTED }),
          new TextRun({ children: [PageNumber.TOTAL_PAGES], size: FOOTER_SIZE, color: MUTED }),
        ],
      }),
    ],
  })

  const document = new Document({
    title: doc.meta.title,
    subject: doc.meta.templateName,
    sections: [{ properties: {}, footers: { default: footer }, children }],
  })

  return Packer.toBlob(document)
}

export async function downloadDocx(doc: ReportDoc, filename: string): Promise<void> {
  downloadBlob(await renderDocxBlob(doc), filename)
}
