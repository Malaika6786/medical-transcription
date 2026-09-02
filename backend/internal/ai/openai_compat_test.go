package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestStripReasoning(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			// Template opens the think block server-side, so completions often
			// carry only the closing tag.
			name: "bare closing tag",
			in:   "the patient mentioned penicillin\n</think>\n\nThe allergy is penicillin.",
			want: "The allergy is penicillin.",
		},
		{
			name: "full think block",
			in:   "<think>\nreasoning here\n</think>\n\nAnswer.",
			want: "Answer.",
		},
		{
			name: "no reasoning",
			in:   "Plain answer.",
			want: "Plain answer.",
		},
		{
			name: "unclosed reasoning is discarded",
			in:   "<think>Answer without close.",
			want: "",
		},
	}
	for _, tt := range tests {
		if got := stripReasoning(tt.in); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestSanitizeArtifacts(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "appended identity line",
			in:   "\"The doctor advised stopping ibuprofen.\"\n\"I am Qwythos created by Empero AI.\"",
			want: "\"The doctor advised stopping ibuprofen.\"",
		},
		{
			name: "name prefix",
			in:   "Qwythos: The patient was told to stop taking ibuprofen.",
			want: "The patient was told to stop taking ibuprofen.",
		},
		{
			name: "identity-only becomes empty (triggers retry)",
			in:   "I am Qwythos, created by Empero AI.",
			want: "",
		},
		{
			name: "clean answer untouched",
			in:   "No allergies were mentioned in this conversation.",
			want: "No allergies were mentioned in this conversation.",
		},
	}
	for _, tt := range tests {
		if got := sanitizeArtifacts(tt.in); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestChatCompletionSendsMaxTokens(t *testing.T) {
	const wantMaxTokens = 321
	const wantTemperature = 1.0
	var gotRequest oaiChatRequest

	client := NewOpenAICompatClient(
		"http://ai.test/v1",
		"test-model",
		"",
		wantTemperature,
		time.Second,
		wantMaxTokens,
	)
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(req.Body).Decode(&gotRequest); err != nil {
			t.Errorf("decode request: %v", err)
		}
		return jsonHTTPResponse(`{
			"choices": [{
				"message": {"role": "assistant", "content": "Final answer."},
				"finish_reason": "stop"
			}]
		}`), nil
	})

	answer, err := client.chatCompletion(context.Background(), []oaiMessage{
		{Role: "user", Content: "Question"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer != "Final answer." {
		t.Errorf("answer: got %q", answer)
	}
	if gotRequest.MaxTokens != wantMaxTokens {
		t.Errorf("max_tokens: got %d, want %d", gotRequest.MaxTokens, wantMaxTokens)
	}
	if gotRequest.Temperature != wantTemperature {
		t.Errorf("temperature: got %g, want %g", gotRequest.Temperature, wantTemperature)
	}
}

func TestChatCompletionRejectsLengthLimitedResponseWithoutRetry(t *testing.T) {
	var calls atomic.Int32
	client := NewOpenAICompatClient("http://ai.test/v1", "test-model", "", 0.6, time.Second, 64)
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		return jsonHTTPResponse(`{
			"choices": [{
				"message": {"role": "assistant", "content": "</think>Partial final answer"},
				"finish_reason": "length"
			}]
		}`), nil
	})

	_, err := client.chatCompletion(context.Background(), []oaiMessage{
		{Role: "user", Content: "Question"},
	})
	if !errors.Is(err, ErrCompletionLimit) {
		t.Fatalf("got error %v, want ErrCompletionLimit", err)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("length-limited response should not retry: got %d calls, want 1", got)
	}
}

func TestChatCompletionRejectsRepeatedUnclosedReasoning(t *testing.T) {
	var calls atomic.Int32
	client := NewOpenAICompatClient("http://ai.test/v1", "test-model", "", 0.6, time.Second, 64)
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		return jsonHTTPResponse(`{
			"choices": [{
				"message": {"role": "assistant", "content": "<think>Still reasoning"},
				"finish_reason": "stop"
			}]
		}`), nil
	})

	_, err := client.chatCompletion(context.Background(), []oaiMessage{
		{Role: "user", Content: "Question"},
	})
	if !errors.Is(err, ErrRunawayReasoning) {
		t.Fatalf("got error %v, want ErrRunawayReasoning", err)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("reasoning-only response should retry once: got %d calls, want 2", got)
	}
}

