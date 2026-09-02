package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// OpenAICompatClient implements LLMClient against any /v1/chat/completions
// endpoint: local Ollama, an Ollama/vLLM VM, Groq, etc. The provider is chosen
// purely by configuration (AI_BASE_URL / AI_MODEL / AI_API_KEY).
type OpenAICompatClient struct {
	baseURL             string
	model               string
	apiKey              string
	temperature         float64
	maxCompletionTokens int
	httpClient          *http.Client
}

// The completion budget must cover the <think> reasoning trace plus the full
// 15-field extraction JSON. Measured against qwythos-16k-chat: a dense
// consultation intermittently truncates at 2048 (reasoning-length variance),
// completes reliably at 4096.
const defaultMaxCompletionTokens = 4096

var (
	// ErrUnavailable marks connectivity failures so handlers can map them to 503.
	ErrUnavailable = errors.New("AI service unavailable")
	// ErrCompletionLimit marks a response truncated by the configured token cap.
	ErrCompletionLimit = errors.New("AI completion token limit reached")
	// ErrRunawayReasoning marks repeated reasoning-only responses with no final answer.
	ErrRunawayReasoning = errors.New("AI model produced no usable final answer")
)

// NewOpenAICompatClient builds a client. timeout covers a full completion
// round-trip (local models may take tens of seconds on first load).
func NewOpenAICompatClient(
	baseURL, model, apiKey string,
	temperature float64,
	timeout time.Duration,
	maxCompletionTokens int,
) *OpenAICompatClient {
	if maxCompletionTokens <= 0 {
		maxCompletionTokens = defaultMaxCompletionTokens
	}
	return &OpenAICompatClient{
		baseURL:             strings.TrimRight(baseURL, "/"),
		model:               model,
		apiKey:              apiKey,
		maxCompletionTokens: maxCompletionTokens,
		temperature:         temperature,
		httpClient:          &http.Client{Timeout: timeout},
	}
}

// --- OpenAI wire types (only the fields we use) ---

type oaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type oaiChatRequest struct {
	Model       string       `json:"model"`
	Messages    []oaiMessage `json:"messages"`
	Temperature float64      `json:"temperature"`
	MaxTokens   int          `json:"max_tokens"`
	Stream      bool         `json:"stream"`
}

type oaiChatResponse struct {
	Choices []struct {
		Message      oaiMessage `json:"message"`
		FinishReason string     `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// chatCompletion performs a completion, retrying once if the model produced
// no usable text (the reasoning fine-tune occasionally emits only its trace or
// identity boilerplate; a fresh sample at temperature 0.6 recovers).
func (c *OpenAICompatClient) chatCompletion(ctx context.Context, messages []oaiMessage) (string, error) {
	answer, err := c.chatCompletionOnce(ctx, messages)
	if err != nil || answer != "" {
		return answer, err
	}

	log.Printf("ai: empty answer after sanitizing, retrying once")
	answer, err = c.chatCompletionOnce(ctx, messages)
	if err != nil {
		return "", err
	}
	if answer == "" {
		return "", fmt.Errorf("%w after retry", ErrRunawayReasoning)
	}
	return answer, nil
}

// chatCompletionOnce performs one non-streaming completion and returns the
// assistant text with reasoning traces stripped.
func (c *OpenAICompatClient) chatCompletionOnce(ctx context.Context, messages []oaiMessage) (string, error) {
	payload, err := json.Marshal(oaiChatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: c.temperature,
		MaxTokens:   c.maxCompletionTokens,
		Stream:      false,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return "", fmt.Errorf("AI completion timed out: %w", context.DeadlineExceeded)
		}
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var parsed oaiChatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("LLM returned non-JSON response (HTTP %d): %s", resp.StatusCode, truncate(string(body), 200))
	}
	if resp.StatusCode != http.StatusOK {
		msg := truncate(string(body), 200)
		if parsed.Error != nil {
			msg = parsed.Error.Message
		}
		return "", fmt.Errorf("LLM error (HTTP %d): %s", resp.StatusCode, msg)
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("LLM returned no choices")
	}

	choice := parsed.Choices[0]
	if strings.EqualFold(strings.TrimSpace(choice.FinishReason), "length") {
		return "", fmt.Errorf(
			"%w (%d tokens); model did not finish its response",
			ErrCompletionLimit,
			c.maxCompletionTokens,
		)
	}

	return sanitizeArtifacts(stripReasoning(choice.Message.Content)), nil
}

// sanitizeArtifacts removes known quirks of the Qwythos fine-tune: it
// occasionally appends its baked-in identity line ("I am Qwythos created by
// Empero AI") to otherwise-correct answers, or prefixes the answer with its
// own name ("Qwythos: ..."). Grounded answers must come from the transcript
// only, so both are always noise here.
func sanitizeArtifacts(s string) string {
	lines := strings.Split(s, "\n")
	kept := lines[:0]
	for _, line := range lines {
		t := strings.ToLower(strings.Trim(strings.TrimSpace(line), `"'`))
		if strings.Contains(t, "qwythos") && strings.Contains(t, "empero") {
			continue
		}
		kept = append(kept, line)
	}
	s = strings.TrimSpace(strings.Join(kept, "\n"))
	if len(s) > 8 && strings.EqualFold(s[:8], "qwythos:") {
		s = strings.TrimSpace(s[8:])
	}
	return s
}

