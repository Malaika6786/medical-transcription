package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"corti-backend/internal/ai"
)

// mockLLMClient records the grounding it was called with and returns canned
// results, so handler behavior can be tested without an LLM endpoint.
type mockLLMClient struct {
	lastGrounding ai.ChatGrounding
	extraction    *ai.ExtractionResult
	answer        string
}

func (m *mockLLMClient) Summarize(ctx context.Context, transcript, language string) (*ai.ExtractionResult, error) {
	return m.extraction, nil
}

func (m *mockLLMClient) Chat(ctx context.Context, grounding ai.ChatGrounding, history []ai.ChatMessage, question string) (string, error) {
	m.lastGrounding = grounding
	return m.answer, nil
}

func (m *mockLLMClient) Status(ctx context.Context) (*ai.StatusResult, error) {
	return &ai.StatusResult{}, nil
}

func newChatTestApp(mock *mockLLMClient) *fiber.App {
	app := fiber.New()
	handler := NewAIHandler(mock, time.Second)
	app.Post("/summarize", handler.HandleSummarize)
	app.Post("/chat", handler.HandleChat)
	return app
}

func postJSON(t *testing.T, app *fiber.App, path, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return response
}

func TestHandleChatRequiresSomeGrounding(t *testing.T) {
	app := newChatTestApp(&mockLLMClient{answer: "Answer."})

	response := postJSON(t, app, "/chat", `{"question": "What allergies?"}`)
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", response.StatusCode)
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error != "extraction or transcript is required" {
		t.Errorf("error message: got %q", body.Error)
	}
}

func TestHandleChatUsesExtractionGrounding(t *testing.T) {
	mock := &mockLLMClient{answer: "The patient takes ibuprofen 400 mg."}
	app := newChatTestApp(mock)

	response := postJSON(t, app, "/chat", `{
		"question": "What medication?",
		"extraction": {"summary": "Patient reports headaches.", "medications": ["ibuprofen 400 mg"]}
	}`)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", response.StatusCode)
	}
	if mock.lastGrounding.Extraction == nil {
		t.Fatal("handler should pass the extraction through as grounding")
	}
	if got := mock.lastGrounding.Extraction.Medications; len(got) != 1 || got[0] != "ibuprofen 400 mg" {
		t.Errorf("extraction medications: got %#v", got)
	}
}

func TestHandleChatAcceptsTranscriptGrounding(t *testing.T) {
	mock := &mockLLMClient{answer: "No allergies were mentioned."}
	app := newChatTestApp(mock)

	response := postJSON(t, app, "/chat", `{
		"question": "Any allergies?",
		"transcript": "Doctor: any allergies? Patient: none."
	}`)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", response.StatusCode)
	}
	if mock.lastGrounding.Extraction != nil {
		t.Error("no extraction was sent, grounding should be transcript-only")
	}
	if mock.lastGrounding.Transcript == "" {
		t.Error("transcript grounding should pass the transcript through")
	}
}

func TestHandleSummarizeReturnsNestedExtraction(t *testing.T) {
	mock := &mockLLMClient{extraction: &ai.ExtractionResult{
		Summary:        "Patient reports headaches.",
		ChiefComplaint: "headaches",
	}}
	app := newChatTestApp(mock)

	response := postJSON(t, app, "/summarize", `{"transcript": "Doctor: hello."}`)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", response.StatusCode)
	}
	var body struct {
		Success    bool `json:"success"`
		Extraction *struct {
			Summary        string `json:"summary"`
			ChiefComplaint string `json:"chiefComplaint"`
		} `json:"extraction"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Success || body.Extraction == nil {
		t.Fatalf("expected success with nested extraction, got %+v", body)
	}
	if body.Extraction.ChiefComplaint != "headaches" {
		t.Errorf("chiefComplaint: got %q", body.Extraction.ChiefComplaint)
	}
}

func TestAIServiceErrorClassifiesGenerationFailures(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantMessage string
	}{
		{
			name:        "completion limit",
			err:         fmt.Errorf("wrapped: %w", ai.ErrCompletionLimit),
			wantStatus:  http.StatusBadGateway,
			wantMessage: "AI model reached its response limit before finishing — please retry",
		},
		{
			name:        "runaway reasoning",
			err:         fmt.Errorf("wrapped: %w", ai.ErrRunawayReasoning),
			wantStatus:  http.StatusBadGateway,
			wantMessage: "AI model did not produce a final answer — please retry",
		},
		{
			name:        "timeout",
			err:         fmt.Errorf("wrapped: %w", context.DeadlineExceeded),
			wantStatus:  http.StatusGatewayTimeout,
			wantMessage: "AI request timed out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/", func(c *fiber.Ctx) error {
				return aiServiceError(c, "test-request", "test-operation", tt.err)
			})

			response, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != tt.wantStatus {
				t.Errorf("status: got %d, want %d", response.StatusCode, tt.wantStatus)
			}

			var body struct {
				Error string `json:"error"`
			}
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Error != tt.wantMessage {
				t.Errorf("error message: got %q, want %q", body.Error, tt.wantMessage)
			}
		})
	}
}
