// Package pdfgen renders a plain clinical letter (title + heading/body
// sections) as a PDF, with zero third-party dependencies — deliberately
// minimal: one font (Helvetica), one page size (US Letter-ish via A4
// dimensions), left-aligned text with simple word-wrap. This exists
// because GP Connect: Send Document requires the document content to be a
// PDF (NHS Digital: "the PDF is created within the sending organisation's
// system" — see SYSTMONE_INTEGRATION_REPORT.md §3), and the project has no
// PDF generation anywhere in the Go backend today (the frontend's pdfmake
// dependency is browser-side JS, not reachable from here).
//
// This intentionally does not attempt rich layout, embedded fonts, tables,
// or images — the one thing it needs to do is turn a
// auth.SavedDocument's sections into something a GP practice can file as a
// consultation summary attachment.
package pdfgen

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	pageWidth   = 595.28 // A4, points
	pageHeight  = 841.89
	marginLeft  = 56.0
	marginTop   = 56.0
	marginRight = 56.0
	lineHeight  = 14.0
	bodyFontPt  = 10.5
	headingPt   = 13.0
	titlePt     = 16.0
	charsPerPt  = 0.52 // rough Helvetica average-glyph-width heuristic for wrapping
)

// Section is one heading + body block, mirroring auth.DocumentSection
// (kept decoupled from the auth package so pdfgen has no dependency on it —
// the caller maps DocumentSection -> pdfgen.Section).
type Section struct {
	Heading string
	Body    string
}

// Letter is the input to Render: a title, optional patient/author byline,
// and the section content.
type Letter struct {
	Title    string
	Byline   string // e.g. "Patient: Jane Doe (NHS 485 777 3456)  ·  Generated 2026-09-07"
	Sections []Section
}

// Render produces a complete, valid single- or multi-page PDF for letter.
func Render(letter Letter) []byte {
	lines := layoutLines(letter)
	pages := paginate(lines)
	return buildPDF(pages)
}

// RenderBase64 is Render followed by base64 encoding — the form
// internal/nhs.BuildSendDocumentBundle's PDFBase64 field expects.
func RenderBase64(letter Letter) string {
	return base64.StdEncoding.EncodeToString(Render(letter))
}

type line struct {
	text string
	size float64
	bold bool
	gap  float64 // extra space *after* this line
}

func layoutLines(letter Letter) []line {
	usableWidth := pageWidth - marginLeft - marginRight
	maxChars := int(usableWidth / (bodyFontPt / charsPerPt))
	if maxChars < 20 {
		maxChars = 20
	}

	var lines []line
	lines = append(lines, line{text: letter.Title, size: titlePt, bold: true, gap: 6})
	if letter.Byline != "" {
		lines = append(lines, line{text: letter.Byline, size: 9, gap: 16})
	} else {
		lines = append(lines, line{gap: 10})
	}

	for _, sec := range letter.Sections {
		if sec.Heading != "" {
			lines = append(lines, line{text: sec.Heading, size: headingPt, bold: true, gap: 4})
		}
		body := strings.TrimSpace(sec.Body)
		if body == "" {
			lines = append(lines, line{gap: 10})
			continue
		}
		for _, para := range strings.Split(body, "\n") {
			if strings.TrimSpace(para) == "" {
				lines = append(lines, line{gap: lineHeight * 0.5})
				continue
			}
			for _, wrapped := range wrapText(para, maxChars) {
				lines = append(lines, line{text: wrapped, size: bodyFontPt})
			}
		}
		lines = append(lines, line{gap: 12})
	}
	return lines
}

func wrapText(text string, maxChars int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var out []string
	cur := words[0]
	for _, w := range words[1:] {
		if len(cur)+1+len(w) > maxChars {
			out = append(out, cur)
			cur = w
			continue
		}
		cur += " " + w
	}
	out = append(out, cur)
	return out
}

