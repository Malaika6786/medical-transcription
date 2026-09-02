package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"corti-backend/internal/ai"
)

// Transcripts are 1-5 minute consultations (a few thousand tokens); this cap
// only guards against runaway payloads, not legitimate use.
const maxTranscriptChars = 200_000

// AIHandler handles AI Assistance requests (summary + grounded Q&A).
// Both endpoints are stateless: the transcript travels with each request.
type AIHandler struct {
	client  ai.LLMClient
	timeout time.Duration
}

// NewAIHandler creates a new AIHandler instance.
func NewAIHandler(client ai.LLMClient, timeout time.Duration) *AIHandler {
	return &AIHandler{client: client, timeout: timeout}
}

// SummarizeRequest is the body of POST /api/ai/summarize.
type SummarizeRequest struct {
	Transcript string `json:"transcript"`
	Language   string `json:"language,omitempty"`
}

// SummarizeResponse is the success payload of POST /api/ai/summarize. The
// extraction object is stored by the frontend and sent back with chat
// requests as the grounding source.
type SummarizeResponse struct {
	Success    bool                 `json:"success"`
	Extraction *ai.ExtractionResult `json:"extraction"`
}

// ChatRequest is the body of POST /api/ai/chat. Clients send either the
// stored extraction (preferred — far fewer prompt tokens) or the raw
// transcript as the grounding source.
type ChatRequest struct {
	Transcript string               `json:"transcript,omitempty"`
	Extraction *ai.ExtractionResult `json:"extraction,omitempty"`
	History    []ai.ChatMessage     `json:"history,omitempty"`
	Question   string               `json:"question"`
}

// ChatResponse is the success payload of POST /api/ai/chat.
type ChatResponse struct {
	Success bool   `json:"success"`
	Answer  string `json:"answer"`
}

// HandleSummarize generates a structured summary of a completed transcript.
// POST /api/ai/summarize
func (h *AIHandler) HandleSummarize(c *fiber.Ctx) error {
	requestID, _ := c.Locals("requestID").(string)

	var req SummarizeRequest
	if err := c.BodyParser(&req); err != nil {
		return aiBadRequest(c, "Invalid request body")
	}
	req.Transcript = strings.TrimSpace(req.Transcript)
	if req.Transcript == "" {
		return aiBadRequest(c, "transcript is required")
	}
	if len(req.Transcript) > maxTranscriptChars {
		return aiBadRequest(c, "transcript too large")
	}

	log.Printf("[%s] AI summarize: %d chars, language=%q", requestID, len(req.Transcript), req.Language)

	ctx, cancel := context.WithTimeout(c.UserContext(), h.timeout)
	defer cancel()

	result, err := h.client.Summarize(ctx, req.Transcript, req.Language)
	if err != nil {
		return aiServiceError(c, requestID, "summarize", err)
	}

	return c.JSON(SummarizeResponse{Success: true, Extraction: result})
}

// HandleChat answers a follow-up question grounded in the transcript.
// POST /api/ai/chat
func (h *AIHandler) HandleChat(c *fiber.Ctx) error {
	requestID, _ := c.Locals("requestID").(string)

	var req ChatRequest
	if err := c.BodyParser(&req); err != nil {
		return aiBadRequest(c, "Invalid request body")
	}
	req.Transcript = strings.TrimSpace(req.Transcript)
	req.Question = strings.TrimSpace(req.Question)
	if req.Question == "" {
		return aiBadRequest(c, "question is required")
	}
	if req.Extraction == nil && req.Transcript == "" {
		return aiBadRequest(c, "extraction or transcript is required")
	}
	if len(req.Transcript) > maxTranscriptChars {
		return aiBadRequest(c, "transcript too large")
	}
	grounding := "transcript"
	if req.Extraction != nil {
		grounding = "extraction"
		if payload, err := json.Marshal(req.Extraction); err != nil || len(payload) > maxTranscriptChars {
			return aiBadRequest(c, "extraction too large")
		}
	}

	log.Printf("[%s] AI chat: grounding=%s, %d history turns, question=%q", requestID, grounding, len(req.History), truncateForLog(req.Question, 80))

	ctx, cancel := context.WithTimeout(c.UserContext(), h.timeout)
	defer cancel()

	answer, err := h.client.Chat(ctx, ai.ChatGrounding{
		Transcript: req.Transcript,
		Extraction: req.Extraction,
	}, req.History, req.Question)
	if err != nil {
		return aiServiceError(c, requestID, "chat", err)
	}

	return c.JSON(ChatResponse{Success: true, Answer: answer})
}

// HandleStatus reports LLM endpoint health so the frontend can disable the
// panel (and ops can debug config) without triggering a full completion.
// GET /api/ai/status
func (h *AIHandler) HandleStatus(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	status, err := h.client.Status(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to check AI status",
			"message": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"success": true, "status": status})
}

func aiBadRequest(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"success": false,
		"error":   msg,
	})
}

func aiServiceError(c *fiber.Ctx, requestID, op string, err error) error {
	log.Printf("[%s] AI %s failed: %v", requestID, op, err)

	code := fiber.StatusInternalServerError
	message := "AI request failed"
	switch {
	case errors.Is(err, ai.ErrCompletionLimit):
		code = fiber.StatusBadGateway
		message = "AI model reached its response limit before finishing — please retry"
	case errors.Is(err, ai.ErrRunawayReasoning):
		code = fiber.StatusBadGateway
		message = "AI model did not produce a final answer — please retry"
	case errors.Is(err, ai.ErrUnavailable):
		code = fiber.StatusServiceUnavailable
		message = "AI service unavailable — is the model endpoint running?"
	case errors.Is(err, context.DeadlineExceeded):
		code = fiber.StatusGatewayTimeout
		message = "AI request timed out"
	}
	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"error":   message,
		"message": err.Error(),
	})
}

func truncateForLog(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
