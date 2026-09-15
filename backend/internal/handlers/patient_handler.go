package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"corti-backend/internal/auth"
	"corti-backend/internal/nhs"
	"corti-backend/internal/pgstore"
)

// PatientHandler implements CRUD for the structured, NHS-number-based
// patient record introduced alongside the NHS/SystmOne integration work —
// see SYSTMONE_INTEGRATION_REPORT.md, "No proper patient identity". Every
// route here requires patients.manage. All patient-touching actions are
// also written to the audit trail (internal/pgstore/audit_store.go).
type PatientHandler struct {
	store *pgstore.Store
}

func NewPatientHandler(store *pgstore.Store) *PatientHandler {
	return &PatientHandler{store: store}
}

type patientRequest struct {
	NHSNumber   string `json:"nhsNumber,omitempty"`
	Name        string `json:"name"`
	DateOfBirth string `json:"dateOfBirth,omitempty"`
	Sex         string `json:"sex,omitempty"`
}

func (h *PatientHandler) audit(c *fiber.Ctx, action, resourceID, patientID string) {
	claims := c.Locals("claims").(*auth.Claims)
	_ = h.store.WriteAuditEvent(c.Context(), pgstore.AuditEvent{
		ActorID:      claims.UserID,
		ActorName:    claims.Email,
		Action:       action,
		ResourceType: "patient",
		ResourceID:   resourceID,
		PatientID:    patientID,
		IPAddress:    c.IP(),
	})
}

// HandleCreatePatient creates a new structured patient record.
// POST /api/patients  (requires patients.manage)
func (h *PatientHandler) HandleCreatePatient(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)
	var req patientRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}
	normalizedNHS := ""
	if req.NHSNumber != "" {
		normalized, err := nhs.ValidateNHSNumber(req.NHSNumber)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		normalizedNHS = normalized
	}
	sex := req.Sex
	if sex == "" {
		sex = "unknown"
	}

	p := &nhs.Patient{
		NHSNumber:   normalizedNHS,
		Name:        req.Name,
		DateOfBirth: req.DateOfBirth,
		Sex:         sex,
		CreatedBy:   claims.UserID,
	}
	created, err := h.store.CreatePatient(c.Context(), p)
	if err != nil {
		if err == pgstore.ErrNHSNumberInUse {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		log.Printf("patient_handler: create patient: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create patient"})
	}
	h.audit(c, pgstore.AuditActionPatientCreate, created.ID, created.ID)
	return c.Status(fiber.StatusCreated).JSON(created)
}

// HandleListPatients returns every patient record.
// GET /api/patients  (requires patients.manage)
func (h *PatientHandler) HandleListPatients(c *fiber.Ctx) error {
	patients, err := h.store.ListPatients(c.Context())
	if err != nil {
		log.Printf("patient_handler: list patients: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list patients"})
	}
	return c.JSON(patients)
}

// HandleGetPatient returns one patient record, and logs a patient.view
// audit event — viewing a specific patient's identity data is exactly the
// kind of access the clinical audit trail exists to record.
// GET /api/patients/:id  (requires patients.manage)
func (h *PatientHandler) HandleGetPatient(c *fiber.Ctx) error {
	id := c.Params("id")
	p, err := h.store.GetPatientByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Patient not found"})
	}
	h.audit(c, pgstore.AuditActionPatientView, id, id)
	return c.JSON(p)
}

// HandleUpdatePatient replaces the mutable fields of a patient record.
// PUT /api/patients/:id  (requires patients.manage)
func (h *PatientHandler) HandleUpdatePatient(c *fiber.Ctx) error {
	id := c.Params("id")
	existing, err := h.store.GetPatientByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Patient not found"})
	}
	var req patientRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.DateOfBirth != "" {
		existing.DateOfBirth = req.DateOfBirth
	}
	if req.Sex != "" {
		existing.Sex = req.Sex
	}
	if req.NHSNumber != "" {
		normalized, err := nhs.ValidateNHSNumber(req.NHSNumber)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if normalized != existing.NHSNumber {
			existing.NHSNumber = normalized
			existing.PDSVerifiedAt = nil // NHS number changed manually — any prior PDS verification no longer applies
		}
	}
	updated, err := h.store.UpdatePatient(c.Context(), existing)
	if err != nil {
		if err == pgstore.ErrNHSNumberInUse {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		log.Printf("patient_handler: update patient %s: %v", id, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update patient"})
	}
	h.audit(c, pgstore.AuditActionPatientUpdate, id, id)
	return c.JSON(updated)
}

// HandleDeletePatient removes a patient record.
// DELETE /api/patients/:id  (requires patients.manage)
func (h *PatientHandler) HandleDeletePatient(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.store.DeletePatient(c.Context(), id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Patient not found"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// HandleLinkSession attaches a structured patient record to a saved
// session, replacing/augmenting that session's free-text patient name for
// NHS-integration purposes (see internal/nhs.Patient's doc comment).
// PUT /api/sessions/:id/patient  (requires patients.manage)
func (h *PatientHandler) HandleLinkSession(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)
	sessionID := c.Params("id")
	var body struct {
		PatientID string `json:"patientId"`
	}
	if err := c.BodyParser(&body); err != nil || body.PatientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "patientId is required"})
	}
	if _, err := h.store.GetPatientByID(c.Context(), body.PatientID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Patient not found"})
	}
	if err := h.store.LinkSessionToPatient(c.Context(), sessionID, claims.UserID, body.PatientID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Session not found"})
	}
	return c.JSON(fiber.Map{"success": true})
}
