package nhs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

// PDSClient talks to NHS England's Personal Demographics Service FHIR API —
// the national record of every NHS patient's demographics (name, DOB,
// address, NHS number, registered GP). A document cannot be reliably
// addressed to a patient's SystmOne record without first confirming their
// identity here (GP Connect: Send Document requires the sender to be, in
// NHS Digital's words, "PDS compliant or capable of performing a PDS
// search" — see SYSTMONE_INTEGRATION_REPORT.md §5).
//
// This client cannot reach the real PDS without NHS-issued credentials
// (an API key registered on the NHS API platform, and — for anything
// beyond the sandbox — an approved onboarding application). BaseURL
// defaults to the public PDS FHIR sandbox
// (https://sandbox.api.service.nhs.uk/personal-demographics/FHIR/R4),
// which serves canned synthetic patients and needs no real credentials —
// useful for testing this client's request/response handling, NOT for
// looking up real patients. Swap PDS_BASE_URL to the integration or
// production PDS endpoint (needs verification of the exact current URL and
// the auth header PDS now expects — API-key vs. NHS CIS2 bearer token —
// against NHS Digital's live API catalogue entry) once real access is
// granted.
type PDSClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewPDSClient builds a client. baseURL/apiKey normally come from
// utils.Config (PDS_BASE_URL / PDS_API_KEY env vars).
func NewPDSClient(baseURL, apiKey string) *PDSClient {
	return &PDSClient{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// pdsPatientResource is the minimal subset of a FHIR Patient resource this
// project actually needs back from PDS (name, birth date, the NHS number
// PDS confirms, and the identifier system tells us it's genuinely the PDS
// "NHS number" identifier, not some other identifier type).
type pdsPatientResource struct {
	ResourceType string `json:"resourceType"`
	ID           string `json:"id"` // PDS echoes the NHS number back as the resource id
	Identifier   []struct {
		System string `json:"system"`
		Value  string `json:"value"`
	} `json:"identifier"`
	Name []struct {
		Use    string   `json:"use"`
		Family string   `json:"family"`
		Given  []string `json:"given"`
	} `json:"name"`
	BirthDate string `json:"birthDate"`
}

// pdsBundle wraps the multiple-candidate response PDS returns from a
// demographic (name+DOB+postcode) search, as opposed to a direct
// by-NHS-number GET which returns a bare Patient resource.
type pdsBundle struct {
	ResourceType string `json:"resourceType"`
	Entry        []struct {
		Resource pdsPatientResource `json:"resource"`
	} `json:"entry"`
}

func (r pdsPatientResource) toFHIRPatientRef() FHIRPatientRef {
	name := ""
	for _, n := range r.Name {
		if n.Use == "official" || name == "" {
			given := ""
			for i, g := range n.Given {
				if i > 0 {
					given += " "
				}
				given += g
			}
			name = given
			if n.Family != "" {
				if name != "" {
					name += " "
				}
				name += n.Family
			}
		}
	}
	return FHIRPatientRef{NHSNumber: r.ID, Name: name, BirthDate: r.BirthDate}
}

// TraceByNHSNumber confirms a specific NHS number resolves to a real PDS
// record ("GET /Patient/{nhsNumber}" — the most reliable form of matching,
// used once a patient has already given you their NHS number, e.g. from a
// letter or their NHS App).
func (c *PDSClient) TraceByNHSNumber(ctx context.Context, nhsNumber string) (*FHIRPatientRef, error) {
	normalized, err := ValidateNHSNumber(nhsNumber)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/Patient/"+normalized, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nhs: PDS request failed (is PDS_BASE_URL reachable and PDS_API_KEY valid?): %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("nhs: no PDS record found for NHS number %s", FormatNHSNumber(normalized))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nhs: PDS returned %d: %s", resp.StatusCode, truncate(string(body), 500))
	}
	var patient pdsPatientResource
	if err := json.Unmarshal(body, &patient); err != nil {
		return nil, fmt.Errorf("nhs: could not parse PDS response: %w", err)
	}
	ref := patient.toFHIRPatientRef()
	return &ref, nil
}

// SearchByDemographics performs the fallback trace when the NHS number
// isn't yet known — matching on name, date of birth, and postcode. PDS may
// return zero, one, or several plausible candidates; the caller (a
// clinician confirming identity, not an automated process) must pick the
// right one. familyName/givenName/birthDate("YYYY-MM-DD")/postcode are all
// required by PDS's own matching rules for an unauthenticated-identity
// trace search.
func (c *PDSClient) SearchByDemographics(ctx context.Context, familyName, givenName, birthDate, postcode string) ([]FHIRPatientRef, error) {
	q := url.Values{}
	q.Set("family", familyName)
	q.Set("given", givenName)
	q.Set("birthdate", "eq"+birthDate)
	q.Set("address-postalcode", postcode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/Patient?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nhs: PDS search request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nhs: PDS search returned %d: %s", resp.StatusCode, truncate(string(body), 500))
	}
	var bundle pdsBundle
	if err := json.Unmarshal(body, &bundle); err != nil {
		return nil, fmt.Errorf("nhs: could not parse PDS search response: %w", err)
	}
	refs := make([]FHIRPatientRef, 0, len(bundle.Entry))
	for _, e := range bundle.Entry {
		refs = append(refs, e.Resource.toFHIRPatientRef())
	}
	return refs, nil
}

func (c *PDSClient) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/fhir+json")
	// Every NHS API platform request requires a unique X-Request-ID
	// (confirmed live against the PDS sandbox: it 400s with
	// "X-Request-ID header must be supplied" without one) — a UUID per
	// request, used for cross-system request tracing.
	req.Header.Set("X-Request-ID", uuid.New().String())
	// PDS's real (non-sandbox) auth is CIS2/application-restricted OAuth2,
	// not a bare API key header — this apikey header is the sandbox's
	// scheme only. Swap for a bearer token from internal/nhs/cis2.go's
	// client-credentials flow once moving past the sandbox (needs
	// verification against the current PDS FHIR API catalogue entry).
	if c.APIKey != "" {
		req.Header.Set("apikey", c.APIKey)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
