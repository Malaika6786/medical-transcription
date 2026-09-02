// Package corti provides client implementations for Corti API integration
package corti

import "time"

// ========================================
// Authentication Types
// ========================================

// TokenRequest represents the OAuth2 token request payload
type TokenRequest struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Audience     string `json:"audience,omitempty"`
}

// TokenResponse represents the OAuth2 token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope,omitempty"`
}

// CachedToken represents a cached token with expiration tracking
type CachedToken struct {
	Token     string
	ExpiresAt time.Time
}

// ========================================
// Interaction Types
// ========================================

// CreateInteractionRequest represents the request to create a new interaction
// Reference: https://docs.corti.ai/api-reference/interactions/create-interaction
type CreateInteractionRequest struct {
	// Required nested objects per Corti API
	Patient   *PatientInfo   `json:"patient"`
	Encounter *EncounterInfo `json:"encounter"`
	// UseCase specifies the workflow type: "transcription", "ambient", etc.
	UseCase string `json:"useCase,omitempty"`
}

// PatientInfo contains patient details for the interaction
type PatientInfo struct {
	Identifier string `json:"identifier"`
	Name       string `json:"name,omitempty"`
	Gender     string `json:"gender,omitempty"`
	BirthDate  string `json:"birthDate,omitempty"`
}

// EncounterInfo contains encounter details for the interaction
type EncounterInfo struct {
	Identifier string           `json:"identifier"`
	Status     string           `json:"status,omitempty"` // planned, in-progress, on-hold, completed, cancelled, deleted
	Type       string           `json:"type,omitempty"`   // first_consultation, consultation, emergency, inpatient, outpatient
	Title      string           `json:"title,omitempty"`
	Period     *EncounterPeriod `json:"period,omitempty"`
}

// EncounterPeriod represents the time period of an encounter
type EncounterPeriod struct {
	StartedAt string `json:"startedAt,omitempty"`
	EndedAt   string `json:"endedAt,omitempty"`
}

// CreateInteractionResponse represents the response from creating an interaction
// Reference: https://docs.corti.ai/api-reference/interactions/create-interaction
type CreateInteractionResponse struct {
	InteractionID string `json:"interactionId"`
	WebSocketURL  string `json:"websocketUrl,omitempty"`
}

