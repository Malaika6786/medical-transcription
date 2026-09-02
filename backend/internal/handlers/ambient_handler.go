// Package handlers provides HTTP request handlers for the Corti backend service
package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"

	"corti-backend/internal/corti"
)

// AmbientHandler handles ambient streaming requests
type AmbientHandler struct {
	ambientProxy *corti.AmbientProxy
}

// NewAmbientHandler creates a new AmbientHandler instance
func NewAmbientHandler(ambientProxy *corti.AmbientProxy) *AmbientHandler {
	return &AmbientHandler{
		ambientProxy: ambientProxy,
	}
}

// StartSessionRequest represents the request to start an ambient session
type StartSessionRequest struct {
	Language   string            `json:"language"`
	ExternalID string            `json:"external_id,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// StartSessionResponse represents the response from starting an ambient session
type StartSessionResponse struct {
	Success       bool   `json:"success"`
	InteractionID string `json:"interaction_id"`
	WebSocketURL  string `json:"websocket_url"`
	Status        string `json:"status"`
	Message       string `json:"message,omitempty"`
}

// HandleStartSession creates a new ambient session and returns WebSocket connection details
// POST /api/ambient/start
func (h *AmbientHandler) HandleStartSession(c *fiber.Ctx) error {
	requestID, _ := c.Locals("requestID").(string)
	log.Printf("[%s] Starting new ambient session", requestID)

	// Parse request body
	var req StartSessionRequest
	if err := c.BodyParser(&req); err != nil {
		// Use defaults if parsing fails
		req = StartSessionRequest{
			Language: "en-US",
		}
	}

	if req.Language == "" {
		req.Language = "en-US"
	}

	// Create session via proxy
	interaction, err := h.ambientProxy.StartSession(req.Language, req.Metadata)
	if err != nil {
		log.Printf("[%s] Failed to create ambient session: %v", requestID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to create ambient session",
			"message": err.Error(),
		})
	}

	log.Printf("[%s] Created ambient session: %s, Corti WS: %s", requestID, interaction.InteractionID, interaction.WebSocketURL)

	// Store Corti WebSocket URL for later use
	h.ambientProxy.StoreCortiWSURL(interaction.InteractionID, interaction.WebSocketURL)

	// Build our WebSocket URL (proxy endpoint)
	host := c.Hostname()
	protocol := "ws"
	if c.Protocol() == "https" {
		protocol = "wss"
	}
	wsURL := protocol + "://" + host + "/api/ambient/ws/" + interaction.InteractionID

	return c.Status(fiber.StatusCreated).JSON(StartSessionResponse{
		Success:       true,
		InteractionID: interaction.InteractionID,
		WebSocketURL:  wsURL,
		Status:        "created",
		Message:       "Ambient session created. Connect to the WebSocket URL to start streaming.",
	})
}

// HandleWebSocketUpgrade is the middleware to check WebSocket upgrade eligibility
func (h *AmbientHandler) HandleWebSocketUpgrade() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			// Store language from query params for WebSocket handler
			language := c.Query("language", "en-US")
			c.Locals("language", language)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	}
}

// HandleWebSocket handles the WebSocket connection for audio streaming
// WS /api/ambient/ws/:interactionId?language=en-US
func (h *AmbientHandler) HandleWebSocket() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		interactionID := c.Params("interactionId")
		language := c.Query("language", "en-US")
		// audioFormat: mobile clients pass "pcm16" to declare headerless raw
		// PCM (16-bit, 16kHz, mono) to Corti. Web clients omit this — their
		// WebM/Opus container is self-describing and Corti auto-detects it.
		audioFormat := ""
		if c.Query("format") == "pcm16" {
			audioFormat = "audio/pcm; rate=16000; channels=1; bits=16"
		}

		log.Printf("WebSocket connection established for interaction: %s, language: %s", interactionID, language)

		if interactionID == "" {
			log.Println("Missing interaction ID in WebSocket request")
			c.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","error":"Missing interaction ID"}`))
			c.Close()
			return
		}

		// Handle the WebSocket proxy connection with language
		if err := h.ambientProxy.HandleWebSocket(c, interactionID, language, audioFormat); err != nil {
			log.Printf("WebSocket proxy error for %s: %v", interactionID, err)
		}

		log.Printf("WebSocket connection closed for interaction: %s", interactionID)
	}, websocket.Config{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		Origins:         []string{"*"}, // Configure appropriately for production
	})
}

// SessionStatusResponse represents the response for session status
type SessionStatusResponse struct {
	Success       bool                       `json:"success"`
	InteractionID string                     `json:"interaction_id"`
	Status        string                     `json:"status"`
	SessionState  *corti.AmbientSessionState `json:"session_state,omitempty"`
	Error         string                     `json:"error,omitempty"`
}

// HandleGetSessionStatus retrieves the current status of an ambient session
// GET /api/ambient/session/:interactionId
func (h *AmbientHandler) HandleGetSessionStatus(c *fiber.Ctx) error {
	requestID, _ := c.Locals("requestID").(string)
	interactionID := c.Params("interactionId")

	if interactionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Interaction ID is required",
		})
	}

	log.Printf("[%s] Getting session status for: %s", requestID, interactionID)

	state, err := h.ambientProxy.GetSessionState(interactionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(SessionStatusResponse{
			Success:       false,
			InteractionID: interactionID,
			Error:         "Session not found or inactive",
		})
	}

	return c.JSON(SessionStatusResponse{
		Success:       true,
		InteractionID: interactionID,
		Status:        state.Status,
		SessionState:  state,
	})
}

// StatsResponse represents the response for server statistics
type StatsResponse struct {
	ActiveSessions int `json:"active_sessions"`
}

// HandleGetStats retrieves server statistics
// GET /api/ambient/stats
func (h *AmbientHandler) HandleGetStats(c *fiber.Ctx) error {
	return c.JSON(StatsResponse{
		ActiveSessions: h.ambientProxy.ActiveSessionCount(),
	})
}
