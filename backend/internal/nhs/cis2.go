package nhs

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
)

// CIS2 (Care Identity Service 2) is how NHS England federates clinician
// identity across national and local systems, via standard OpenID Connect
// (Authorization Code + PKCE) — see SYSTMONE_INTEGRATION_REPORT.md §5. This
// project's own JWT-based login (internal/auth/jwt.go) is unaffected and
// keeps working for normal app accounts; CIS2 would only be needed if/when
// a specific NHS API this project calls requires a user-restricted
// (clinician-identity-bound) access token rather than a system-level
// credential — GP Connect: Send Document and MESH, the routes this project
// actually needs today, are both system-to-system and do NOT require CIS2.
// This scaffold exists for that future case, and is not wired into any
// route yet.
//
// CIS2Config has no working default — IssuerURL/ClientID/ClientSecret/
// RedirectURL are all issued when an application registers with NHS
// Digital's identity team (needs verification of the current onboarding
// process and discovery endpoint).
type CIS2Config struct {
	IssuerURL    string // e.g. the CIS2 OIDC discovery issuer
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// PKCEPair is one Authorization Code + PKCE challenge/verifier pair. Store
// Verifier server-side (session/state), keyed by State, until the callback
// arrives — never send Verifier to the browser.
type PKCEPair struct {
	Verifier  string
	Challenge string // S256(Verifier), base64url, no padding
	State     string
}

// NewPKCEPair generates a fresh verifier/challenge/state triple for one
// login attempt, per RFC 7636.
func NewPKCEPair() (*PKCEPair, error) {
	verifier, err := randomURLSafeString(64)
	if err != nil {
		return nil, err
	}
	state, err := randomURLSafeString(32)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return &PKCEPair{Verifier: verifier, Challenge: challenge, State: state}, nil
}

// BuildAuthorizationURL returns the URL to redirect the clinician's browser
// to for CIS2 sign-in. authorizeEndpoint comes from the issuer's OIDC
// discovery document (GET {IssuerURL}/.well-known/openid-configuration) —
// fetching and caching that document is intentionally left to the caller
// rather than hardcoded here, since the exact discovery shape needs
// verification against CIS2's real metadata.
func BuildAuthorizationURL(cfg CIS2Config, authorizeEndpoint string, pkce *PKCEPair, scopes []string) string {
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", cfg.RedirectURL)
	q.Set("scope", joinScopes(scopes))
	q.Set("state", pkce.State)
	q.Set("code_challenge", pkce.Challenge)
	q.Set("code_challenge_method", "S256")
	return authorizeEndpoint + "?" + q.Encode()
}

// TokenExchangeRequest is what BuildTokenExchangeForm needs to prepare the
// token-endpoint POST body for the Authorization Code + PKCE grant. The
// actual HTTP call is left to the caller (a small, ordinary
// application/x-www-form-urlencoded POST to the issuer's token_endpoint) —
// not implemented here since it needs a live discovery document to target,
// same reasoning as BuildAuthorizationURL.
type TokenExchangeRequest struct {
	Code         string
	CodeVerifier string
}

// BuildTokenExchangeForm returns the form-encoded body for the token
// exchange POST.
func BuildTokenExchangeForm(cfg CIS2Config, req TokenExchangeRequest) url.Values {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", req.Code)
	form.Set("redirect_uri", cfg.RedirectURL)
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("code_verifier", req.CodeVerifier)
	return form
}

func joinScopes(scopes []string) string {
	out := ""
	for i, s := range scopes {
		if i > 0 {
			out += " "
		}
		out += s
	}
	return out
}

func randomURLSafeString(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("nhs: generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
