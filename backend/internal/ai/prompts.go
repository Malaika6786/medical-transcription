package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// The transcript is always injected inside these delimiters in the system
// message, so user questions can never mix with it (prompt-injection guard +
// grounding anchor).
const transcriptOpen = "<transcript>"
const transcriptClose = "</transcript>"

// The stored extraction is injected inside these delimiters when it is the
// grounding source for chat, for the same reasons.
const recordOpen = "<consultation_record>"
const recordClose = "</consultation_record>"

const extractionSystemPrompt = `You are a clinical documentation assistant. You will be given the transcript of a doctor-patient conversation between the delimiters %s and %s.

Your task: extract clinical information from the conversation, using ONLY information stated in the transcript. Do not use outside medical knowledge to invent facts. This is documentation support, not medical advice.

Respond with a single JSON object and nothing else, using exactly these keys:
{
  "summary": "3-6 sentence summary of the consultation",
  "chiefComplaint": "the patient's main reason for the visit, as one short phrase",
  "history": ["relevant medical, family or social history explicitly mentioned"],
  "allergies": ["each allergy explicitly mentioned"],
  "medications": ["each medication mentioned, with dose and frequency exactly as spoken"],
  "symptoms": ["each symptom the patient reports"],
  "diagnosis": ["each diagnosis or clinical impression the doctor states"],
  "differentialDiagnosis": ["alternative or suspected diagnoses the doctor is still considering"],
  "treatment": ["each treatment or intervention advised or performed"],
  "labTests": ["each lab test or imaging study ordered, performed or discussed"],
  "procedures": ["each procedure performed or planned"],
  "followUp": ["follow-up plans, referrals and their timing"],
  "riskFactors": ["risk factors explicitly mentioned"],
  "medicalTerms": ["notable medical terms used in the conversation"],
  "actionItems": ["concrete next steps for the doctor or the patient"]
}

Rules:
- Use "" or empty arrays for anything not mentioned in the transcript. Never invent entries.
- Quote medication names, doses and measurements exactly as spoken.
- Write the summary in third-person clinical style ("The patient reports...", "The doctor advised..."). Never copy first-person speech verbatim.
- Write the summary in %s.

%s
%s
%s`

const chatSystemPrompt = `You are a clinical documentation assistant. Below, between the delimiters %s and %s, is the transcript of a doctor-patient conversation.

Answer the user's questions using ONLY the transcript. Rules:
- If the transcript does not contain the answer, say so explicitly (e.g. "The conversation does not mention any allergies."). Never guess.
- Do not use outside medical knowledge to add facts that are not in the transcript.
- Answer directly in your own words, in third person ("The doctor advised...", "The patient reported..."). Do not reply with raw transcript lines unless the user asks for an exact quote.
- Quote medication names, doses and measurements exactly as spoken.
- Be concise and direct. This is documentation support, not medical advice.

%s
%s
%s`

const chatFromRecordSystemPrompt = `You are a clinical documentation assistant. Below, between the delimiters %s and %s, is a structured record extracted from the transcript of a doctor-patient conversation.

Answer the user's questions using ONLY this record. Rules:
- If the record does not contain the answer, say so explicitly (e.g. "The extracted record does not include any allergies."). Never guess.
- Do not use outside medical knowledge to add facts that are not in the record.
- Answer directly in your own words, in third person ("The doctor advised...", "The patient reported...").
- Quote medication names, doses and measurements exactly as recorded.
- Be concise and direct. This is documentation support, not medical advice.

%s
%s
%s`

// languageName maps common language codes to a plain-English instruction
// target; anything unrecognized falls through as-is (the model copes).
func languageName(code string) string {
	switch strings.ToLower(strings.SplitN(code, "-", 2)[0]) {
	case "", "en":
		return "English"
	case "es":
		return "Spanish"
	case "de":
		return "German"
	case "fr":
		return "French"
	case "da":
		return "Danish"
	default:
		return code
	}
}

// buildExtractionMessages assembles the message list for an extraction call.
func buildExtractionMessages(transcript, language string) []oaiMessage {
	system := fmt.Sprintf(extractionSystemPrompt,
		transcriptOpen, transcriptClose,
		languageName(language),
		transcriptOpen, transcript, transcriptClose)
	return []oaiMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: "Extract the clinical information from this consultation as the specified JSON object."},
	}
}

// buildChatMessages assembles the message list for a Q&A call grounded in the
// raw transcript (fallback path when no extraction is supplied).
func buildChatMessages(transcript string, history []ChatMessage, question string) []oaiMessage {
	system := fmt.Sprintf(chatSystemPrompt,
		transcriptOpen, transcriptClose,
		transcriptOpen, transcript, transcriptClose)
	return assembleChatMessages(system, history, question)
}

// buildChatMessagesFromRecord assembles the message list for a Q&A call
// grounded in the stored extraction — the preferred path: the record carries
// the consultation's clinical content in far fewer prompt tokens than the
// transcript.
func buildChatMessagesFromRecord(extraction *ExtractionResult, history []ChatMessage, question string) []oaiMessage {
	record, err := json.MarshalIndent(extraction, "", "  ")
	if err != nil {
		// Cannot happen for a plain struct; keep a defensive fallback.
		record = []byte("{}")
	}
	system := fmt.Sprintf(chatFromRecordSystemPrompt,
		recordOpen, recordClose,
		recordOpen, string(record), recordClose)
	return assembleChatMessages(system, history, question)
}

// assembleChatMessages combines a system prompt, prior turns and the new
// question. History is capped to the most recent turns to keep prompts
// bounded, and non-chat roles are dropped.
func assembleChatMessages(system string, history []ChatMessage, question string) []oaiMessage {
	const maxHistoryTurns = 12
	if len(history) > maxHistoryTurns {
		history = history[len(history)-maxHistoryTurns:]
	}

	messages := make([]oaiMessage, 0, len(history)+2)
	messages = append(messages, oaiMessage{Role: "system", Content: system})
	for _, m := range history {
		role := m.Role
		if role != "user" && role != "assistant" {
			continue
		}
		messages = append(messages, oaiMessage{Role: role, Content: m.Content})
	}
	messages = append(messages, oaiMessage{Role: "user", Content: question})
	return messages
}
