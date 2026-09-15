// Package utils provides utility functions for the Corti backend service
package utils

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	// Corti API Configuration
	CortiClientID     string
	CortiClientSecret string
	CortiAuthURL      string
	CortiAPIBaseURL   string
	CortiTenant       string
	CortiEnvironment  string

	// Server Configuration
	ServerPort string
	ServerHost string

	// CORS Configuration
	CORSAllowedOrigins string

	// Logging
	LogLevel string

	// Token Management
	TokenRefreshBuffer int

	// Embedded Assistant (ROPC)
	CortiEmbeddedClientID string
	CortiEmbeddedUsername string
	CortiEmbeddedPassword string

	// AI Assistance module (OpenAI-compatible endpoint: Ollama local/VM, Groq, ...)
	AIBaseURL             string
	AIModel               string
	AIAPIKey              string
	AITemperature         float64
	AITimeoutSeconds      int
	AIMaxCompletionTokens int

	// Database (PostgreSQL + pgvector — the system of record, docs/adr/0001)
	DatabaseURL string

	// Embedding model (OpenAI-compatible /v1/embeddings endpoint — configured
	// separately from the chat endpoint, which may not serve embeddings).
	// Model must produce 384-dim vectors: bge-small-en-v1.5 (docs/adr/0002).
	EmbedBaseURL        string
	EmbedModel          string
	EmbedAPIKey         string
	EmbedTimeoutSeconds int

	// Field-level encryption (internal/cryptofield) for patient PII columns
	// — see SYSTMONE_INTEGRATION_REPORT.md, "No encryption at application/
	// database field level". Hex-encoded 32-byte key; generate with
	// `openssl rand -hex 32`. Empty in local/demo use is tolerated (patient
	// routes are simply unavailable, see main.go) but must never be empty
	// wherever real patient data is handled.
	FieldEncryptionKey string

	// RequireUKDataResidency, when true (the default), makes main.go refuse
	// to start if CortiEnvironment or AIBaseURL point outside an allowed
	// UK/EU/local set — see ValidateDataResidency. Set
	// DATA_RESIDENCY_UK_ONLY=false only for throwaway local/demo use with
	// synthetic data, never with real patient data.
	RequireUKDataResidency bool

	// NHS/SystmOne integration (internal/nhs) — see docs/nhs/README.md for
	// what each of these unlocks and what still needs NHS-issued
	// credentials before it can reach a real environment.
	PDSBaseURL          string
	PDSAPIKey           string
	MeshBaseURL         string
	MeshMailboxID       string
	MeshMailboxPassword string
	MeshSharedKey       string
	NHSOrgODSCode       string // this organisation's ODS code, used as the FHIR message author
	CIS2IssuerURL       string
	CIS2ClientID        string
	CIS2ClientSecret    string
	CIS2RedirectURL     string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	// Load .env — strip UTF-8 BOM if present, then parse
	loaded := false
	for _, path := range []string{".env", "backend/.env", "../backend/.env", "../../.env"} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		// Strip UTF-8 BOM (0xEF 0xBB 0xBF) added by some editors
		data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
		envMap, err := godotenv.Parse(strings.NewReader(string(data)))
		if err != nil {
			log.Printf(".env parse error at %s: %v", path, err)
			continue
		}
		for k, v := range envMap {
			// Keep explicit process environment values authoritative. This lets
			// a deployment inject secrets such as AI_API_KEY without editing or
			// committing a machine-specific .env file.
			if _, exists := os.LookupEnv(k); exists {
				continue
			}
			if err := os.Setenv(k, v); err != nil {
				log.Printf("Failed to set env var %s: %v", k, err)
			}
		}
		log.Printf("Loaded .env from: %s (process environment takes precedence)", path)
		loaded = true
		break
	}
	if !loaded {
		log.Println("No .env file found, using environment variables")
	}

	config := &Config{
		// Corti API Configuration
		CortiClientID:     getEnv("CORTI_CLIENT_ID", ""),
		CortiClientSecret: getEnv("CORTI_CLIENT_SECRET", ""),
		CortiTenant:       getEnv("CORTI_TENANT", "base"),
		CortiEnvironment:  getEnv("CORTI_ENVIRONMENT", "us"),
		CortiAuthURL:      getEnv("CORTI_AUTH_URL", "https://auth.us.corti.app/realms/base/protocol/openid-connect/token"),
		CortiAPIBaseURL:   getEnv("CORTI_API_BASE_URL", "https://api.us.corti.app"),

		// Server Configuration
		ServerPort: getEnv("SERVER_PORT", "8080"),
		ServerHost: getEnv("SERVER_HOST", "0.0.0.0"),

		// CORS Configuration
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:517,https://localhost:3000,https://avt.xstek.net"),

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "debug"),

		// Token Management
		TokenRefreshBuffer: getEnvAsInt("TOKEN_REFRESH_BUFFER", 60),

		// Embedded Assistant (ROPC)
		CortiEmbeddedClientID: getEnv("CORTI_EMBEDDED_CLIENT_ID", ""),
		CortiEmbeddedUsername: getEnv("CORTI_EMBEDDED_USERNAME", ""),
		CortiEmbeddedPassword: getEnv("CORTI_EMBEDDED_PASSWORD", ""),

		// AI Assistance module
		AIBaseURL: getEnv("AI_BASE_URL", "http://localhost:11434/v1"),
		AIModel:   getEnv("AI_MODEL", "qwythos-16k-chat"),
		AIAPIKey:  getEnv("AI_API_KEY", ""),
		// Hosted live-demo profile uses low temperature for stable structured JSON;
		// local Qwythos fallback can override this in backend/.env.
		AITemperature: getEnvAsFloat64("AI_TEMPERATURE", 0.6),
		// The 15-field extraction takes ~2-2.5 min on a laptop-class machine
		// (fits comfortably on the GPU VM); 120 timed out locally.
		AITimeoutSeconds: getEnvAsInt("AI_TIMEOUT_SECONDS", 300),
		// 4096 covers the <think> reasoning trace plus a dense 15-field
		// extraction; 2048 truncated intermittently, 1024 always.
		AIMaxCompletionTokens: getEnvAsInt("AI_MAX_COMPLETION_TOKENS", 4096),

		// Database
		DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost:5432/meditrans?sslmode=disable"),

		// Embeddings (local Ollama by default)
		EmbedBaseURL:        getEnv("EMBED_BASE_URL", "http://localhost:11434/v1"),
		EmbedModel:          getEnv("EMBED_MODEL", "hf.co/CompendiumLabs/bge-small-en-v1.5-gguf"),
		EmbedAPIKey:         getEnv("EMBED_API_KEY", ""),
		EmbedTimeoutSeconds: getEnvAsInt("EMBED_TIMEOUT_SECONDS", 60),

		FieldEncryptionKey:     getEnv("FIELD_ENCRYPTION_KEY", ""),
		RequireUKDataResidency: getEnvAsBool("DATA_RESIDENCY_UK_ONLY", true),

		PDSBaseURL:          getEnv("PDS_BASE_URL", "https://sandbox.api.service.nhs.uk/personal-demographics/FHIR/R4"),
		PDSAPIKey:           getEnv("PDS_API_KEY", ""),
		MeshBaseURL:         getEnv("MESH_BASE_URL", "https://msg.intspineservices.nhs.uk"),
		MeshMailboxID:       getEnv("MESH_MAILBOX_ID", ""),
		MeshMailboxPassword: getEnv("MESH_MAILBOX_PASSWORD", ""),
		MeshSharedKey:       getEnv("MESH_SHARED_KEY", ""),
		NHSOrgODSCode:       getEnv("NHS_ORG_ODS_CODE", ""),
		CIS2IssuerURL:       getEnv("CIS2_ISSUER_URL", ""),
		CIS2ClientID:        getEnv("CIS2_CLIENT_ID", ""),
		CIS2ClientSecret:    getEnv("CIS2_CLIENT_SECRET", ""),
		CIS2RedirectURL:     getEnv("CIS2_REDIRECT_URL", ""),
	}
	if config.AIMaxCompletionTokens <= 0 {
		log.Printf(
			"WARNING: AI_MAX_COMPLETION_TOKENS must be positive; using default %d",
			4096,
		)
		config.AIMaxCompletionTokens = 4096
	}
	if math.IsNaN(config.AITemperature) || math.IsInf(config.AITemperature, 0) ||
		config.AITemperature < 0 || config.AITemperature > 2 {
		log.Printf(
			"WARNING: AI_TEMPERATURE must be between 0 and 2; using default %.1f",
			0.6,
		)
		config.AITemperature = 0.6
	}

	// Validate required configuration
	if isPlaceholderConfigValue(config.CortiClientID) || isPlaceholderConfigValue(config.CortiClientSecret) {
		log.Println("WARNING: CORTI_CLIENT_ID and CORTI_CLIENT_SECRET are still set to example placeholder values")
	} else if config.CortiClientID == "" || config.CortiClientSecret == "" {
		log.Println("WARNING: CORTI_CLIENT_ID and CORTI_CLIENT_SECRET must be set")
	}

	if config.CortiEmbeddedClientID == "" || config.CortiEmbeddedUsername == "" || config.CortiEmbeddedPassword == "" {
		log.Println("WARNING: CORTI_EMBEDDED_* vars not set — Embedded Assistant will not work")
	}

	return config
}

