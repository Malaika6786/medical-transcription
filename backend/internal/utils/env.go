// Package utils provides utility functions for the Corti backend service
package utils

import (
	"bytes"
	"log"
	"math"
	"os"
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
