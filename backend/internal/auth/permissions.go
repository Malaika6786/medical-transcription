package auth

import "fmt"

// Permission is the atomic access unit in the RBAC system.
type Permission string

const (
	PermAmbientAccess     Permission = "ambient.access"
	PermFileTranscription Permission = "file_transcription.access"
	PermDictation         Permission = "dictation.access"
	PermClinicalFacts     Permission = "clinical_facts.view"
	PermUsersManage       Permission = "users.manage"
	PermDocumentationView Permission = "documentation.view"
	PermEmbeddedAssistant Permission = "embedded_assistant.access"
	PermTemplatesManage   Permission = "templates.manage"
	PermCortiSectionsView Permission = "corti_sections.view"

	// NHS/SystmOne integration permissions (SYSTMONE_INTEGRATION_REPORT.md).
	PermPatientsManage Permission = "patients.manage" // create/view/edit structured (NHS-number-based) patient records
	PermNHSIntegration Permission = "nhs.integration" // PDS lookup + GP Connect: Send Document
	PermAuditView      Permission = "audit.view"      // read the clinical audit trail
)

// AllPermissions is the complete list of permissions defined in the system.
// Used by the API to expose available permissions for role/user management.
var AllPermissions = []Permission{
	PermAmbientAccess,
	PermFileTranscription,
	PermDictation,
	PermClinicalFacts,
	PermUsersManage,
	PermDocumentationView,
	PermEmbeddedAssistant,
	PermTemplatesManage,
	PermCortiSectionsView,
	PermPatientsManage,
	PermNHSIntegration,
	PermAuditView,
}

// ValidatePermissions rejects any permission not in AllPermissions.
func ValidatePermissions(perms []Permission) error {
	valid := make(map[Permission]bool, len(AllPermissions))
	for _, p := range AllPermissions {
		valid[p] = true
	}
	for _, p := range perms {
		if !valid[p] {
			return fmt.Errorf("%w: %s", ErrInvalidPerm, p)
		}
	}
	return nil
}
