// Package middleware provides HTTP middleware for the Corti backend service
package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"corti-backend/internal/utils"
)

// CORSConfig creates a CORS middleware configuration
func CORSConfig(config *utils.Config) fiber.Handler {
	allowedOrigins := config.CORSAllowedOrigins
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173"
	}

	return cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     strings.Join([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}, ","),
		AllowHeaders:     strings.Join([]string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"}, ","),
		ExposeHeaders:    strings.Join([]string{"Content-Length", "Content-Type", "X-Request-ID"}, ","),
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	})
}

