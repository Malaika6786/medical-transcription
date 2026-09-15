package pgstore

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"corti-backend/internal/nhs"
)

// ErrPatientNotFound is returned when a patient ID has no matching row.
var ErrPatientNotFound = errors.New("patient not found")

// ErrNHSNumberInUse is returned when creating/updating a patient with an
// NHS number that already belongs to a different patient row.
var ErrNHSNumberInUse = errors.New("this NHS number is already registered to another patient record")

const patientNHSNumberUniqueConstraint = "patients_nhs_number_hash_key"

// requireCipher panics with a clear message if the server was booted
// without FIELD_ENCRYPTION_KEY — patients must never be written or read
// without encryption, so this is a programming-error-level guard, not a
// recoverable condition (main.go's startup check should have refused to
// register the patient routes at all in that case; see cmd/server/main.go).
func (s *Store) requireCipher() {
	if s.cipher == nil {
		panic("pgstore: patient operation attempted without FIELD_ENCRYPTION_KEY configured — this should be unreachable, see main.go's startup guard")
	}
}

func (s *Store) scanPatient(row pgx.Row) (*nhs.Patient, error) {
	s.requireCipher()
	var (
		p                                       nhs.Patient
		nhsNumberEnc, nameEnc, dobEnc           *string // nhs_number_enc/date_of_birth_enc are nullable columns — nil means "not on file", not an error
		pdsVerifiedAt                           *time.Time
	)
	err := row.Scan(&p.ID, &nhsNumberEnc, &nameEnc, &dobEnc, &p.Sex, &pdsVerifiedAt, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPatientNotFound
	}
	if err != nil {
		return nil, err
	}
	if p.Name, err = s.cipher.Decrypt(deref(nameEnc)); err != nil {
		return nil, fmt.Errorf("decrypt patient name: %w", err)
	}
	if p.NHSNumber, err = s.cipher.Decrypt(deref(nhsNumberEnc)); err != nil {
		return nil, fmt.Errorf("decrypt patient NHS number: %w", err)
	}
	if p.DateOfBirth, err = s.cipher.Decrypt(deref(dobEnc)); err != nil {
		return nil, fmt.Errorf("decrypt patient date of birth: %w", err)
	}
	p.PDSVerifiedAt = pdsVerifiedAt
	return &p, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

const patientColumns = `id, nhs_number_enc, name_enc, date_of_birth_enc, sex, pds_verified_at, created_by, created_at, updated_at`

// CreatePatient encrypts and stores a new patient record. p.ID, p.CreatedAt,
// p.UpdatedAt are assigned here; the caller should not set them.
func (s *Store) CreatePatient(ctx context.Context, p *nhs.Patient) (*nhs.Patient, error) {
	s.requireCipher()
	if p.Sex == "" {
		p.Sex = "unknown"
	}
	if !nhs.ValidAdministrativeGenders[p.Sex] {
		return nil, fmt.Errorf("invalid sex %q: must be one of male/female/other/unknown", p.Sex)
	}
	nameEnc, err := s.cipher.Encrypt(p.Name)
	if err != nil {
		return nil, err
	}
	nhsNumberEnc, err := s.cipher.Encrypt(p.NHSNumber)
	if err != nil {
		return nil, err
	}
	dobEnc, err := s.cipher.Encrypt(p.DateOfBirth)
	if err != nil {
		return nil, err
	}
	var nhsHash *string
	if p.NHSNumber != "" {
		h := s.cipher.LookupHash(p.NHSNumber)
		nhsHash = &h
	}

	id := "pt_" + newRandomID()
	now := time.Now().UTC()
	_, err = s.pool.Exec(ctx, `
		INSERT INTO patients (id, nhs_number_enc, nhs_number_hash, name_enc, date_of_birth_enc, sex, pds_verified_at, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)`,
		id, nullIfEmpty(nhsNumberEnc), nhsHash, nameEnc, nullIfEmpty(dobEnc), p.Sex, p.PDSVerifiedAt, p.CreatedBy, now)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation && pgErr.ConstraintName == patientNHSNumberUniqueConstraint {
			return nil, ErrNHSNumberInUse
		}
		return nil, err
	}
	return s.GetPatientByID(ctx, id)
}

// GetPatientByID returns one patient, decrypted.
func (s *Store) GetPatientByID(ctx context.Context, id string) (*nhs.Patient, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+patientColumns+` FROM patients WHERE id = $1`, id)
	return s.scanPatient(row)
}

