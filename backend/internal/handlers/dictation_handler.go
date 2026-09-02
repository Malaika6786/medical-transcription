// Package handlers provides HTTP handlers for dictation endpoints
// Reference: https://docs.corti.ai/quickstart/dictation
package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"

	"corti-backend/internal/corti"
)

// DictationHandler handles dictation-related requests
type DictationHandler struct {
	dictationProxy *corti.DictationProxy
}

// NewDictationHandler creates a new dictation handler
func NewDictationHandler(dictationProxy *corti.DictationProxy) *DictationHandler {
	return &DictationHandler{
		dictationProxy: dictationProxy,
	}
}

// HandleWebSocketUpgrade returns middleware for WebSocket upgrade
func (h *DictationHandler) HandleWebSocketUpgrade() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	}
}

// HandleWebSocket handles WebSocket connections for dictation
// WS /api/dictation/ws
func (h *DictationHandler) HandleWebSocket() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		// Get language from query params
		language := c.Query("language", "en")
		// audioFormat: mobile clients pass "pcm16" to declare headerless raw
		// PCM (16-bit, 16kHz, mono) to Corti — see ambient_handler.go for the
		// same convention.
		audioFormat := ""
		if c.Query("format") == "pcm16" {
			audioFormat = "audio/pcm; rate=16000; channels=1; bits=16"
		}

		log.Printf("Dictation WebSocket connection - Language: %s", language)

		// Handle WebSocket connection
		err := h.dictationProxy.HandleWebSocket(c, language, audioFormat)
		if err != nil {
			log.Printf("Dictation WebSocket error: %v", err)
		}
	}, websocket.Config{
		ReadBufferSize:  16384,
		WriteBufferSize: 16384,
	})
}

// HandleGetStats returns dictation statistics
// GET /api/dictation/stats
func (h *DictationHandler) HandleGetStats(c *fiber.Ctx) error {
	stats := h.dictationProxy.GetStats()
	return c.JSON(fiber.Map{
		"success": true,
		"stats":   stats,
	})
}

