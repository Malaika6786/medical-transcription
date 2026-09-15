package pgstore

import (
	"context"
	"encoding/json"
	"time"
)

// AuditEvent is one row of the append-only clinical audit trail
// (audit_log table, schema.sql) — "who did what to which patient's data,
// when", distinct from internal/middleware/logger.go's operational request
// log. See SYSTMONE_INTEGRATION_REPORT.md, "No proper clinical audit
// trail".
type AuditEvent struct {
	ID           int64          `json:"id"`
	OccurredAt   time.Time      `json:"occurredAt"`
	ActorID      string         `json:"actorId"`
	ActorName    string         `json:"actorName"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resourceType"`
	ResourceID   string         `json:"resourceId"`
	PatientID    string         `json:"patientId,omitempty"`
	IPAddress    string         `json:"ipAddress,omitempty"`
	Detail       map[string]any `json:"detail,omitempty"`
}

// Well-known audit actions. Handlers should use these constants rather than
// inline strings so ListAuditEvents filtering and any future reporting
// stays consistent.
const (
	AuditActionSessionView   = "session.view"
	AuditActionSessionExport = "session.export"
	AuditActionPatientView   = "patient.view"
	AuditActionPatientCreate = "patient.create"
	AuditActionPatientUpdate = "patient.update"
	AuditActionPDSLookup     = "nhs.pds_lookup"
	AuditActionSendToGP      = "nhs.send_to_gp"
)

// WriteAuditEvent appends one row. There is deliberately no corresponding
// Update/Delete method anywhere in this package — the table is append-only
// by convention (see schema.sql's comment on audit_log).
func (s *Store) WriteAuditEvent(ctx context.Context, e AuditEvent) error {
	var detail []byte
	if e.Detail != nil {
		var err error
		if detail, err = json.Marshal(e.Detail); err != nil {
			return err
		}
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_log (actor_id, actor_name, action, resource_type, resource_id, patient_id, ip_address, detail)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		e.ActorID, e.ActorName, e.Action, e.ResourceType, e.ResourceID, nullIfEmpty(e.PatientID), e.IPAddress, detail)
	return err
}

// AuditEventFilter narrows ListAuditEvents. Zero-value fields are ignored.
type AuditEventFilter struct {
	PatientID string
	ActorID   string
	Limit     int
}

// ListAuditEvents returns matching audit rows, newest first. Intended for
// the superuser-only audit.view screen — see internal/handlers/audit_handler.go.
func (s *Store) ListAuditEvents(ctx context.Context, filter AuditEventFilter) ([]AuditEvent, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	query := `SELECT id, occurred_at, actor_id, actor_name, action, resource_type, resource_id, COALESCE(patient_id, ''), ip_address, detail
		FROM audit_log WHERE ($1 = '' OR patient_id = $1) AND ($2 = '' OR actor_id = $2)
		ORDER BY occurred_at DESC LIMIT $3`
	rows, err := s.pool.Query(ctx, query, filter.PatientID, filter.ActorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []AuditEvent{}
	for rows.Next() {
		var e AuditEvent
		var detail []byte
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.ActorID, &e.ActorName, &e.Action, &e.ResourceType, &e.ResourceID, &e.PatientID, &e.IPAddress, &detail); err != nil {
			return nil, err
		}
		if len(detail) > 0 {
			_ = json.Unmarshal(detail, &e.Detail)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
