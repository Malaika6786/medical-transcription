/** Trigger a browser download for an in-memory blob. */
export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

/**
 * `{template-slug}-{timestamp}.{ext}` — the same convention the file
 * transcription page already uses for its exports.
 */
export function reportFilename(templateName: string, ext: string, now: Date = new Date()): string {
  const slug = templateName.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '')
  const stamp = now.toISOString().replace(/[:.]/g, '-').slice(0, 19)
  return `${slug || 'report'}-${stamp}.${ext}`
}
