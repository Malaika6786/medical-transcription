# Embeddings are 384-dim bge-small-en-v1.5 — the spec's "368" does not exist

The mentor's task list specifies an embedding model of "bge-1.5 368 dimentions". No BGE model has 368 dimensions; the v1.5 family is small=384, base=768, large=1024. We use `bge-small-en-v1.5` at **384**, served by Ollama through the same OpenAI-compatible endpoint pattern as the LLM (`AI_BASE_URL` + a new embed-model variable). The correction was flagged to the mentor on Slack before building.

## Consequences

- Every `vector(384)` column bakes in the dimension: swapping embedding models later means re-embedding all stored content, not just changing config.
- Embeddings are computed asynchronously after session save (columns nullable = pending; startup backfill retries NULLs and doubles as the import path). A save never fails because the embedder is down.
- Search queries chunk embeddings ∪ extraction embeddings. The whole-transcript embedding column is stored (spec compliance, future doc-level similarity) but deliberately not queried — it is redundant with its own chunks and would double-count sessions.
