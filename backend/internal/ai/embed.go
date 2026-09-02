package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// EmbeddingDimensions is fixed by the chosen model, bge-small-en-v1.5
// (docs/adr/0002). Every vector(384) column in the schema depends on it.
const EmbeddingDimensions = 384

// bgeQueryPrefix is the instruction bge v1.5 models expect on short retrieval
// queries (passages are embedded without it).
const bgeQueryPrefix = "Represent this sentence for searching relevant passages: "

// Embedder computes text embeddings via any OpenAI-compatible /v1/embeddings
// endpoint — the same provider-agnostic pattern as the chat client, configured
// independently (EMBED_* vars) because the chat endpoint may be a hosted
// provider that serves no embedding models.
type Embedder struct {
	baseURL    string
	model      string
	apiKey     string
	httpClient *http.Client
}

// NewEmbedder creates an embedding client for an OpenAI-compatible endpoint.
func NewEmbedder(baseURL, model, apiKey string, timeout time.Duration) *Embedder {
	return &Embedder{
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// Model returns the configured embedding model name.
func (e *Embedder) Model() string { return e.model }

type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// EmbedPassages embeds a batch of passage texts (no query instruction).
func (e *Embedder) EmbedPassages(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	body, err := json.Marshal(embeddingRequest{Model: e.model, Input: texts})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding endpoint unreachable: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	var parsed embeddingResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("embedding response parse (HTTP %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(raw))
		if parsed.Error != nil {
			msg = parsed.Error.Message
		}
		return nil, fmt.Errorf("embedding endpoint HTTP %d: %s", resp.StatusCode, msg)
	}
	if len(parsed.Data) != len(texts) {
		return nil, fmt.Errorf("embedding count mismatch: sent %d texts, got %d vectors", len(texts), len(parsed.Data))
	}

	vectors := make([][]float32, len(texts))
	for _, d := range parsed.Data {
		if d.Index < 0 || d.Index >= len(texts) {
			return nil, fmt.Errorf("embedding index %d out of range", d.Index)
		}
		// Guards a misconfigured EMBED_MODEL: the schema is vector(384) and a
		// wrong-dimension model must fail loudly, not corrupt the table.
		if len(d.Embedding) != EmbeddingDimensions {
			return nil, fmt.Errorf("model %q returned %d dimensions, expected %d (bge-small-en-v1.5)", e.model, len(d.Embedding), EmbeddingDimensions)
		}
		vectors[d.Index] = d.Embedding
	}
	return vectors, nil
}

// EmbedQuery embeds a search query with the bge retrieval instruction prefix.
func (e *Embedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	vectors, err := e.EmbedPassages(ctx, []string{bgeQueryPrefix + query})
	if err != nil {
		return nil, err
	}
	return vectors[0], nil
}

// ExtractionText is the canonical serialization of an Extraction for
// embedding: labelled non-empty fields, one per line. Deterministic — the
// stored extraction_text column always shows exactly what the extraction
// vector was computed from.
func ExtractionText(r *ExtractionResult) string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	add := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			b.WriteString(label)
			b.WriteString(": ")
			b.WriteString(strings.TrimSpace(value))
			b.WriteString("\n")
		}
	}
	addList := func(label string, values []string) {
		add(label, strings.Join(values, "; "))
	}
	add("Summary", r.Summary)
	add("Chief Complaint", r.ChiefComplaint)
	addList("History", r.History)
	addList("Allergies", r.Allergies)
	addList("Medications", r.Medications)
	addList("Symptoms", r.Symptoms)
	addList("Diagnosis", r.Diagnosis)
	addList("Differential Diagnosis", r.DifferentialDiagnosis)
	addList("Treatment", r.Treatment)
	addList("Lab Tests", r.LabTests)
	addList("Procedures", r.Procedures)
	addList("Follow Up", r.FollowUp)
	addList("Risk Factors", r.RiskFactors)
	addList("Medical Terms", r.MedicalTerms)
	addList("Action Items", r.ActionItems)
	return strings.TrimRight(b.String(), "\n")
}

// Chunking parameters: bge-small accepts 512 tokens; ~280 English words stays
// comfortably under that, and the overlap keeps context across boundaries.
const (
	chunkTargetWords  = 280
	chunkOverlapWords = 50
)

// ChunkTranscript splits a transcript into overlapping chunks for per-passage
// embedding. Sentence-aware where punctuation exists; dictated transcripts
// often have none, so oversized "sentences" fall back to word windows.
func ChunkTranscript(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	var chunks []string
	var current []string // words of the chunk being built

	emit := func(carryOverlap bool) {
		if len(current) == 0 {
			return
		}
		chunks = append(chunks, strings.Join(current, " "))
		if carryOverlap && len(current) > chunkOverlapWords {
			current = append([]string{}, current[len(current)-chunkOverlapWords:]...)
		} else {
			current = nil
		}
	}

	for _, sentence := range splitSentences(text) {
		words := strings.Fields(sentence)

		// A single sentence longer than a whole chunk (common in dictation
		// with no punctuation): emit fixed word windows with overlap.
		if len(words) > chunkTargetWords {
			emit(false)
			for len(words) > chunkTargetWords {
				chunks = append(chunks, strings.Join(words[:chunkTargetWords], " "))
				words = words[chunkTargetWords-chunkOverlapWords:]
			}
			current = append([]string{}, words...) // tail already starts with overlap
			continue
		}

		if len(current)+len(words) > chunkTargetWords {
			emit(true)
		}
		current = append(current, words...)
	}
	// The trailing chunk is only worth emitting if it holds more than the
	// overlap seed (or is the only content).
	if len(current) > chunkOverlapWords || len(chunks) == 0 {
		emit(false)
	}
	return chunks
}

// splitSentences splits on sentence-ending punctuation followed by space, and
// on newlines. Returns the whole text as one "sentence" when unpunctuated.
func splitSentences(text string) []string {
	var sentences []string
	var b strings.Builder
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		b.WriteRune(runes[i])
		endOfSentence := (runes[i] == '.' || runes[i] == '!' || runes[i] == '?') &&
			(i+1 >= len(runes) || runes[i+1] == ' ' || runes[i+1] == '\n')
		if endOfSentence || runes[i] == '\n' {
			if s := strings.TrimSpace(b.String()); s != "" {
				sentences = append(sentences, s)
			}
			b.Reset()
		}
	}
	if s := strings.TrimSpace(b.String()); s != "" {
		sentences = append(sentences, s)
	}
	return sentences
}
