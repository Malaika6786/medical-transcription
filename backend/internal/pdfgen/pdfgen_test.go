package pdfgen

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestRender_ProducesValidPDFHeaderAndTrailer(t *testing.T) {
	letter := Letter{
		Title:  "GP Letter",
		Byline: "Patient: Jane Doe (NHS 943 476 5919)",
		Sections: []Section{
			{Heading: "Summary", Body: "Patient presented with mild headache, resolved with rest."},
			{Heading: "Medications", Body: "Ibuprofen 400mg as needed."},
		},
	}
	out := Render(letter)

	if !bytes.HasPrefix(out, []byte("%PDF-1.4")) {
		t.Fatal("output does not start with a PDF header")
	}
	if !bytes.Contains(out, []byte("%%EOF")) {
		t.Fatal("output does not contain the PDF end-of-file marker")
	}
	if !bytes.Contains(out, []byte("xref")) {
		t.Fatal("output does not contain an xref table")
	}
	if len(out) < 200 {
		t.Fatalf("output suspiciously small: %d bytes", len(out))
	}
}

func TestRender_MultiPageForLongContent(t *testing.T) {
	var sections []Section
	longBody := strings.Repeat("This is a long clinical note line that should wrap and take up plenty of vertical space. ", 40)
	for i := 0; i < 20; i++ {
		sections = append(sections, Section{Heading: "Section", Body: longBody})
	}
	out := Render(Letter{Title: "Long Letter", Sections: sections})

	// A single-page-worth of this content would never fit; expect more than
	// one /Type /Page object in the resulting PDF.
	count := bytes.Count(out, []byte("/Type /Page "))
	if count < 2 {
		t.Fatalf("expected multiple pages for long content, found %d /Type /Page objects", count)
	}
}

func TestRenderBase64_IsValidBase64(t *testing.T) {
	encoded := RenderBase64(Letter{Title: "Test"})
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("RenderBase64 output is not valid base64: %v", err)
	}
	if !bytes.HasPrefix(decoded, []byte("%PDF-1.4")) {
		t.Fatal("decoded RenderBase64 output is not a PDF")
	}
}

func TestWrapText(t *testing.T) {
	lines := wrapText("one two three four five", 11)
	for _, l := range lines {
		if len(l) > 11 {
			// A single word longer than maxChars is allowed to overflow;
			// this input has no such word, so every line must fit.
			t.Errorf("line %q exceeds maxChars=11", l)
		}
	}
	if len(lines) < 2 {
		t.Errorf("expected wrapping to produce multiple lines, got %d", len(lines))
	}
}