// getEnv retrieves an environment variable with a default fallback
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func isPlaceholderConfigValue(value string) bool {
	switch strings.TrimSpace(value) {
	case "your_client_id_here", "your_client_secret_here":
		return true
	default:
		return false
	}
}

// getEnvAsInt retrieves an environment variable as an integer with a default fallback
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsFloat64 retrieves an environment variable as a float with a default fallback.
func getEnvAsFloat64(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// getEnvAsBool retrieves an environment variable as a bool with a default
// fallback. Accepts the same forms as strconv.ParseBool ("true"/"false"/
// "1"/"0"/"t"/"f", case-insensitive).
func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

// ukApprovedCortiEnvironments/ukApprovedAIHosts are the allowlists
// ValidateDataResidency checks against. Corti's EU region value needs
// confirming against Corti's own docs once an account exists there (see
// SYSTMONE_INTEGRATION_REPORT.md §7) — "eu" is this project's best-effort
// guess at the value Corti expects, flagged for verification, not asserted
// as certain.
var ukApprovedCortiEnvironments = map[string]bool{
	"eu": true,
	"uk": true,
}

// ukApprovedAIHosts lists hosts considered acceptable for the AI
// summarization endpoint when data residency is enforced: a local/private
// deployment (localhost, the project's own GPU VM pattern) is always fine
// since the operator controls exactly where that runs.
var ukApprovedAIHosts = []string{
	"localhost",
	"127.0.0.1",
}

// ValidateDataResidency enforces config.RequireUKDataResidency (default
// true): if set, it is fatal for CortiEnvironment or AIBaseURL to point
// somewhere outside the UK/EU/local allowlist. This exists because the
// project's own shipped defaults previously sent transcript/extraction
// content to Corti's US region and a non-UK LLM endpoint by default — see
// SYSTMONE_INTEGRATION_REPORT.md, "Default AI data flows are not UK-
// hosted". Returns a non-nil error describing exactly what's out of policy
// instead of logging and continuing, because silently continuing is
// exactly the failure mode this check exists to prevent.
func (c *Config) ValidateDataResidency() error {
	if !c.RequireUKDataResidency {
		return nil
	}
	var problems []string
	if !ukApprovedCortiEnvironments[strings.ToLower(c.CortiEnvironment)] {
		problems = append(problems, fmt.Sprintf(
			"CORTI_ENVIRONMENT=%q is not in the UK/EU allowlist (%v) — patient audio/transcript data would leave the UK/EU by default",
			c.CortiEnvironment, sortedKeys(ukApprovedCortiEnvironments)))
	}
	if host := hostOf(c.AIBaseURL); host != "" && !isApprovedAIHost(host) {
		problems = append(problems, fmt.Sprintf(
			"AI_BASE_URL host %q is not in the approved list (%v, or your own private/self-hosted endpoint added to ukApprovedAIHosts) — clinical extraction content would be sent there",
			host, ukApprovedAIHosts))
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf(
		"data residency check failed (set DATA_RESIDENCY_UK_ONLY=false only for local/demo use with synthetic data, never with real patient data):\n  - %s",
		strings.Join(problems, "\n  - "))
}

func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func isApprovedAIHost(host string) bool {
	for _, h := range ukApprovedAIHosts {
		if strings.EqualFold(h, host) {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
