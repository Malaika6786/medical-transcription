package handlers

import (
	"log"

	"corti-backend/internal/corti"

	"github.com/gofiber/fiber/v2"
)

// EmbeddedHandler serves Corti Embedded Assistant tokens to the frontend.
// The JWT middleware runs before these handlers — any caller already has a valid app session.
type EmbeddedHandler struct {
	tokenCache *corti.EmbeddedTokenCache
}

// NewEmbeddedHandler creates a new EmbeddedHandler.
func NewEmbeddedHandler(cache *corti.EmbeddedTokenCache) *EmbeddedHandler {
	return &EmbeddedHandler{tokenCache: cache}
}

// HandleGetToken returns valid Embedded Assistant OAuth tokens to the frontend.
// The frontend passes these directly to the <corti-embedded> web component's auth() call.
// GET /api/embedded/token
func (h *EmbeddedHandler) HandleGetToken(c *fiber.Ctx) error {
	tokens, err := h.tokenCache.GetToken()
	if err != nil {
		log.Printf("EmbeddedHandler: failed to get tokens: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to obtain Embedded Assistant tokens. Check CORTI_EMBEDDED_* env vars.",
		})
	}

	return c.JSON(fiber.Map{
		"success":       true,
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"id_token":      tokens.IDToken,
		"token_type":    "Bearer",
	})
}