// stripReasoning removes <think>...</think> reasoning traces so they never
// reach the frontend. The generation prompt opens the think block server-side,
// so the completion may contain a bare closing tag with no opener — everything
// up to the last </think> is reasoning either way. An unclosed opener means the
// model never produced a final answer, so return empty and let the caller retry.
func stripReasoning(s string) string {
	lastOpen := strings.LastIndex(s, "<think>")
	lastClose := strings.LastIndex(s, "</think>")
	if lastOpen > lastClose {
		return ""
	}
	if lastClose >= 0 {
		s = s[lastClose+len("</think>"):]
	}
	s = strings.ReplaceAll(s, "<think>", "")
	return strings.TrimSpace(s)
}

// Summarize implements LLMClient.
func (c *OpenAICompatClient) Summarize(ctx context.Context, transcript, language string) (*ExtractionResult, error) {
	raw, err := c.chatCompletion(ctx, buildExtractionMessages(transcript, language))
	if err != nil {
		return nil, err
	}

	result, err := parseExtractionJSON(raw)
	if err != nil {
		// Plain-text fallback: the model ignored the JSON instruction. Keep
		// the demo alive by treating the whole answer as the summary.
		log.Printf("ai: extraction JSON parse failed (%v), falling back to plain text", err)
		result = &ExtractionResult{Summary: raw}
		result.normalize()
	}
	return result, nil
}

// parseExtractionJSON extracts the first JSON object from the model output —
// tolerates markdown fences and prose around it.
func parseExtractionJSON(raw string) (*ExtractionResult, error) {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return nil, errors.New("no JSON object in output")
	}

	var result ExtractionResult
	if err := json.Unmarshal([]byte(raw[start:end+1]), &result); err != nil {
		return nil, err
	}
	if result.Summary == "" {
		return nil, errors.New("summary field missing or empty")
	}
	// Normalize nils so the API always returns arrays.
	result.normalize()
	return &result, nil
}

// Chat implements LLMClient. The stored extraction is the preferred grounding
// source; the raw transcript is the fallback.
func (c *OpenAICompatClient) Chat(ctx context.Context, grounding ChatGrounding, history []ChatMessage, question string) (string, error) {
	var messages []oaiMessage
	if grounding.Extraction != nil {
		messages = buildChatMessagesFromRecord(grounding.Extraction, history, question)
	} else {
		messages = buildChatMessages(grounding.Transcript, history, question)
	}

	answer, err := c.chatCompletion(ctx, messages)
	if err != nil {
		return "", err
	}
	if answer == "" {
		return "", errors.New("LLM returned an empty answer")
	}
	return answer, nil
}

// Status implements LLMClient: checks endpoint reachability via /v1/models and
// reports whether the configured model is served there.
func (c *OpenAICompatClient) Status(ctx context.Context) (*StatusResult, error) {
	status := &StatusResult{Model: c.model, BaseURL: c.baseURL}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		status.Detail = err.Error()
		return status, nil
	}
	defer resp.Body.Close()

	status.Reachable = resp.StatusCode == http.StatusOK
	if !status.Reachable {
		status.Detail = fmt.Sprintf("HTTP %d from %s/models", resp.StatusCode, c.baseURL)
		return status, nil
	}

	var models struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1024*1024)).Decode(&models); err == nil {
		for _, m := range models.Data {
			// Ollama lists models with an explicit tag ("name:latest").
			if m.ID == c.model || strings.TrimSuffix(m.ID, ":latest") == c.model {
				status.ModelAvailable = true
				break
			}
		}
		if !status.ModelAvailable {
			status.Detail = fmt.Sprintf("model %q not in endpoint's model list", c.model)
		}
	}
	return status, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
