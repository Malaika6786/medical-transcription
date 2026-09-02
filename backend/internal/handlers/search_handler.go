package handlers

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"

	"corti-backend/internal/ai"
	"corti-backend/internal/auth"
	"corti-backend/internal/pgstore"
)

// searchMinScore filters out hits whose cosine similarity is too low to be a
// meaningful match; the UI shows scores, so keep this permissive.
const searchMinScore = 0.35

const (
	searchDefaultLimit = 10
	searchMaxLimit     = 25
	snippetMaxRunes    = 280
)

// SearchHandler serves semantic search over the caller's own sessions.
type SearchHandler struct {
	store    *pgstore.Store
	embedder *ai.Embedder
}

// NewSearchHandler creates a new semantic search handler.
func NewSearchHandler(store *pgstore.Store, embedder *ai.Embedder) *SearchHandler {
	return &SearchHandler{store: store, embedder: embedder}
}

// SearchRequest is the body of POST /api/search.
type SearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// HandleSearch embeds the query and ranks the user's sessions by cosine
// similarity over chunk + extraction embeddings.
func (h *SearchHandler) HandleSearch(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)

	var req SearchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "Invalid request body",
		})
	}
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "Query is required",
		})
	}
	if req.Limit <= 0 {
		req.Limit = searchDefaultLimit
	}
	if req.Limit > searchMaxLimit {
		req.Limit = searchMaxLimit
	}

	ctx := c.UserContext()
	queryVec, err := h.embedder.EmbedQuery(ctx, req.Query)
	if err != nil {
		log.Printf("search: embed query: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"success": false,
			"error":   "Embedding service unavailable — check that the embedding endpoint (Ollama) is running",
		})
	}

	results, err := h.store.SearchSessions(ctx, claims.UserID, queryVec, req.Limit, searchMinScore)
	if err != nil {
		log.Printf("search: query: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "Search failed",
		})
	}

	for i := range results {
		results[i].Snippet = truncateSnippet(results[i].Snippet)
	}
	return c.JSON(fiber.Map{
		"success": true,
		"query":   req.Query,
		"results": results,
	})
}

func truncateSnippet(s string) string {
	runes := []rune(s)
	if len(runes) <= snippetMaxRunes {
		return s
	}
	return strings.TrimRight(string(runes[:snippetMaxRunes]), " ") + "…"
}