func paginate(lines []line) [][]line {
	usableHeight := pageHeight - marginTop - marginTop // symmetric top/bottom margin
	var pages [][]line
	var cur []line
	y := 0.0
	for _, l := range lines {
		// Mirror renderContentStream's treatment exactly: a gap-only line
		// (l.size == 0) advances y by just its gap, not a full line height.
		needed := l.size + l.gap
		if y+needed > usableHeight && len(cur) > 0 {
			pages = append(pages, cur)
			cur = nil
			y = 0
		}
		cur = append(cur, l)
		y += needed
	}
	if len(cur) > 0 {
		pages = append(pages, cur)
	}
	if len(pages) == 0 {
		pages = [][]line{{}}
	}
	return pages
}

// buildPDF hand-assembles a minimal valid PDF: one Catalog, one Pages tree,
// one Page + one content stream per rendered page, one shared Helvetica
// font resource, and a correct xref table + trailer. PDF object numbering:
// 1 = Catalog, 2 = Pages, 3 = Font, then for each page i (0-based):
// (4+2i) = Page object, (5+2i) = its content stream.
func buildPDF(pages [][]line) []byte {
	var objects [][]byte
	// Placeholder for Catalog (obj 1) and Pages (obj 2) — filled in after we
	// know the page object numbers.
	objects = append(objects, nil, nil)
	// obj 3: font
	objects = append(objects, []byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"))
	objects = append(objects, []byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >>"))

	pageObjNums := make([]int, len(pages))
	for i, pageLines := range pages {
		content := renderContentStream(pageLines)
		contentObjNum := len(objects) + 2 // +1 for the page obj we add right after, +1 for 1-indexing
		pageDict := fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> /Contents %d 0 R >>",
			pageWidth, pageHeight, contentObjNum,
		)
		objects = append(objects, []byte(pageDict))
		pageObjNums[i] = len(objects) // 1-indexed object number of the page we just appended

		streamObj := buildStreamObject(content)
		objects = append(objects, streamObj)
	}

	kids := make([]string, len(pageObjNums))
	for i, n := range pageObjNums {
		kids[i] = fmt.Sprintf("%d 0 R", n)
	}
	objects[0] = []byte("<< /Type /Catalog /Pages 2 0 R >>")
	objects[1] = []byte(fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pageObjNums)))

	return assemblePDF(objects)
}

func renderContentStream(lines []line) []byte {
	var buf bytes.Buffer
	y := pageHeight - marginTop
	buf.WriteString("BT\n")
	for _, l := range lines {
		lh := l.size
		if lh == 0 {
			y -= l.gap
			continue
		}
		y -= lh
		font := "/F1"
		if l.bold {
			font = "/F2"
		}
		fmt.Fprintf(&buf, "%s %.1f Tf\n", font, l.size)
		fmt.Fprintf(&buf, "1 0 0 1 %.2f %.2f Tm\n", marginLeft, y)
		fmt.Fprintf(&buf, "(%s) Tj\n", escapePDFString(l.text))
		y -= l.gap
	}
	buf.WriteString("ET\n")
	return buf.Bytes()
}

func escapePDFString(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
	return replacer.Replace(s)
}

func buildStreamObject(content []byte) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "<< /Length %d >>\nstream\n", len(content))
	buf.Write(content)
	buf.WriteString("\nendstream")
	return buf.Bytes()
}

// assemblePDF writes the header, every numbered object, the xref table,
// and the trailer, tracking byte offsets as it goes (required for a valid
// xref table).
func assemblePDF(objects [][]byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	offsets := make([]int, len(objects)+1) // 1-indexed; offsets[0] unused
	for i, obj := range objects {
		objNum := i + 1
		offsets[objNum] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n", objNum)
		buf.Write(obj)
		buf.WriteString("\nendobj\n")
	}

	xrefStart := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(objects)+1, xrefStart)

	return buf.Bytes()
}
