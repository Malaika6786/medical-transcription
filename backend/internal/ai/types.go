// Package ai provides the AI Assistance module: clinical information
// extraction and transcript-grounded Q&A via any OpenAI-compatible LLM
// endpoint (local Ollama, a self-hosted VM, Groq, ...). Nothing in this
// package touches the Corti API — the completed transcript text is the only
// handoff between the two modules.
package ai

// ChatMessage is a single turn in the grounded Q&A history.
type ChatMessage struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

// ExtractionResult is the structured clinical extraction produced from a
// completed consultation transcript. It is both the payload of
// POST /api/ai/summarize and the preferred grounding source for chat: the
// frontend stores it and sends it back with each question instead of the far
// larger raw transcript. Field names and JSON keys follow the team-agreed
// extraction schema (camelCase on the wire).
type ExtractionResult struct {
	Summary               string   `json:"summary"`
	ChiefComplaint        string   `json:"chiefComplaint"`
	History               []string `json:"history"`
	Allergies             []string `json:"allergies"`
	Medications           []string `json:"medications"`
	Symptoms              []string `json:"symptoms"`
	Diagnosis             []string `json:"diagnosis"`
	DifferentialDiagnosis []string `json:"differentialDiagnosis"`
	Treatment             []string `json:"treatment"`
	LabTests              []string `json:"labTests"`
	Procedures            []string `json:"procedures"`
	FollowUp              []string `json:"followUp"`
	RiskFactors           []string `json:"riskFactors"`
	MedicalTerms          []string `json:"medicalTerms"`
	ActionItems           []string `json:"actionItems"`
}

// listFields returns pointers to every array field so they can be handled
// uniformly (normalization, size accounting) without repeating the field list.
func (r *ExtractionResult) listFields() []*[]string {
	return []*[]string{
		&r.History, &r.Allergies, &r.Medications, &r.Symptoms,
		&r.Diagnosis, &r.DifferentialDiagnosis, &r.Treatment, &r.LabTests,
		&r.Procedures, &r.FollowUp, &r.RiskFactors, &r.MedicalTerms,
		&r.ActionItems,
	}
}

// normalize replaces nil arrays with empty ones so the API always returns
// (and prompts always render) arrays, never null.
func (r *ExtractionResult) normalize() {
	for _, f := range r.listFields() {
		if *f == nil {
			*f = []string{}
		}
	}
}

// ChatGrounding selects the grounding source for a chat call. Extraction is
// preferred when present — it carries the clinical content of the consultation
// in far fewer prompt tokens than the raw transcript. Transcript is the
// fallback for callers that have not run extraction.
type ChatGrounding struct {
	Transcript string
	Extraction *ExtractionResult
}
