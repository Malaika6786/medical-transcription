// Package embedding runs the asynchronous embedding pipeline: session saves
// never wait on (or fail because of) the embedding model. Sessions whose
// embedded_at is NULL or older than updated_at are (re-)embedded in the
// background; the same loop backfills data imported by cmd/migrate-json.
package embedding

import (
	"context"
	"log"
	"time"

	"corti-backend/internal/ai"
	"corti-backend/internal/pgstore"
)

const (
	batchSize    = 8
	tickInterval = 60 * time.Second
)

// Worker owns the background embedding loop.
type Worker struct {
	store    *pgstore.Store
	embedder *ai.Embedder
	kick     chan struct{}
}

// NewWorker creates a Worker; call Run in a goroutine to start it.
func NewWorker(store *pgstore.Store, embedder *ai.Embedder) *Worker {
	return &Worker{
		store:    store,
		embedder: embedder,
		kick:     make(chan struct{}, 1),
	}
}

// Notify wakes the worker after a session save. Non-blocking; the pending
// query is the source of truth, so a dropped signal only means waiting for
// the next tick.
func (w *Worker) Notify(sessionID string) {
	select {
	case w.kick <- struct{}{}:
	default:
	}
}

// Run processes pending sessions until ctx is cancelled: once at startup
// (the backfill pass), then on every Notify and every tick.
func (w *Worker) Run(ctx context.Context) {
	log.Printf("embedding worker: started (model=%s, %d dims)", w.embedder.Model(), ai.EmbeddingDimensions)
	w.processPending(ctx)

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("embedding worker: stopped")
			return
		case <-w.kick:
			w.processPending(ctx)
		case <-ticker.C:
			w.processPending(ctx)
		}
	}
}

// processPending drains the pending set in batches. On any error it returns
// and waits for the next tick rather than hot-looping against a down endpoint.
func (w *Worker) processPending(ctx context.Context) {
	for {
		pending, err := w.store.PendingEmbeddingSessions(ctx, batchSize)
		if err != nil {
			log.Printf("embedding worker: query pending: %v", err)
			return
		}
		if len(pending) == 0 {
			return
		}
		for _, p := range pending {
			if ctx.Err() != nil {
				return
			}
			if err := w.embedSession(ctx, p); err != nil {
				log.Printf("embedding worker: session %s: %v", p.ID, err)
				return
			}
			log.Printf("embedding worker: embedded session %s", p.ID)
		}
	}
}

// embedSession computes and stores all vectors for one session in a single
// embedding call: transcript chunks, the doc-level transcript vector, and the
// extraction serialization.
func (w *Worker) embedSession(ctx context.Context, p pgstore.PendingSession) error {
	chunkTexts := ai.ChunkTranscript(p.Transcript)
	extractionText := ai.ExtractionText(p.Extraction)

	inputs := make([]string, 0, len(chunkTexts)+2)
	inputs = append(inputs, chunkTexts...)

	transcriptIdx := -1
	if p.Transcript != "" {
		// Doc-level vector: the model reads only its first 512 tokens of a
		// long transcript — acceptable, since search runs on the chunks.
		transcriptIdx = len(inputs)
		inputs = append(inputs, p.Transcript)
	}
	extractionIdx := -1
	if extractionText != "" {
		extractionIdx = len(inputs)
		inputs = append(inputs, extractionText)
	}

	if len(inputs) == 0 {
		// Nothing embeddable — mark done so the row stops showing up as pending.
		return w.store.StoreSessionEmbeddings(ctx, p.ID, nil, "", nil, nil)
	}

	vectors, err := w.embedder.EmbedPassages(ctx, inputs)
	if err != nil {
		return err
	}

	chunks := make([]pgstore.ChunkEmbedding, len(chunkTexts))
	for i, text := range chunkTexts {
		chunks[i] = pgstore.ChunkEmbedding{Index: i, Text: text, Vector: vectors[i]}
	}
	var transcriptVec, extractionVec []float32
	if transcriptIdx >= 0 {
		transcriptVec = vectors[transcriptIdx]
	}
	if extractionIdx >= 0 {
		extractionVec = vectors[extractionIdx]
	}
	return w.store.StoreSessionEmbeddings(ctx, p.ID, transcriptVec, extractionText, extractionVec, chunks)
}
