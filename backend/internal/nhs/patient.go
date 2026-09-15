package nhs

import "time"

// Patient is the structured patient-identity record this project did not
// have before (see SYSTMONE_INTEGRATION_REPORT.md, "No proper patient
// identity"/"No NHS number"). It exists alongside, not instead of, the
// pre-existing free-text patientName field used by general (non-NHS)
// document generation — see backend/internal/handlers/async_handler.go.
//
// NHSNumber/Name/DateOfBirth are plaintext here (the in-memory/API shape);
// internal/pgstore/patient_store.go is responsible for encrypting them with
// internal/cryptofield before they ever reach a SQL statement, and
// decrypting them on the way back out. Nothing outside pgstore should ever
// see ciphertext.
type Patient struct {
	ID string `json:"id"`

	// NHSNumber is empty until either PDS-verified (LookupByNHSNumber
	// succeeded) or manually entered and passed ValidateNHSNumber. Always
	// store/compare the normalized (digits-only) form.
	NHSNumber string `json:"nhsNumber,omitempty"`
	Name      string `json:"name"`
	// DateOfBirth is an ISO-8601 date ("2026-09-07"), not a full timestamp —
	// FHIR's Patient.birthDate is a date, not a dateTime.
	DateOfBirth string `json:"dateOfBirth,omitempty"`
	// Sex is a FHIR AdministrativeGender code: male | female | other | unknown.
	Sex string `json:"sex"`

	// PDSVerifiedAt is non-nil once a PDS trace has confirmed this identity
	// against the national demographics record — a document should not be
	// sent to a GP practice for a patient who has never cleared this check,
	// per GP Connect: Send Document's requirement to be "PDS compliant".
	PDSVerifiedAt *time.Time `json:"pdsVerifiedAt,omitempty"`

	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ValidAdministrativeGenders is the FHIR-defined value set for
// Patient.gender — anything else is rejected by CreatePatient/UpdatePatient.
var ValidAdministrativeGenders = map[string]bool{
	"male":    true,
	"female":  true,
	"other":   true,
	"unknown": true,
}
