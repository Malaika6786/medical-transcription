// Package middleware provides HTTP middleware for the Corti backend service
package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// RequestLogger creates a structured logging middleware
func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Generate or extract request ID
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("X-Request-ID", requestID)
		c.Locals("requestID", requestID)

		// Log incoming request
		log.Printf("[%s] %s %s %s - Started",
			requestID,
			c.Method(),
			c.Path(),
			c.IP(),
		)

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Log response
		log.Printf("[%s] %s %s - %d - %v",
			requestID,
			c.Method(),
			c.Path(),
			c.Response().StatusCode(),
			duration,
		)

		return err
	}
}

