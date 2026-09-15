package handlers

import (
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"corti-backend/internal/pgstore"
)

// AuditHandler exposes the clinical audit trail (audit_log table) for
// review — requires audit.view, which only the superuser role has by
// default (see auth.DefaultRoles). This is a read-only surface: there is
// deliberately no way to edit or delete an audit event anywhere in the
// system (see pgstore/audit_store.go's doc comment).
type AuditHandler struct {
	store *pgstore.Store
}

func NewAuditHandler(store *pgstore.Store) *AuditHandler {
	return &AuditHandler{store: store}
}

// HandleListAuditEvents returns audit events, optionally filtered.
// GET /api/audit?patientId=&actorId=&limit=  (requires audit.view)
func (h *AuditHandler) HandleListAuditEvents(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit"))
	events, err := h.store.ListAuditEvents(c.Context(), pgstore.AuditEventFilter{
		PatientID: c.Query("patientId"),
		ActorID:   c.Query("actorId"),
		Limit:     limit,
	})
	if err != nil {
		log.Printf("audit_handler: list audit events: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list audit events"})
	}
	return c.JSON(events)
}
