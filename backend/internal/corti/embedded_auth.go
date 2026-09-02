package corti

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"corti-backend/internal/utils"
)

// EmbeddedTokenCache manages OAuth2 tokens for the Embedded Assistant via ROPC.
// A single shared token is cached and refreshed for all demo users.
type EmbeddedTokenCache struct {
	clientID    string
	username    string
	password    string
	authURL     string
	httpClient  *http.Client

	mu          sync.RWMutex
	cachedToken *CachedEmbeddedToken
}

// CachedEmbeddedToken holds all three tokens required by the Embedded Assistant auth() call.
type CachedEmbeddedToken struct {
	AccessToken        string
	RefreshToken       string
	IDToken            string
	AccessExpiresAt    time.Time
	RefreshExpiresAt   time.Time
}

// embeddedTokenResponse maps the Keycloak ROPC/refresh response fields.
type embeddedTokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	IDToken          string `json:"id_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
}

// NewEmbeddedTokenCache creates a new cache using credentials from config.
func NewEmbeddedTokenCache(config *utils.Config) *EmbeddedTokenCache {
	return &EmbeddedTokenCache{
		clientID:   config.CortiEmbeddedClientID,
		username:   config.CortiEmbeddedUsername,
		password:   config.CortiEmbeddedPassword,
		authURL:    config.CortiAuthURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetToken returns valid embedded tokens, refreshing proactively 60 seconds before expiry.
// Tries refresh_token first; falls back to full ROPC if refresh has expired.
func (e *EmbeddedTokenCache) GetToken() (*CachedEmbeddedToken, error) {
	const refreshBuffer = 60 * time.Second

	// Fast path — read lock
	e.mu.RLock()
	if e.cachedToken != nil && time.Now().Add(refreshBuffer).Before(e.cachedToken.AccessExpiresAt) {
		t := e.cachedToken
		e.mu.RUnlock()
		return t, nil
	}
	e.mu.RUnlock()

	// Slow path — write lock
	e.mu.Lock()
	defer e.mu.Unlock()

	// Double-check after acquiring write lock
	if e.cachedToken != nil && time.Now().Add(refreshBuffer).Before(e.cachedToken.AccessExpiresAt) {
		return e.cachedToken, nil
	}

	// Try refresh_token if we have one and it's still valid
	if e.cachedToken != nil && time.Now().Before(e.cachedToken.RefreshExpiresAt) {
		log.Println("EmbeddedAuth: refreshing access token via refresh_token")
		t, err := e.fetchViaRefresh(e.cachedToken.RefreshToken)
		if err == nil {
			e.cachedToken = t
			return t, nil
		}
		log.Printf("EmbeddedAuth: refresh failed, falling back to ROPC: %v", err)
	}

	// Full ROPC exchange
	log.Println("EmbeddedAuth: fetching tokens via ROPC")
	t, err := e.fetchViaROPC()
	if err != nil {
		return nil, err
	}
	e.cachedToken = t
	return t, nil
}

// fetchViaROPC performs a Resource Owner Password Credentials exchange.
// scope=openid is mandatory — without it Keycloak does not return id_token.
func (e *EmbeddedTokenCache) fetchViaROPC() (*CachedEmbeddedToken, error) {
	if e.clientID == "" || e.username == "" || e.password == "" {
		return nil, errors.New("embedded auth: missing CORTI_EMBEDDED_CLIENT_ID, CORTI_EMBEDDED_USERNAME, or CORTI_EMBEDDED_PASSWORD")
	}

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", e.clientID)
	form.Set("username", e.username)
	form.Set("password", e.password)
	form.Set("scope", "openid profile email")

	return e.doTokenRequest(form)
}

// fetchViaRefresh exchanges a refresh_token for a new access_token.
func (e *EmbeddedTokenCache) fetchViaRefresh(refreshToken string) (*CachedEmbeddedToken, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", e.clientID)
	form.Set("refresh_token", refreshToken)
	form.Set("scope", "openid profile email")

	return e.doTokenRequest(form)
}

// doTokenRequest sends a form-urlencoded POST to the Keycloak token endpoint.
func (e *EmbeddedTokenCache) doTokenRequest(form url.Values) (*CachedEmbeddedToken, error) {
	req, err := http.NewRequest("POST", e.authURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("embedded auth: failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedded auth: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("embedded auth: failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedded auth: token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tr embeddedTokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("embedded auth: failed to parse response: %w", err)
	}

	if tr.AccessToken == "" {
		return nil, errors.New("embedded auth: empty access_token in response")
	}
	if tr.IDToken == "" {
		return nil, errors.New("embedded auth: empty id_token — ensure scope=openid is included")
	}

	now := time.Now()
	refreshExpiry := now.Add(30 * 24 * time.Hour) // default 30 days
	if tr.RefreshExpiresIn > 0 {
		refreshExpiry = now.Add(time.Duration(tr.RefreshExpiresIn) * time.Second)
	}

	log.Printf("EmbeddedAuth: tokens obtained, access expires in %ds", tr.ExpiresIn)

	return &CachedEmbeddedToken{
		AccessToken:      tr.AccessToken,
		RefreshToken:     tr.RefreshToken,
		IDToken:          tr.IDToken,
		AccessExpiresAt:  now.Add(time.Duration(tr.ExpiresIn) * time.Second),
		RefreshExpiresAt: refreshExpiry,
	}, nil
}
