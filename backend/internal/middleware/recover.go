// Package middleware provides HTTP middleware for the Corti backend service
package middleware

import (
	"log"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
)

// RecoverConfig creates a panic recovery middleware
func RecoverConfig() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				requestID, _ := c.Locals("requestID").(string)
				
				log.Printf("[%s] PANIC RECOVERED: %v\n%s",
					requestID,
					r,
					string(debug.Stack()),
				)

				// Return internal server error
				_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success":    false,
					"error":      "Internal server error",
					"request_id": requestID,
				})
			}
		}()

		return c.Next()
	}
}

// ErrorHandler is the global error handler for the application
func ErrorHandler(c *fiber.Ctx, err error) error {
	requestID, _ := c.Locals("requestID").(string)

	// Default to 500
	code := fiber.StatusInternalServerError
	message := "Internal server error"

	// Check for fiber error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	log.Printf("[%s] Error: %v (status: %d)", requestID, err, code)

	return c.Status(code).JSON(fiber.Map{
		"success":    false,
		"error":      message,
		"request_id": requestID,
	})
}