// Interaction represents a Corti interaction
type Interaction struct {
	InteractionID string            `json:"interaction_id"`
	Status        string            `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	ExternalID    string            `json:"external_id,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// ========================================
// Recording Types
// ========================================

// UploadRecordingResponse represents the response from uploading a recording
type UploadRecordingResponse struct {
	RecordingID string `json:"recordingId"` // API returns camelCase
	Status      string `json:"status"`
	Duration    int    `json:"duration,omitempty"`
}

// ========================================
// Transcription Types
// ========================================

// RequestTranscriptRequest represents the request to generate a transcript
type RequestTranscriptRequest struct {
	RecordingID string   `json:"recordingId,omitempty"` // Required: ID of the recording to transcribe
	Language    string   `json:"language,omitempty"`
	Features    []string `json:"features,omitempty"`
	Diarize     bool     `json:"diarize,omitempty"`
	Punctuate   bool     `json:"punctuate,omitempty"`
	Formatting  bool     `json:"formatting,omitempty"`
}

// RequestTranscriptResponse represents the response from requesting a transcript
type RequestTranscriptResponse struct {
	TranscriptID string           `json:"transcript_id"`
	Status       string           `json:"status"`
	// Populated when Corti returns the full transcript synchronously (short audio)
	Segments     []TranscriptEntry `json:"segments,omitempty"`
	Text         string            `json:"text,omitempty"`
}

// TranscriptResult represents the final transcript result
type TranscriptResult struct {
	TranscriptID string            `json:"transcript_id"`
	Status       string            `json:"status"`
	Text         string            `json:"text"`
	Segments     []TranscriptEntry `json:"segments,omitempty"` // Uses Corti API format
	Duration     float64           `json:"duration,omitempty"`
	Language     string            `json:"language,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	CompletedAt  time.Time         `json:"completed_at,omitempty"`
}

// TranscriptSegment represents a segment of the transcript (legacy format)
type TranscriptSegment struct {
	ID         string  `json:"id"`
	Text       string  `json:"text"`
	StartTime  float64 `json:"start_time"`
	EndTime    float64 `json:"end_time"`
	Speaker    string  `json:"speaker,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

// TranscriptEntry represents a transcript entry from the Corti API
// Reference: https://docs.corti.ai/api-reference/transcripts/get-transcript
type TranscriptEntry struct {
	Channel     int    `json:"channel"`     // The channel associated with this phrase/utterance
	Participant int    `json:"participant"` // The identifier of the participant
	SpeakerID   int    `json:"speakerId"`   // ID to tag an identified speaker
	Text        string `json:"text"`        // The spoken phrase or utterance
	Start       int    `json:"start"`       // Start time in milliseconds
	End         int    `json:"end"`         // End time in milliseconds
}

// TranscriptStatus represents possible transcript statuses
type TranscriptStatus string

const (
	TranscriptStatusPending    TranscriptStatus = "pending"
	TranscriptStatusProcessing TranscriptStatus = "processing"
	TranscriptStatusCompleted  TranscriptStatus = "completed"
	TranscriptStatusFailed     TranscriptStatus = "failed"
)

// ========================================
// Text Generation / Ambient Types
// ========================================

// GenerateDocumentRequest represents the request to generate a clinical document
// Reference: https://docs.corti.ai/api-reference/documents
type GenerateDocumentRequest struct {
	TemplateKey string `json:"templateKey"` // Required: one of corti-soap, corti-h-and-p, etc.
}

// GenerateDocumentResponse represents the response from document generation
type GenerateDocumentResponse struct {
	DocumentID    string            `json:"id"`
	Status        string            `json:"status"`
	InteractionID string            `json:"interactionId,omitempty"`
	TemplateName  string            `json:"templateName,omitempty"`
	Sections      []DocumentSection `json:"sections,omitempty"`
}

// DocumentResult represents the generated document
// Reference: https://docs.corti.ai/api-reference/documents/get-document
type DocumentResult struct {
	DocumentID     string            `json:"id"`
	Name           string            `json:"name,omitempty"`
	TemplateRef    string            `json:"templateRef,omitempty"`
	IsStream       bool              `json:"isStream,omitempty"`
	Sections       []DocumentSection `json:"sections,omitempty"`
	CreatedAt      time.Time         `json:"createdAt,omitempty"`
	UpdatedAt      time.Time         `json:"updatedAt,omitempty"`
	OutputLanguage string            `json:"outputLanguage,omitempty"`
	UsageInfo      *DocumentUsage    `json:"usageInfo,omitempty"`
	// For frontend compatibility
	Status string `json:"status,omitempty"`
}

// DocumentSection represents a section in the generated document
// Reference: https://docs.corti.ai/api-reference/documents/get-document
type DocumentSection struct {
	Key       string    `json:"key"`                 // Section key identifier
	Name      string    `json:"name"`                // Section name (e.g., "Subjective", "Objective")
	Text      string    `json:"text"`                // Section text content
	Sort      int       `json:"sort,omitempty"`      // Sort order
	CreatedAt time.Time `json:"createdAt,omitempty"` // When section was created
	UpdatedAt time.Time `json:"updatedAt,omitempty"` // When section was updated
	// For frontend compatibility (maps Text to Content)
	Content string `json:"content,omitempty"`
}

// DocumentUsage represents credits consumed for document generation
type DocumentUsage struct {
	CreditsConsumed int `json:"creditsConsumed"`
}

// Available template keys for document generation
const (
	TemplateSOAP               = "corti-soap"
	TemplateBriefClinicalNote  = "corti-brief-clinical-note"
	TemplateOutpatientVisit    = "corti-outpatient-visit-note"
	TemplateEmergencyNote      = "corti-emergency-note"
	TemplateNursingNote        = "corti-nursing-note"
	TemplateEmergencyResponse  = "corti-emergency-response-note"
	TemplateHistoryAndPhysical = "corti-h-and-p"
	TemplateReferral           = "corti-referral"
	TemplatePatientSummary     = "corti-patient-summary"
)

// ========================================
// Template Customization Types
// Reference: https://docs.corti.ai/api-reference/documents/generate-document
// New functionality for customizing template sections with override prompts
// ========================================

// OutputVerbosity defines the level of detail in generated documents
type OutputVerbosity string

const (
	VerbosityConcise  OutputVerbosity = "concise"  // Default Corti style - brief, fact-based
	VerbosityStandard OutputVerbosity = "standard" // Moderate detail
	VerbosityDetailed OutputVerbosity = "detailed" // Comprehensive narrative output
)

// CustomTemplateRequest represents a request to generate a document with custom template configuration
type CustomTemplateRequest struct {
	// Context contains the facts or transcript data for document generation
	Context []TemplateContext `json:"context"`
	// Template defines the custom template structure
	Template CustomTemplate `json:"template"`
	// OutputLanguage is the BCP-47 language code (e.g., "en")
	OutputLanguage string `json:"outputLanguage"`
	// Name is an optional custom name for the generated document
	Name string `json:"name,omitempty"`
	// DisableGuardrails disables AI safety guardrails (use with caution)
	DisableGuardrails bool `json:"disableGuardrails,omitempty"`
}

// TemplateContext represents a context item for document generation
type TemplateContext struct {
	// Type is the context type: "facts", "transcript", or "string"
	Type string `json:"type"`
	// Data contains the context data (structure depends on Type)
	Data interface{} `json:"data"`
}

// FactContext represents a fact item in the context
type FactContext struct {
	Text   string `json:"text"`
	Group  string `json:"group"`
	Source string `json:"source,omitempty"`
}

// CustomTemplate defines the template structure with section overrides
type CustomTemplate struct {
	// Description is a custom description for the document
	Description string `json:"description,omitempty"`
	// AdditionalInstructionsOverride applies to the entire document
	AdditionalInstructionsOverride string `json:"additionalInstructionsOverride,omitempty"`
	// Sections defines the sections to include and their customizations
	Sections []CustomTemplateSection `json:"sections"`
}

// CustomTemplateSection defines a section with override options
// Reference: Corti API runtime section assembly approach
// Available section keys: corti-subjective, corti-objective, corti-assessment, corti-plan,
// corti-referral, corti-patient-summary, and legacy keys like corti-hpi-legacy, etc.
type CustomTemplateSection struct {
	// Key is the section identifier (e.g., "corti-subjective", "corti-objective")
	Key string `json:"key"`
	// NameOverride overrides the display name of the section (e.g., "Patient History" instead of "Subjective")
	NameOverride string `json:"nameOverride,omitempty"`
	// ContentOverride controls which facts are included/excluded
	// Format: "Include: [what to include]. Exclude: [what to exclude]."
	ContentOverride string `json:"contentOverride,omitempty"`
	// WritingStyleOverride controls the tone (formal, conversational, clinical, etc.)
	WritingStyleOverride string `json:"writingStyleOverride,omitempty"`
	// FormatRuleOverride defines output formatting (numbered lists, paragraphs, bullet points, etc.)
	FormatRuleOverride string `json:"formatRuleOverride,omitempty"`
	// AdditionalInstructionsOverride adds custom instructions for this section
	AdditionalInstructionsOverride string `json:"additionalInstructionsOverride,omitempty"`
}

// Predefined section keys for SOAP template
const (
	SectionChiefComplaint    = "corti-chief-complaint-legacy"
	SectionSubjective        = "corti-subjective"
	SectionObjective         = "corti-objective"
	SectionAssessment        = "corti-assessment"
	SectionPlan              = "corti-plan"
	SectionAssessmentAndPlan = "corti-assessment-and-plan"
)

// Template types for document generation
type TemplateType string

const (
	TemplateTypeSOAP     TemplateType = "soap"
	TemplateTypeReferral TemplateType = "referral"
	TemplateTypeGPLetter TemplateType = "gp-letter-with-summary"
)

// GPLetterConfig holds configuration for GP letter generation
type GPLetterConfig struct {
	PatientName     string `json:"patientName,omitempty"`
	RecipientDoctor string `json:"recipientDoctor,omitempty"`
	SenderDoctor    string `json:"senderDoctor,omitempty"`
	PracticeName    string `json:"practiceName,omitempty"`
}

// SectionOverride represents customizable overrides for a single section
type SectionOverride struct {
	WritingStyle           string `json:"writingStyle,omitempty"`
	FormatRule             string `json:"formatRule,omitempty"`
	AdditionalInstructions string `json:"additionalInstructions,omitempty"`
}

// SOAPSectionOverrides contains user-customizable overrides for all SOAP sections
type SOAPSectionOverrides struct {
	Subjective SectionOverride `json:"subjective,omitempty"`
	Objective  SectionOverride `json:"objective,omitempty"`
	Assessment SectionOverride `json:"assessment,omitempty"`
	Plan       SectionOverride `json:"plan,omitempty"`
}

// VerbosityPresets maps verbosity levels to writing style prompts
var VerbosityPresets = map[OutputVerbosity]map[string]string{
	VerbosityConcise: {
		"subjective": "Concise summary in third person, focus on key clinical findings only",
		"objective":  "Brief objective findings, essential data only",
		"assessment": "Concise clinical assessment with primary diagnosis",
		"plan":       "Brief management plan with key action items",
	},
	VerbosityStandard: {
		"subjective": "Clear narrative in third person, include relevant history and context. Use bullet points for each key finding.",
		"objective":  "Complete examination findings with relevant diagnostic results. Use bullet points.",
		"assessment": "Clinical assessment with reasoning and differential considerations",
		"plan":       "Detailed management plan with specific instructions. Use bullet points for each action item.",
	},
	VerbosityDetailed: {
		"subjective": "Write in third person using bullet points. Each bullet should capture one distinct clinical finding. Include: chief complaint and reason for visit, symptom characteristics (onset, location, duration, triggers, relieving factors), impact on daily activities, and relevant history. Start each bullet with a dash (-) followed by the finding.",
		"objective":  "Write using bullet points for each finding. Include examination findings, vital signs, and any diagnostic results. Use subheadings like 'Past Medical History:' where relevant. Start each bullet with a dash (-).",
		"assessment": "State the diagnosis or clinical impression clearly. If multiple diagnoses, list each one. Be concise but complete.",
		"plan":       "Write using bullet points for each action item. Include: counselling provided, treatment recommendations, medications with dosages, follow-up instructions, patient education, and safety netting advice. Each bullet should be a complete actionable item. Start each bullet with a dash (-).",
	},
}

// ========================================
// WebSocket / Streaming Types
// ========================================

// StreamMessage represents a message in the WebSocket stream
type StreamMessage struct {
	Type      string       `json:"type"`
	Data      interface{}  `json:"data,omitempty"`
	Error     *StreamError `json:"error,omitempty"`
	Timestamp time.Time    `json:"timestamp,omitempty"`
}

// StreamError represents an error in the WebSocket stream
type StreamError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// StreamTranscriptEvent represents a transcript event from the stream
type StreamTranscriptEvent struct {
	Type       string  `json:"type"` // "partial" or "final"
	Text       string  `json:"text"`
	StartTime  float64 `json:"start_time,omitempty"`
	EndTime    float64 `json:"end_time,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	Speaker    string  `json:"speaker,omitempty"`
	IsFinal    bool    `json:"is_final"`
}

// StreamMedicalEvent represents a medical event detected in the stream
type StreamMedicalEvent struct {
	Type       string                 `json:"type"`
	Category   string                 `json:"category"`
	Value      string                 `json:"value"`
	Confidence float64                `json:"confidence,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// AmbientSessionState represents the state of an ambient session
type AmbientSessionState struct {
	InteractionID string                  `json:"interaction_id"`
	Status        string                  `json:"status"`
	StartedAt     time.Time               `json:"started_at"`
	Transcripts   []StreamTranscriptEvent `json:"transcripts,omitempty"`
	MedicalEvents []StreamMedicalEvent    `json:"medical_events,omitempty"`
}

// ========================================
// API Error Types
// ========================================

// APIError represents an error from the Corti API
type APIError struct {
	StatusCode int    `json:"status_code"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Details    string `json:"details,omitempty"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return e.Message
}

// ========================================
// WebSocket Message Types (Client -> Backend)
// ========================================

// ClientWSMessage represents a message from the frontend client
type ClientWSMessage struct {
	Type   string      `json:"type"`
	Action string      `json:"action,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

// ClientWSMessageType constants
const (
	WSMessageTypeAudio   = "audio"
	WSMessageTypeConfig  = "config"
	WSMessageTypeControl = "control"
	WSMessageTypeStop    = "stop"
	WSMessageTypeStart   = "start"
	WSMessageTypePing    = "ping"
)

// ========================================
// WebSocket Message Types (Backend -> Client)
// ========================================

// ServerWSMessage represents a message from the backend to frontend
type ServerWSMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// ServerWSMessageType constants
const (
	WSMessageTypeTranscript   = "transcript"
	WSMessageTypeMedicalEvent = "medical_event"
	WSMessageTypeFact         = "fact"
	WSMessageTypeStatus       = "status"
	WSMessageTypeError        = "error"
	WSMessageTypePong         = "pong"
	WSMessageTypeConnected    = "connected"
	WSMessageTypeDisconnected = "disconnected"
)

// ========================================
// Stream WebSocket Configuration
// Reference: https://docs.corti.ai/api-reference/stream
// ========================================

// StreamConfigMessage represents the configuration message sent to Corti WebSocket
// Format per https://docs.corti.ai/api-reference/stream#stream-configuration
type StreamConfigMessage struct {
	Type          string              `json:"type"`          // Must be "config"
	Configuration StreamConfiguration `json:"configuration"` // Configuration settings
}

// StreamConfiguration contains transcription and mode settings
type StreamConfiguration struct {
	Transcription StreamTranscriptionConfig `json:"transcription"`
	Mode          StreamModeConfig          `json:"mode"`
	// AudioFormat declares the raw audio format for clients that stream
	// headerless PCM (e.g. "audio/pcm; rate=16000; channels=1; bits=16").
	// Omitted for clients that send a self-describing container (WebM/Opus),
	// which Corti auto-detects from the stream itself.
	AudioFormat string `json:"audioFormat,omitempty"`
}

// StreamTranscriptionConfig contains transcription-specific settings
type StreamTranscriptionConfig struct {
	PrimaryLanguage string              `json:"primaryLanguage"` // Primary spoken language (e.g., "en")
	IsDiarization   bool                `json:"isDiarization"`   // Enable speaker diarization
	IsMultichannel  bool                `json:"isMultichannel"`  // Multiple audio channels
	Participants    []StreamParticipant `json:"participants"`    // Participant configuration
}

// StreamModeConfig specifies the operation mode
type StreamModeConfig struct {
	Type         string `json:"type"`         // "facts" or "transcript"
	OutputLocale string `json:"outputLocale"` // Output language locale (required for facts)
}

// StreamParticipant represents a participant in the stream
type StreamParticipant struct {
	Channel int    `json:"channel"` // Audio channel (0 for mono)
	Role    string `json:"role"`    // "clinician", "patient", "multiple", "other"
}

// CortiStreamMessage represents messages received from Corti WebSocket
// Reference: https://docs.corti.ai/api-reference/stream#handshake-responses
type CortiStreamMessage struct {
	Type    string            `json:"type"` // "transcript", "facts", "error", "CONFIG_ACCEPTED", "CONFIG_DENIED", "ENDED", "usage"
	Data    interface{}       `json:"data,omitempty"`
	Fact    []CortiFactData   `json:"fact,omitempty"`    // For type="facts"
	Error   *CortiStreamError `json:"error,omitempty"`   // For type="error"
	Credits float64           `json:"credits,omitempty"` // For type="usage"
	Reason  string            `json:"reason,omitempty"`  // For config errors
}

// CortiTranscriptData represents transcript data from the stream
// Reference: https://docs.corti.ai/api-reference/stream#transcripts-data-streams
type CortiTranscriptData struct {
	ID          string                     `json:"id"`
	Transcript  string                     `json:"transcript"`
	Final       bool                       `json:"final"`
	SpeakerID   int                        `json:"speakerId"` // -1 if diarization is off
	Participant CortiTranscriptParticipant `json:"participant"`
	Time        CortiTranscriptTime        `json:"time"`
}

// CortiTranscriptParticipant contains channel info
type CortiTranscriptParticipant struct {
	Channel int `json:"channel"`
}

// CortiTranscriptTime contains timing info
type CortiTranscriptTime struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// CortiFactData represents a fact from the stream
// Reference: https://docs.corti.ai/api-reference/stream#facts-data-streams
type CortiFactData struct {
	ID          string  `json:"id"`
	Text        string  `json:"text"`
	Group       string  `json:"group"` // e.g., "medical-history"
	GroupID     string  `json:"groupId"`
	IsDiscarded bool    `json:"isDiscarded"`
	Source      string  `json:"source"` // e.g., "core"
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   *string `json:"updatedAt,omitempty"`
}

// CortiStreamError represents an error from the stream
type CortiStreamError struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  int    `json:"status"`
	Details string `json:"details"`
	Doc     string `json:"doc"`
}

// CortiFactMessage represents a fact message for backward compatibility
type CortiFactMessage struct {
	ID         string  `json:"id,omitempty"`
	Category   string  `json:"category"`
	Value      string  `json:"value"`
	Text       string  `json:"text,omitempty"`
	Group      string  `json:"group,omitempty"`
	GroupID    string  `json:"groupId,omitempty"`
	Source     string  `json:"source,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}