// FindPatientByNHSNumber looks up a patient by NHS number without ever
// decrypting other rows — it matches on nhs_number_hash, a deterministic
// HMAC computed the same way at write time (see CreatePatient).
func (s *Store) FindPatientByNHSNumber(ctx context.Context, nhsNumber string) (*nhs.Patient, error) {
	s.requireCipher()
	hash := s.cipher.LookupHash(nhsNumber)
	row := s.pool.QueryRow(ctx, `SELECT `+patientColumns+` FROM patients WHERE nhs_number_hash = $1`, hash)
	return s.scanPatient(row)
}

// ListPatients returns every patient record, newest first. There is no
// pagination yet — fine at the project's current scale, revisit if this
// table grows large.
func (s *Store) ListPatients(ctx context.Context) ([]*nhs.Patient, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+patientColumns+` FROM patients ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	patients := []*nhs.Patient{}
	for rows.Next() {
		p, err := s.scanPatient(rows)
		if err != nil {
			log.Printf("pgstore: scan patient: %v", err)
			continue
		}
		patients = append(patients, p)
	}
	return patients, rows.Err()
}

// UpdatePatient re-encrypts and replaces the mutable fields of a patient
// record (name, date of birth, sex, NHS number, PDS verification stamp).
func (s *Store) UpdatePatient(ctx context.Context, p *nhs.Patient) (*nhs.Patient, error) {
	s.requireCipher()
	if !nhs.ValidAdministrativeGenders[p.Sex] {
		return nil, fmt.Errorf("invalid sex %q: must be one of male/female/other/unknown", p.Sex)
	}
	nameEnc, err := s.cipher.Encrypt(p.Name)
	if err != nil {
		return nil, err
	}
	nhsNumberEnc, err := s.cipher.Encrypt(p.NHSNumber)
	if err != nil {
		return nil, err
	}
	dobEnc, err := s.cipher.Encrypt(p.DateOfBirth)
	if err != nil {
		return nil, err
	}
	var nhsHash *string
	if p.NHSNumber != "" {
		h := s.cipher.LookupHash(p.NHSNumber)
		nhsHash = &h
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE patients SET
			nhs_number_enc = $2, nhs_number_hash = $3, name_enc = $4, date_of_birth_enc = $5,
			sex = $6, pds_verified_at = $7, updated_at = $8
		WHERE id = $1`,
		p.ID, nullIfEmpty(nhsNumberEnc), nhsHash, nameEnc, nullIfEmpty(dobEnc), p.Sex, p.PDSVerifiedAt, time.Now().UTC())
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation && pgErr.ConstraintName == patientNHSNumberUniqueConstraint {
			return nil, ErrNHSNumberInUse
		}
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrPatientNotFound
	}
	return s.GetPatientByID(ctx, p.ID)
}

// DeletePatient removes a patient record. Sessions referencing it have
// patient_id set to NULL (ON DELETE SET NULL — see schema.sql) rather than
// being deleted themselves.
func (s *Store) DeletePatient(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM patients WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPatientNotFound
	}
	return nil
}

// GetSessionPatient returns the patient linked to a session via
// sessions.patient_id (see LinkSessionToPatient), or an error if the
// session has no linked patient — every NHS Send Document flow needs a
// structured, PDS-verifiable identity, so "no patient linked" is treated
// as a caller error here rather than returning (nil, nil).
func (s *Store) GetSessionPatient(ctx context.Context, sessionID string) (*nhs.Patient, error) {
	var patientID *string
	err := s.pool.QueryRow(ctx, `SELECT patient_id FROM sessions WHERE id = $1`, sessionID).Scan(&patientID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("session not found")
	}
	if err != nil {
		return nil, err
	}
	if patientID == nil {
		return nil, errors.New("this session has no linked patient — call PUT /api/sessions/:id/patient first")
	}
	return s.GetPatientByID(ctx, *patientID)
}

// LinkSessionToPatient sets sessions.patient_id, scoped to the owning user
// (mirrors the ownership check every other session mutation already does).
func (s *Store) LinkSessionToPatient(ctx context.Context, sessionID, userID, patientID string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE sessions SET patient_id = $3, updated_at = now() WHERE id = $1 AND user_id = $2`,
		sessionID, userID, patientID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("session not found")
	}
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
