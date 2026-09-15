package handlers

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"

	"corti-backend/internal/auth"
	"corti-backend/internal/nhs"
	"corti-backend/internal/pdfgen"
	"corti-backend/internal/pgstore"
)

// NHSHandler exposes the two SystmOne-facing actions this project actually
// needs (see SYSTMONE_INTEGRATION_REPORT.md §3-4): confirming a patient's
// identity against PDS, and sending a generated clinical document to their
// registered GP practice via GP Connect: Send Document over MESH. Every
// route requires nhs.integration. Both PDS calls and Send Document sends
// will fail with a clear error against the sandbox/no configured
// credentials — see internal/nhs/pds.go and internal/nhs/mesh.go — until
// real NHS-issued credentials are configured (docs/regulatory/PREREQUISITES.md).
type NHSHandler struct {
	sessionStore  auth.SessionStorage
	store         *pgstore.Store
	pds           *nhs.PDSClient
	mesh          *nhs.MESHClient
	authorODSCode string
	authorOrgName string
}

func NewNHSHandler(sessionStore auth.SessionStorage, store *pgstore.Store, pds *nhs.PDSClient, mesh *nhs.MESHClient, authorODSCode, authorOrgName string) *NHSHandler {
	return &NHSHandler{
		sessionStore:  sessionStore,
		store:         store,
		pds:           pds,
		mesh:          mesh,
		authorODSCode: authorODSCode,
		authorOrgName: authorOrgName,
	}
}

func (h *NHSHandler) audit(c *fiber.Ctx, action, resourceType, resourceID, patientID string, detail map[string]any) {
	claims := c.Locals("claims").(*auth.Claims)
	_ = h.store.WriteAuditEvent(c.Context(), pgstore.AuditEvent{
		ActorID:      claims.UserID,
		ActorName:    claims.Email,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		PatientID:    patientID,
		IPAddress:    c.IP(),
		Detail:       detail,
	})
}

// HandleTraceByNHSNumber confirms a patient's identity via PDS, given an
// NHS number a clinician already has (e.g. from a letter or the NHS App).
// POST /api/nhs/pds/trace  { "nhsNumber": "..." }  (requires nhs.integration)
func (h *NHSHandler) HandleTraceByNHSNumber(c *fiber.Ctx) error {
	var req struct {
		NHSNumber string `json:"nhsNumber"`
	}
	if err := c.BodyParser(&req); err != nil || req.NHSNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "nhsNumber is required"})
	}
	ref, err := h.pds.TraceByNHSNumber(c.Context(), req.NHSNumber)
	h.audit(c, pgstore.AuditActionPDSLookup, "pds_trace", req.NHSNumber, "", map[string]any{"found": err == nil})
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(ref)
}

// HandleSearchByDemographics is the fallback trace when the NHS number
// isn't known — see nhs.PDSClient.SearchByDemographics.
// POST /api/nhs/pds/search  { "familyName", "givenName", "birthDate", "postcode" }
func (h *NHSHandler) HandleSearchByDemographics(c *fiber.Ctx) error {
	var req struct {
		FamilyName string `json:"familyName"`
		GivenName  string `json:"givenName"`
		BirthDate  string `json:"birthDate"`
		Postcode   string `json:"postcode"`
	}
	if err := c.BodyParser(&req); err != nil || req.FamilyName == "" || req.BirthDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "familyName and birthDate are required"})
	}
	refs, err := h.pds.SearchByDemographics(c.Context(), req.FamilyName, req.GivenName, req.BirthDate, req.Postcode)
	h.audit(c, pgstore.AuditActionPDSLookup, "pds_search", req.FamilyName, "", map[string]any{"candidateCount": len(refs)})
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"candidates": refs})
}

