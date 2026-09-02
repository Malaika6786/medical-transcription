// Package corti provides client implementations for Corti API integration
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

// TokenManager handles OAuth2 token lifecycle management
// It provides thread-safe token caching with automatic refresh
type TokenManager struct {
	clientID      string
	clientSecret  string
	authURL       string
	audience      string
	refreshBuffer time.Duration
	httpClient    *http.Client

	mu          sync.RWMutex
	cachedToken *CachedToken
}

// NewTokenManager creates a new TokenManager instance
func NewTokenManager(config *utils.Config) *TokenManager {
	return &TokenManager{
		clientID:      config.CortiClientID,
		clientSecret:  config.CortiClientSecret,
		authURL:       config.CortiAuthURL,
		audience:      config.CortiAPIBaseURL,
		refreshBuffer: time.Duration(config.TokenRefreshBuffer) * time.Second,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetToken retrieves a valid access token, refreshing if necessary
// This method is thread-safe and handles concurrent requests
func (tm *TokenManager) GetToken() (string, error) {
	// First, try to get cached token with read lock
	tm.mu.RLock()
	if tm.cachedToken != nil && tm.isTokenValid() {
		token := tm.cachedToken.Token
		tm.mu.RUnlock()
		return token, nil
	}
	tm.mu.RUnlock()

	// Need to refresh - acquire write lock
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine may have refreshed)
	if tm.cachedToken != nil && tm.isTokenValid() {
		return tm.cachedToken.Token, nil
	}

	// Fetch new token
	return tm.fetchNewToken()
}

// isTokenValid checks if the cached token is still valid
// Must be called with at least a read lock held
func (tm *TokenManager) isTokenValid() bool {
	if tm.cachedToken == nil {
		return false
	}
	// Token is valid if it expires after now + refresh buffer
	return time.Now().Add(tm.refreshBuffer).Before(tm.cachedToken.ExpiresAt)
}

// fetchNewToken requests a new access token from the OAuth2 server
// Must be called with write lock held
func (tm *TokenManager) fetchNewToken() (string, error) {
	log.Println("Fetching new access token from Corti OAuth2 server")
	log.Printf("Auth URL: %s", tm.authURL)

	// Validate credentials
	if tm.clientID == "" || tm.clientSecret == "" {
		return "", errors.New("missing Corti client credentials")
	}
	if isPlaceholderCredential(tm.clientID) || isPlaceholderCredential(tm.clientSecret) {
		return "", errors.New("Corti client credentials are still set to example placeholder values")
	}

	// Prepare request body as form-urlencoded (required by Corti OAuth2)
	// Reference: https://docs.corti.ai/authentication/quickstart
	formData := url.Values{}
	formData.Set("client_id", tm.clientID)
	formData.Set("client_secret", tm.clientSecret)
	formData.Set("grant_type", "client_credentials")
	formData.Set("scope", "openid")

	// Create HTTP request with form-urlencoded body
	req, err := http.NewRequest("POST", tm.authURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// Execute request with retry logic
	var resp *http.Response
	var lastErr error
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err = tm.httpClient.Do(req)
		if err == nil {
			break
		}
		lastErr = err
		log.Printf("Token request attempt %d failed: %v", attempt, err)
		if attempt < maxRetries {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	if resp == nil {
		return "", fmt.Errorf("failed to fetch token after %d attempts: %w", maxRetries, lastErr)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	// Check for error response
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", errors.New("received empty access token")
	}

	// Cache the token
	tm.cachedToken = &CachedToken{
		Token:     tokenResp.AccessToken,
		ExpiresAt: time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	log.Printf("Successfully obtained new access token, expires in %d seconds", tokenResp.ExpiresIn)
	return tm.cachedToken.Token, nil
}

func isPlaceholderCredential(value string) bool {
	switch strings.TrimSpace(value) {
	case "your_client_id_here", "your_client_secret_here":
		return true
	default:
		return false
	}
}

// InvalidateToken invalidates the current cached token
func (tm *TokenManager) InvalidateToken() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.cachedToken = nil
	log.Println("Token invalidated")
}

