package ai

import "context"

// LLMClient is the provider-agnostic interface for the AI Assistance module.
// Implementations must return answers grounded exclusively in the supplied
// consultation material and must never let it leave the configured endpoint.
type LLMClient interface {
	// Summarize produces the structured clinical extraction for a completed
	// consultation transcript. language is a BCP-47-ish hint ("en", "en-GB",
	// ...); empty means English.
	Summarize(ctx context.Context, transcript string, language string) (*ExtractionResult, error)

	// Chat answers a follow-up question grounded in the consultation.
	// grounding carries either the stored extraction (preferred — much
	// cheaper in prompt tokens) or the raw transcript; history carries prior
	// Q&A turns so the conversation stays coherent. The grounding travels
	// with every call (stateless backend, matching the app's existing
	// patterns).
	Chat(ctx context.Context, grounding ChatGrounding, history []ChatMessage, question string) (string, error)

	// Status reports whether the LLM endpoint is reachable and whether the
	// configured model is available on it.
	Status(ctx context.Context) (*StatusResult, error)
}

// StatusResult describes the health of the configured LLM endpoint.
type StatusResult struct {
	Reachable      bool   `json:"reachable"`
	Model          string `json:"model"`
	ModelAvailable bool   `json:"model_available"`
	BaseURL        string `json:"base_url"`
	Detail         string `json:"detail,omitempty"`
}