// HandleVerifyPatientAgainstPDS runs TraceByNHSNumber for an existing
// structured patient record and, on success, stamps pds_verified_at —
// GP Connect: Send Document requires the sender to be "PDS compliant", and
// this is the concrete point where that compliance is evidenced per patient.
// POST /api/patients/:id/verify-pds  (requires nhs.integration)
func (h *NHSHandler) HandleVerifyPatientAgainstPDS(c *fiber.Ctx) error {
	id := c.Params("id")
	patient, err := h.store.GetPatientByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Patient not found"})
	}
	if patient.NHSNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Patient has no NHS number on file to verify"})
	}
	ref, err := h.pds.TraceByNHSNumber(c.Context(), patient.NHSNumber)
	h.audit(c, pgstore.AuditActionPDSLookup, "pds_verify", id, id, map[string]any{"verified": err == nil})
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "PDS could not confirm this identity: " + err.Error()})
	}
	now := time.Now().UTC()
	patient.PDSVerifiedAt = &now
	// A verified trace is the authoritative record of the patient's name —
	// take PDS's spelling over whatever was typed in manually.
	if ref.Name != "" {
		patient.Name = ref.Name
	}
	updated, err := h.store.UpdatePatient(c.Context(), patient)
	if err != nil {
		log.Printf("nhs_handler: record PDS verification for patient %s: %v", id, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to record PDS verification"})
	}
	return c.JSON(updated)
}

// HandleSendToGP is GP Connect: Send Document end to end for one saved
// session: builds a PDF from the session's generated document, wraps it in
// the FHIR Bundle BuildSendDocumentBundle expects, and transmits it over
// MESH using the GPFED_CONSULT_REPORT workflow. Refuses to proceed unless
// the linked patient has a PDS-verified NHS number — sending a clinical
// document addressed to an unverified identity is exactly the mismatch
// this whole pipeline exists to prevent.
// POST /api/nhs/sessions/:id/send-to-gp  { "recipientMailboxId": "..." }  (requires nhs.integration)
func (h *NHSHandler) HandleSendToGP(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)
	sessionID := c.Params("id")
	var req struct {
		RecipientMailboxID string `json:"recipientMailboxId"`
	}
	if err := c.BodyParser(&req); err != nil || req.RecipientMailboxID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "recipientMailboxId is required — the destination practice's MESH mailbox ID"})
	}

	session, err := h.sessionStore.GetSessionByID(sessionID, claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Session not found"})
	}
	if session.Document == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "This session has no generated document to send yet — generate one first"})
	}

	patient, err := h.store.GetSessionPatient(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	patientID := patient.ID
	if patient.PDSVerifiedAt == nil {
		return c.Status(fiber.StatusPreconditionFailed).JSON(fiber.Map{
			"error": "This patient has not been PDS-verified — call POST /api/patients/:id/verify-pds first, then retry",
		})
	}

	letter := pdfgen.Letter{
		Title:  session.Document.TemplateName,
		Byline: "Patient: " + patient.Name + " (NHS " + nhs.FormatNHSNumber(patient.NHSNumber) + ")  ·  Generated " + time.Now().UTC().Format("2006-01-02"),
	}
	for _, sec := range session.Document.Sections {
		body := sec.Text
		if body == "" {
			body = sec.Content
		}
		letter.Sections = append(letter.Sections, pdfgen.Section{Heading: sec.Name, Body: body})
	}
	pdfBase64 := pdfgen.RenderBase64(letter)

	bundle, err := nhs.BuildSendDocumentBundle(nhs.SendDocumentInput{
		Patient: nhs.FHIRPatientRef{
			NHSNumber: patient.NHSNumber,
			Name:      patient.Name,
			BirthDate: patient.DateOfBirth,
		},
		AuthorODSCode: h.authorODSCode,
		AuthorName:    h.authorOrgName,
		DocumentTitle: session.Document.TemplateName,
		PDFBase64:     pdfBase64,
	})
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}

	messageID, err := h.mesh.SendMessage(c.Context(), req.RecipientMailboxID, nhs.WorkflowGPFedConsultReport, sessionID+".json", bundle)
	h.audit(c, pgstore.AuditActionSendToGP, "session", sessionID, patientID, map[string]any{
		"recipientMailboxId": req.RecipientMailboxID,
		"success":            err == nil,
	})
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "meshMessageId": messageID})
}
