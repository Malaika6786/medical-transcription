package handlers

import (
	"corti-backend/internal/ai"
	"corti-backend/internal/auth"

	"github.com/gofiber/fiber/v2"
)

// EmbeddingNotifier wakes the background embedding worker after a session
// save. Notification is fire-and-forget: saves never wait on embeddings.
type EmbeddingNotifier interface {
	Notify(sessionID string)
}

// SessionsHandler handles session-related API requests
type SessionsHandler struct {
	sessionStore auth.SessionStorage
	embedNotify  EmbeddingNotifier // may be nil
}

// NewSessionsHandler creates a new sessions handler
func NewSessionsHandler(sessionStore auth.SessionStorage, embedNotify EmbeddingNotifier) *SessionsHandler {
	return &SessionsHandler{
		sessionStore: sessionStore,
		embedNotify:  embedNotify,
	}
}

// HandleListSessions returns all sessions for the current user
func (h *SessionsHandler) HandleListSessions(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)
	sessions := h.sessionStore.GetSessionsByUserID(claims.UserID)
	return c.JSON(sessions)
}

// HandleGetSession returns a specific session
func (h *SessionsHandler) HandleGetSession(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)
	sessionID := c.Params("id")

	session, err := h.sessionStore.GetSessionByID(sessionID, claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Session not found",
		})
	}

	return c.JSON(session)
}

// SaveSessionRequest represents the request body for saving a session
type SaveSessionRequest struct {
	ID            string               `json:"id"`
	Type          string               `json:"type"`
	Title         string               `json:"title"`
	Transcript    string               `json:"transcript"`
	Document      *auth.SavedDocument  `json:"document,omitempty"`
	Extraction    *ai.ExtractionResult `json:"extraction,omitempty"`
	InteractionID string               `json:"interactionId,omitempty"`
}

// HandleSaveSession creates or updates a session
func (h *SessionsHandler) HandleSaveSession(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)

	var req SaveSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.ID == "" || req.Type == "" || req.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing required fields: id, type, title",
		})
	}

	session := &auth.SavedSession{
		ID:            req.ID,
		UserID:        claims.UserID,
		Type:          req.Type,
		Title:         req.Title,
		Transcript:    req.Transcript,
		Document:      req.Document,
		Extraction:    req.Extraction,
		InteractionID: req.InteractionID,
	}

	savedSession, err := h.sessionStore.SaveSession(session)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save session",
		})
	}

	if h.embedNotify != nil {
		h.embedNotify.Notify(savedSession.ID)
	}

	return c.Status(fiber.StatusCreated).JSON(savedSession)
}

// UpdateDocumentRequest represents the request body for updating a session document
type UpdateDocumentRequest struct {
	Document *auth.SavedDocument `json:"document"`
}

// HandleUpdateSessionDocument updates only the document of a session
func (h *SessionsHandler) HandleUpdateSessionDocument(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)
	sessionID := c.Params("id")

	var req UpdateDocumentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Document == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Document is required",
		})
	}

	err := h.sessionStore.UpdateSessionDocument(sessionID, claims.UserID, req.Document)
	if err != nil {
		if err == auth.ErrSessionNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Session not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update session document",
		})
	}

	// Return updated session
	session, _ := h.sessionStore.GetSessionByID(sessionID, claims.UserID)
	return c.JSON(session)
}

// HandleDeleteSession removes a session
func (h *SessionsHandler) HandleDeleteSession(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)
	sessionID := c.Params("id")

	err := h.sessionStore.DeleteSession(sessionID, claims.UserID)
	if err != nil {
		if err == auth.ErrSessionNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Session not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete session",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Session deleted successfully",
	})
}
