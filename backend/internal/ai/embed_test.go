package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestChunkTranscriptEmpty(t *testing.T) {
	if got := ChunkTranscript("   "); got != nil {
		t.Fatalf("expected nil for blank transcript, got %v", got)
	}
}

func TestChunkTranscriptShortSingleChunk(t *testing.T) {
	text := "Patient presents with dry cough. No fever today. Lungs clear on auscultation."
	chunks := ChunkTranscript(text)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != text {
		t.Fatalf("short transcript should be one unmodified chunk, got %q", chunks[0])
	}
}

func TestChunkTranscriptUnpunctuatedDictation(t *testing.T) {
	// Dictated transcripts often have no punctuation at all; a 1000-word
	// "sentence" must still be windowed into bounded chunks.
	words := make([]string, 1000)
	for i := range words {
		words[i] = "word"
	}
	chunks := ChunkTranscript(strings.Join(words, " "))
	if len(chunks) < 3 {
		t.Fatalf("expected multiple chunks for 1000 unpunctuated words, got %d", len(chunks))
	}
	for i, c := range chunks {
		if n := len(strings.Fields(c)); n > chunkTargetWords {
			t.Fatalf("chunk %d has %d words, exceeds target %d", i, n, chunkTargetWords)
		}
	}
}

func TestChunkTranscriptOverlap(t *testing.T) {
	// Build sentences of 10 words each; enough to span several chunks.
	var b strings.Builder
	for i := 0; i < 120; i++ {
		b.WriteString(strings.TrimSpace(strings.Repeat("token ", 9)))
		b.WriteString(" end. ")
	}
	chunks := ChunkTranscript(b.String())
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
	// Each later chunk must start with words carried over from its predecessor.
	prev := strings.Fields(chunks[0])
	next := strings.Fields(chunks[1])
	tail := strings.Join(prev[len(prev)-chunkOverlapWords:], " ")
	head := strings.Join(next[:chunkOverlapWords], " ")
	if tail != head {
		t.Fatalf("chunk 1 should start with the %d-word tail of chunk 0", chunkOverlapWords)
	}
}

func TestExtractionTextLabelsNonEmptyFieldsOnly(t *testing.T) {
	r := &ExtractionResult{
		Summary:        "Viral URTI, symptomatic care.",
		ChiefComplaint: "Dry cough",
		Diagnosis:      []string{"URTI"},
		Medications:    []string{}, // empty — must not appear
	}
	got := ExtractionText(r)
	for _, want := range []string{"Summary: Viral URTI", "Chief Complaint: Dry cough", "Diagnosis: URTI"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "Medications") {
		t.Errorf("empty field must be omitted, got %q", got)
	}
	if ExtractionText(nil) != "" {
		t.Error("nil extraction must serialize to empty string")
	}
}

func TestEmbedPassagesRejectsWrongDimensions(t *testing.T) {
	// A misconfigured EMBED_MODEL returning non-384 vectors must fail loudly
	// instead of writing incompatible vectors at the schema boundary.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{{"index": 0, "embedding": make([]float32, 768)}},
		}
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer srv.Close()

	e := NewEmbedder(srv.URL, "wrong-model", "", 5*time.Second)
	_, err := e.EmbedPassages(context.Background(), []string{"text"})
	if err == nil || !strings.Contains(err.Error(), "768 dimensions") {
		t.Fatalf("expected dimension mismatch error, got %v", err)
	}
}

func TestEmbedQueryAddsInstructionPrefix(t *testing.T) {
	var received embeddingRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received) //nolint:errcheck
		resp := map[string]any{
			"data": []map[string]any{{"index": 0, "embedding": make([]float32, EmbeddingDimensions)}},
		}
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer srv.Close()

	e := NewEmbedder(srv.URL, "bge", "", 5*time.Second)
	if _, err := e.EmbedQuery(context.Background(), "night cough"); err != nil {
		t.Fatal(err)
	}
	if len(received.Input) != 1 || !strings.HasPrefix(received.Input[0], bgeQueryPrefix) {
		t.Fatalf("query must carry the bge instruction prefix, got %q", received.Input)
	}
}
