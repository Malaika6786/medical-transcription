package nhs

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// This file builds the FHIR message body for GP Connect: Send Document —
// NOT a general-purpose FHIR library. Send Document delivers a PDF
// consultation summary to a patient's registered GP practice over MESH,
// wrapped as a FHIR Bundle containing a Composition (the clinical metadata)
// and a Binary (the PDF itself, base64-encoded) — see
// SYSTMONE_INTEGRATION_REPORT.md §3/§6 and docs/nhs/README.md.
//
// The exact structure below follows the publicly documented shape of an
// ITK3 "Send Document" message (a FHIR Bundle of type "document", entry 0 a
// Composition, entry 1+ the content it references). The precise required
// Composition.type/category coding and any additional mandatory extensions
// are governed by NHS England's GP Connect Send Document FHIR profile,
// which was not accessible from this codebase alone — treat the coding
// values below as placeholders to confirm against the live profile
// (https://digital.nhs.uk/developer/api-catalogue/gp-connect-send-document-fhir)
// before sending anything to a real or NHS test environment. This is
// flagged explicitly rather than guessed at silently.

// FHIRPatientRef identifies the patient a document is being sent about —
// deliberately minimal (only what Send Document's envelope needs), not a
// full FHIR Patient resource.
type FHIRPatientRef struct {
	NHSNumber string // normalized 10 digits, see ValidateNHSNumber
	Name      string
	BirthDate string // ISO-8601 date, "" if unknown
}

// SendDocumentInput is everything BuildSendDocumentBundle needs to produce
// one message.
type SendDocumentInput struct {
	Patient FHIRPatientRef
	// AuthorODSCode is this organisation's ODS (Organisation Data Service)
	// code — the NHS-wide identifier for "who sent this", assigned when the
	// organisation registers as a supplier/provider. Needs verification /
	// real value once the organisational registration in
	// docs/regulatory/PREREQUISITES.md is complete.
	AuthorODSCode string
	AuthorName    string
	DocumentTitle string
	// PDFBase64 is the already-base64-encoded PDF produced by
	// internal/pdfgen — kept as a string here rather than []byte so the
	// caller can pass pdfgen's own output straight through.
	PDFBase64 string
	CreatedAt time.Time
}

// fhirBundle, fhirEntry etc. are a deliberately tiny, hand-rolled subset of
// the FHIR R4 JSON shape — just enough fields for this one message type.
// Reach for a real FHIR library (see docs/nhs/README.md) if this project
// ever needs to also read GP Connect Access Record or handle other
// resource types; a full R4 type system is out of scope for one message.
type fhirBundle struct {
	ResourceType string      `json:"resourceType"`
	Type         string      `json:"type"`
	Timestamp    string      `json:"timestamp"`
	Entry        []fhirEntry `json:"entry"`
}

type fhirEntry struct {
	FullURL  string `json:"fullUrl"`
	Resource any    `json:"resource"`
}

type fhirComposition struct {
	ResourceType string              `json:"resourceType"`
	ID           string              `json:"id"`
	Status       string              `json:"status"`
	Type         fhirCodeableConcept `json:"type"`
	Subject      fhirReference       `json:"subject"`
	Date         string              `json:"date"`
	Author       []fhirReference     `json:"author"`
	Title        string              `json:"title"`
	Section      []fhirSection       `json:"section"`
}

type fhirSection struct {
	Title string          `json:"title"`
	Entry []fhirReference `json:"entry"`
}

type fhirBinary struct {
	ResourceType string `json:"resourceType"`
	ID           string `json:"id"`
	ContentType  string `json:"contentType"`
	Data         string `json:"data"`
}

type fhirCodeableConcept struct {
	Coding []fhirCoding `json:"coding"`
	Text   string       `json:"text,omitempty"`
}

type fhirCoding struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display"`
}

type fhirReference struct {
	Reference  string          `json:"reference,omitempty"`
	Identifier *fhirIdentifier `json:"identifier,omitempty"`
	Display    string          `json:"display,omitempty"`
}

type fhirIdentifier struct {
	System string `json:"system"`
	Value  string `json:"value"`
}

// BuildSendDocumentBundle produces the FHIR Bundle JSON for one GP Connect:
// Send Document message. The caller (internal/nhs/mesh.go's SendDocument
// convenience wrapper, or a handler) is responsible for transporting the
// returned bytes as a MESH message body with the correct workflow ID
// (WorkflowGPFedConsultReport, see mesh.go).
func BuildSendDocumentBundle(in SendDocumentInput) ([]byte, error) {
	if in.Patient.NHSNumber == "" {
		return nil, fmt.Errorf("nhs: cannot build a Send Document message without a PDS-verified NHS number")
	}
	if _, err := ValidateNHSNumber(in.Patient.NHSNumber); err != nil {
		return nil, fmt.Errorf("nhs: patient NHS number is not valid: %w", err)
	}
	if in.PDFBase64 == "" {
		return nil, fmt.Errorf("nhs: PDFBase64 is required")
	}
	if _, err := base64.StdEncoding.DecodeString(in.PDFBase64); err != nil {
		return nil, fmt.Errorf("nhs: PDFBase64 is not valid base64: %w", err)
	}
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now().UTC()
	}

	compositionID := uuid.New().String()
	binaryID := uuid.New().String()
	patientRef := fhirReference{
		Identifier: &fhirIdentifier{
			// NHS number system URI — the standard FHIR way to reference a
			// patient by NHS number without a prior /Patient read.
			System: "https://fhir.nhs.uk/Id/nhs-number",
			Value:  in.Patient.NHSNumber,
		},
		Display: in.Patient.Name,
	}

	composition := fhirComposition{
		ResourceType: "Composition",
		ID:           compositionID,
		Status:       "final",
		Type: fhirCodeableConcept{
			// SNOMED CT "Consultation note" — placeholder pending
			// confirmation against GP Connect's mandated
			// Composition.type binding (needs verification).
			Coding: []fhirCoding{{
				System:  "http://snomed.info/sct",
				Code:    "371530004",
				Display: "Clinical consultation report",
			}},
		},
		Subject: patientRef,
		Date:    in.CreatedAt.Format(time.RFC3339),
		Author: []fhirReference{{
			Display: in.AuthorName,
			Identifier: &fhirIdentifier{
				System: "https://fhir.nhs.uk/Id/ods-organization-code",
				Value:  in.AuthorODSCode,
			},
		}},
		Title: in.DocumentTitle,
		Section: []fhirSection{{
			Title: "Consultation Document",
			Entry: []fhirReference{{Reference: "Binary/" + binaryID}},
		}},
	}

	binary := fhirBinary{
		ResourceType: "Binary",
		ID:           binaryID,
		ContentType:  "application/pdf",
		Data:         in.PDFBase64,
	}

	bundle := fhirBundle{
		ResourceType: "Bundle",
		Type:         "document",
		Timestamp:    in.CreatedAt.Format(time.RFC3339),
		Entry: []fhirEntry{
			{FullURL: "urn:uuid:" + compositionID, Resource: composition},
			{FullURL: "urn:uuid:" + binaryID, Resource: binary},
		},
	}

	return json.MarshalIndent(bundle, "", "  ")
}
