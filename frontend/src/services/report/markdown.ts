import { EMPTY_PLACEHOLDER, QA_HEADING, metaRows, sectionDepth } from './buildReport'
import type { ReportDoc, ReportGroup } from './types'

/**
 * Serialize the report to Markdown. Content is emitted verbatim — clinical text
 * is not Markdown-escaped, because mangling a drug name or a dose to protect
 * emphasis syntax is the worse failure here.
 */
export function toMarkdown(doc: ReportDoc): string {
  const lines: string[] = [`# ${doc.meta.title}`, '']

  // Two trailing spaces force a hard line break inside the metadata block.
  const rows = metaRows(doc.meta)
  rows.forEach((row, index) => {
    const isLast = index === rows.length - 1
    lines.push(`**${row.label}:** ${row.value}${isLast ? '' : '  '}`)
  })
  lines.push('', '---', '')

  for (const group of doc.groups) lines.push(...groupLines(group))

  if (doc.qa.length) {
    lines.push(`## ${QA_HEADING}`, '')
    for (const turn of doc.qa) {
      lines.push(`**${turn.role === 'user' ? 'Q' : 'A'}:** ${turn.content}`, '')
    }
  }

  lines.push('---', '', `_${doc.disclaimer}_`, '')
  return lines.join('\n')
}

function groupLines(group: ReportGroup): string[] {
  const lines: string[] = []
  if (group.heading) lines.push(`## ${group.heading}`, '')

  const hashes = '#'.repeat(sectionDepth(group) + 1)
  for (const block of group.blocks) {
    lines.push(`${hashes} ${block.heading}`, '')
    if (block.empty) {
      lines.push(`_${EMPTY_PLACEHOLDER}_`, '')
    } else if (block.kind === 'paragraph') {
      lines.push(block.text ?? '', '')
    } else {
      lines.push(...(block.items ?? []).map(item => `- ${item}`), '')
    }
  }

  return lines
}