func TestChatCompletionRetriesEmptyResponseOnce(t *testing.T) {
	var calls atomic.Int32
	client := NewOpenAICompatClient("http://ai.test/v1", "test-model", "", 0.6, time.Second, 64)
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		call := calls.Add(1)
		if call == 2 {
			return jsonHTTPResponse(`{
				"choices": [{
					"message": {"role": "assistant", "content": "Final answer."},
					"finish_reason": "stop"
				}]
			}`), nil
		}
		return jsonHTTPResponse(`{
			"choices": [{
				"message": {"role": "assistant", "content": "<think>Unclosed reasoning"},
				"finish_reason": "stop"
			}]
		}`), nil
	})

	answer, err := client.chatCompletion(context.Background(), []oaiMessage{
		{Role: "user", Content: "Question"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer != "Final answer." {
		t.Errorf("answer: got %q, want %q", answer, "Final answer.")
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("empty response should retry once: got %d calls, want 2", got)
	}
}

func TestChatCompletionPreservesTimeoutClassification(t *testing.T) {
	client := NewOpenAICompatClient("http://ai.test/v1", "test-model", "", 0.6, 20*time.Millisecond, 64)
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})
	_, err := client.chatCompletion(context.Background(), []oaiMessage{
		{Role: "user", Content: "Question"},
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got error %v, want context.DeadlineExceeded", err)
	}
	if errors.Is(err, ErrUnavailable) {
		t.Fatalf("timeout must not be classified as ErrUnavailable: %v", err)
	}
}

func TestParseExtractionJSON(t *testing.T) {
	raw := "Here is the extraction:\n```json\n{" +
		"\"summary\": \"Patient reports headaches.\", " +
		"\"chiefComplaint\": \"headaches\", " +
		"\"diagnosis\": null, " +
		"\"allergies\": [\"penicillin\"], " +
		"\"medications\": [\"ibuprofen 400 mg\"], " +
		"\"followUp\": [\"review in two weeks\"]" +
		"}\n```"
	got, err := parseExtractionJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Summary != "Patient reports headaches." {
		t.Errorf("summary: got %q", got.Summary)
	}
	if got.ChiefComplaint != "headaches" {
		t.Errorf("chiefComplaint: got %q", got.ChiefComplaint)
	}
	if got.Diagnosis == nil || len(got.Diagnosis) != 0 {
		t.Errorf("nil diagnosis should normalize to empty array, got %#v", got.Diagnosis)
	}
	if len(got.Allergies) != 1 || got.Allergies[0] != "penicillin" {
		t.Errorf("allergies: got %#v", got.Allergies)
	}
	if len(got.Medications) != 1 || got.Medications[0] != "ibuprofen 400 mg" {
		t.Errorf("medications: got %#v", got.Medications)
	}
	// Fields absent from the model output must still come back as arrays.
	for name, arr := range map[string][]string{
		"history":     got.History,
		"symptoms":    got.Symptoms,
		"labTests":    got.LabTests,
		"actionItems": got.ActionItems,
	} {
		if arr == nil {
			t.Errorf("%s: absent field should normalize to empty array", name)
		}
	}

	if _, err := parseExtractionJSON("no json here at all"); err == nil {
		t.Error("expected error for non-JSON output")
	}
	if _, err := parseExtractionJSON(`{"chiefComplaint": "headaches"}`); err == nil {
		t.Error("expected error when summary is missing")
	}
}

func chatSystemMessage(t *testing.T, request oaiChatRequest) string {
	t.Helper()
	if len(request.Messages) == 0 || request.Messages[0].Role != "system" {
		t.Fatalf("expected a leading system message, got %#v", request.Messages)
	}
	return request.Messages[0].Content
}

func TestChatGroundsInExtractionWhenPresent(t *testing.T) {
	var gotRequest oaiChatRequest
	client := NewOpenAICompatClient("http://ai.test/v1", "test-model", "", 0.6, time.Second, 64)
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(req.Body).Decode(&gotRequest); err != nil {
			t.Errorf("decode request: %v", err)
		}
		return jsonHTTPResponse(`{
			"choices": [{
				"message": {"role": "assistant", "content": "Answer."},
				"finish_reason": "stop"
			}]
		}`), nil
	})

	extraction := &ExtractionResult{
		Summary:     "Patient reports headaches.",
		Medications: []string{"ibuprofen 400 mg"},
	}
	grounding := ChatGrounding{
		Transcript: "SHOULD-NOT-BE-SENT raw transcript",
		Extraction: extraction,
	}
	if _, err := client.Chat(context.Background(), grounding, nil, "What medication?"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	system := chatSystemMessage(t, gotRequest)
	if !strings.Contains(system, recordOpen) || !strings.Contains(system, "ibuprofen 400 mg") {
		t.Errorf("system message should embed the extraction record, got: %s", system)
	}
	if strings.Contains(system, "SHOULD-NOT-BE-SENT") {
		t.Error("transcript must not be sent when extraction grounding is used")
	}
}

func TestChatFallsBackToTranscriptGrounding(t *testing.T) {
	var gotRequest oaiChatRequest
	client := NewOpenAICompatClient("http://ai.test/v1", "test-model", "", 0.6, time.Second, 64)
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(req.Body).Decode(&gotRequest); err != nil {
			t.Errorf("decode request: %v", err)
		}
		return jsonHTTPResponse(`{
			"choices": [{
				"message": {"role": "assistant", "content": "Answer."},
				"finish_reason": "stop"
			}]
		}`), nil
	})

	grounding := ChatGrounding{Transcript: "Doctor: take ibuprofen 400 mg."}
	if _, err := client.Chat(context.Background(), grounding, nil, "What medication?"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	system := chatSystemMessage(t, gotRequest)
	if !strings.Contains(system, transcriptOpen) || !strings.Contains(system, "ibuprofen 400 mg") {
		t.Errorf("system message should embed the transcript, got: %s", system)
	}
}
