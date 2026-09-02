// Package corti provides client implementations for Corti API integration
package corti

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"corti-backend/internal/utils"
)

// AsyncClient handles asynchronous transcription API calls
type AsyncClient struct {
	baseURL      string
	tenantName   string
	tokenManager *TokenManager
	httpClient   *http.Client
}

// NewAsyncClient creates a new AsyncClient instance
func NewAsyncClient(config *utils.Config, tokenManager *TokenManager) *AsyncClient {
	return &AsyncClient{
		baseURL:      config.CortiAPIBaseURL,
		tenantName:   config.CortiTenant,
		tokenManager: tokenManager,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// CreateInteraction creates a new interaction in the Corti system
// Reference: https://docs.corti.ai/api-reference/interactions/create-interaction
// Based on working example from Corti support:
// - encounter: required object with identifier, status, type, period.startedAt, title
// - patient: optional object with identifier, name
func (c *AsyncClient) CreateInteraction(req *CreateInteractionRequest) (*CreateInteractionResponse, error) {
	log.Println("Creating new interaction for transcription")

	// Build request exactly as per working example from Corti support
	now := time.Now()
	timestamp := now.UnixMilli()

	requestBody := map[string]interface{}{
		"encounter": map[string]interface{}{
			"identifier": fmt.Sprintf("encounter-%d", timestamp),
			"status":     "in-progress",
			"type":       "consultation",
			"period": map[string]string{
				"startedAt": now.UTC().Format(time.RFC3339),
			},
			"title": "Audio Upload - Transcription",
		},
		"patient": map[string]interface{}{
			"identifier": fmt.Sprintf("patient-%d", timestamp),
			"name":       "Test Patient",
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal interaction request: %w", err)
	}

	// POST /v2/interactions/
	url := fmt.Sprintf("%s/v2/interactions/", c.baseURL)
	log.Printf("Creating interaction at URL: %s", url)
	log.Printf("Request body: %s", string(jsonBody))

	respBody, err := c.doRequest("POST", url, jsonBody, "application/json")
	if err != nil {
		return nil, err
	}

	if len(respBody) == 0 {
		return nil, fmt.Errorf("empty response from create interaction endpoint")
	}

	var resp CreateInteractionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse interaction response: %w (body: %s)", err, string(respBody))
	}

	log.Printf("Created interaction: %s", resp.InteractionID)
	return &resp, nil
}

// UploadRecording uploads an audio file to an existing interaction
// Reference: https://docs.corti.ai/api-reference/recordings/upload-recording
// Based on working example from Corti support:
// - Content-Type: application/octet-stream (raw binary, NOT multipart)
// - POST /v2/interactions/{interactionId}/recordings/
func (c *AsyncClient) UploadRecording(interactionID, filename string, audioData []byte, contentType string) (*UploadRecordingResponse, error) {
	fileSizeMB := float64(len(audioData)) / (1024 * 1024)
	log.Printf("Uploading recording for interaction: %s (file: %s, size: %.2f MB)", interactionID, filename, fileSizeMB)

	// Check file size limit (150MB per Corti docs)
	if len(audioData) > 150*1024*1024 {
		return nil, fmt.Errorf("file size exceeds 150MB limit")
	}

	// POST /v2/interactions/{interactionId}/recordings/
	// Use application/octet-stream as per working example from Corti support
	url := fmt.Sprintf("%s/v2/interactions/%s/recordings/", c.baseURL, interactionID)
	log.Printf("Upload URL: %s", url)

	// Send raw binary data with application/octet-stream
	respBody, err := c.doRequest("POST", url, audioData, "application/octet-stream")
	if err != nil {
		return nil, err
	}

	// Parse response - must get recordingId
	if len(respBody) == 0 {
		return nil, fmt.Errorf("upload succeeded but response body is empty - cannot get recordingId")
	}

	var resp UploadRecordingResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse upload response: %w (body: %s)", err, string(respBody))
	}

	if resp.RecordingID == "" {
		return nil, fmt.Errorf("upload response missing recordingId (body: %s)", string(respBody))
	}

	resp.Status = "uploaded"
	log.Printf("Successfully uploaded recording: %s", resp.RecordingID)
	return &resp, nil
}

// TranscriptConfig holds configuration for transcript creation
type TranscriptConfig struct {
	RecordingID     string
	PrimaryLanguage string
	IsDictation     bool
	IsMultichannel  bool
	Diarize         bool
	Participants    []ParticipantConfig
}

// ParticipantConfig defines a participant for transcript creation
type ParticipantConfig struct {
	Channel int    `json:"channel"`
	Role    string `json:"role"`
}

// RequestTranscript triggers transcript generation for an interaction
// Reference: https://docs.corti.ai/api-reference/transcripts/create-transcript
// Based on working example from Corti support:
// - POST /v2/interactions/{interactionId}/transcripts/
// - Body: { "recordingId": "uuid", "primaryLanguage": "en" }
func (c *AsyncClient) RequestTranscript(interactionID string, req *RequestTranscriptRequest) (*RequestTranscriptResponse, error) {
	log.Printf("Requesting transcript for interaction: %s with recording: %s, diarization: %v", interactionID, req.RecordingID, req.Diarize)

	// Validate recordingId
	if req.RecordingID == "" {
		return nil, fmt.Errorf("recordingId is required")
	}

	// POST /v2/interactions/{interactionId}/transcripts/
	url := fmt.Sprintf("%s/v2/interactions/%s/transcripts/", c.baseURL, interactionID)
	log.Printf("Transcript request URL: %s", url)

	// Build request body per Corti API
	// - recordingId: required
	// - primaryLanguage: required
	// - isDiarization: enable speaker identification (mutually exclusive with isDictation)
	// - isDictation: enables spoken command processing (punctuation, formatting)
	transcriptReq := map[string]interface{}{
		"recordingId":     req.RecordingID,
		"primaryLanguage": "en",
	}

	// Diarization and dictation are mutually exclusive
	// When diarization is enabled, we identify speakers
	// When dictation is enabled, we process formatting commands
	if req.Diarize {
		transcriptReq["isDiarization"] = true
		transcriptReq["isDictation"] = false
		transcriptReq["isMultichannel"] = false // Single channel with multiple speakers
		transcriptReq["participants"] = []map[string]interface{}{
			{"channel": 0, "role": "multiple"}, // AI will identify multiple speakers on channel 0
		}
		log.Printf("Speaker diarization ENABLED - will identify different speakers")
	} else {
		transcriptReq["isDictation"] = true // Enable dictation mode for formatting
		transcriptReq["isDiarization"] = false
		log.Printf("Dictation mode ENABLED - formatting and punctuation active")
	}

	jsonBody, err := json.Marshal(transcriptReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transcript request: %w", err)
	}

	log.Printf("Transcript request payload: %s", string(jsonBody))

	respBody, headers, err := c.doRequestWithHeaders("POST", url, jsonBody, "application/json")
	if err != nil {
		return nil, err
	}

	// Build response
	resp := &RequestTranscriptResponse{
		Status: "processing",
	}

	// Check for Location header (async processing for longer audio files)
	if location := headers.Get("Location"); location != "" {
		log.Printf("Async processing - Location: %s", location)
		resp.TranscriptID = extractTranscriptIDFromLocation(location)
	}

	// Parse response body if present (sync completion for short audio)
	if len(respBody) > 0 {
		var transcriptResp struct {
			ID          string           `json:"id"`
			Metadata    interface{}      `json:"metadata"`
			Transcripts []TranscriptEntry `json:"transcripts"`
		}
		if err := json.Unmarshal(respBody, &transcriptResp); err == nil && transcriptResp.ID != "" {
			resp.TranscriptID = transcriptResp.ID
			resp.Status = "completed"
			// The full transcript is already in this response — store it so callers
			// can skip the separate GetTranscriptByID round-trip (avoids 403 on EU region)
			if len(transcriptResp.Transcripts) > 0 {
				resp.Segments = transcriptResp.Transcripts
				var textParts []string
				for _, seg := range transcriptResp.Transcripts {
					if seg.Text != "" {
						textParts = append(textParts, seg.Text)
					}
				}
				resp.Text = strings.Join(textParts, " ")
			}
		}
	}

	log.Printf("Transcript created: transcriptId=%s, status=%s", resp.TranscriptID, resp.Status)
	return resp, nil
}

// extractTranscriptIDFromLocation extracts transcript ID from Location header
func extractTranscriptIDFromLocation(location string) string {
	parts := strings.Split(strings.TrimSuffix(location, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// GetTranscript retrieves the transcript for an interaction
// Reference: https://docs.corti.ai/api-reference/transcripts/list-transcripts
// Reference: https://docs.corti.ai/api-reference/transcripts/get-transcript
func (c *AsyncClient) GetTranscript(interactionID string) (*TranscriptResult, error) {
	log.Printf("Getting transcript for interaction: %s", interactionID)

	// Step 1: List transcripts to get the transcript ID
	// GET /v2/interactions/{interaction_id}/transcripts
	listURL := fmt.Sprintf("%s/v2/interactions/%s/transcripts", c.baseURL, interactionID)

	respBody, err := c.doRequest("GET", listURL, nil, "")
	if err != nil {
		// 404 / 202 / 503 all mean "transcript not ready yet" — treat as still processing
		// rather than a fatal error so the caller keeps polling
		if apiErr, ok := err.(*APIError); ok {
			if apiErr.StatusCode == 404 || apiErr.StatusCode == 202 || apiErr.StatusCode == 503 {
				log.Printf("Transcript list returned %d for interaction %s — still processing", apiErr.StatusCode, interactionID)
				return &TranscriptResult{Status: "processing"}, nil
			}
		}
		return nil, err
	}

	if len(respBody) == 0 {
		return &TranscriptResult{Status: "processing"}, nil
	}

	// Parse list response
	var listResp struct {
		Transcripts []struct {
			ID               string `json:"id"`
			RecordingID      string `json:"recordingId"`
			TranscriptSample string `json:"transcriptSample"`
		} `json:"transcripts"`
	}

	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse transcript list response: %w (body: %s)", err, string(respBody))
	}

	if len(listResp.Transcripts) == 0 {
		return &TranscriptResult{Status: "processing"}, nil
	}

	// Get the first transcript ID
	transcriptID := listResp.Transcripts[0].ID
	log.Printf("Found transcript ID: %s, fetching full content...", transcriptID)

	// Step 2: Fetch full transcript content using the transcript ID
	// GET /v2/interactions/{interaction_id}/transcripts/{transcript_id}/
	return c.GetTranscriptByID(interactionID, transcriptID)
}

// GetTranscriptByID retrieves a specific transcript with full content
// Reference: https://docs.corti.ai/api-reference/transcripts/get-transcript
func (c *AsyncClient) GetTranscriptByID(interactionID, transcriptID string) (*TranscriptResult, error) {
	log.Printf("Getting full transcript %s for interaction: %s", transcriptID, interactionID)

	// GET /v2/interactions/{interaction_id}/transcripts/{transcript_id}  (no trailing slash — EU region returns 403 with one)
	url := fmt.Sprintf("%s/v2/interactions/%s/transcripts/%s", c.baseURL, interactionID, transcriptID)

	respBody, err := c.doRequest("GET", url, nil, "")
	if err != nil {
		return nil, err
	}

	if len(respBody) == 0 {
		return &TranscriptResult{
			TranscriptID: transcriptID,
			Status:       "processing",
		}, nil
	}

	// Log the response for debugging
	log.Printf("Full transcript response size: %d bytes", len(respBody))
	if len(respBody) < 2000 {
		log.Printf("Full transcript response: %s", string(respBody))
	}

	// Parse the response - Corti returns nested structure
	var resp struct {
		ID       string `json:"id"`
		Metadata struct {
			RecordingID     string `json:"recordingId"`
			PrimaryLanguage string `json:"primaryLanguage"`
		} `json:"metadata"`
		Transcripts []TranscriptEntry `json:"transcripts"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse transcript response: %w (body: %s)", err, string(respBody))
	}

	result := &TranscriptResult{
		TranscriptID: resp.ID,
		Status:       "completed",
		Segments:     resp.Transcripts,
	}

	// Build full text from all segments and log speaker info for debugging
	var textParts []string
	speakerIDCounts := make(map[int]int)
	for _, seg := range resp.Transcripts {
		if seg.Text != "" {
			textParts = append(textParts, seg.Text)
		}
		speakerIDCounts[seg.SpeakerID]++
	}
	result.Text = strings.Join(textParts, " ")

	// Log speaker ID distribution to debug diarization
	log.Printf("Speaker ID distribution in transcript: %v", speakerIDCounts)
	log.Printf("Retrieved full transcript: %s with %d segments, text length: %d chars",
		result.TranscriptID, len(result.Segments), len(result.Text))
	return result, nil
}

// GetTranscriptStatus checks transcript status
// Reference: https://docs.corti.ai/api-reference/transcripts/get-transcript
// PollTranscript polls for transcript completion
func (c *AsyncClient) PollTranscript(interactionID string, timeout, interval time.Duration) (*TranscriptResult, error) {
	log.Printf("Polling transcript for interaction: %s", interactionID)

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		result, err := c.GetTranscript(interactionID)
		if err != nil {
			return nil, err
		}

		switch TranscriptStatus(result.Status) {
		case TranscriptStatusCompleted:
			log.Printf("Transcript completed")
			return result, nil
		case TranscriptStatusFailed:
			return nil, fmt.Errorf("transcript generation failed")
		default:
			log.Printf("Transcript status: %s, waiting...", result.Status)
			time.Sleep(interval)
		}
	}

	return nil, fmt.Errorf("transcript polling timed out after %v", timeout)
}

// GetInteraction retrieves interaction details
// getContextForInteraction builds the document context by trying facts first (ambient sessions),
// falling back to transcript text (file upload sessions).
func (c *AsyncClient) getContextForInteraction(interactionID string) ([]map[string]interface{}, error) {
	facts, err := c.GetInteractionFacts(interactionID)
	if err == nil && len(facts) > 0 {
		log.Printf("Using %d facts as context for interaction: %s", len(facts), interactionID)
		return []map[string]interface{}{{"type": "facts", "data": facts}}, nil
	}

	log.Printf("Facts not available for interaction %s, falling back to transcript", interactionID)
	transcript, err := c.GetTranscript(interactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transcript: %w", err)
	}
	if transcript.Text == "" {
		return nil, fmt.Errorf("transcript text is empty, cannot generate document")
	}
	log.Printf("Using transcript with %d characters as context", len(transcript.Text))
	return []map[string]interface{}{{"type": "string", "data": transcript.Text}}, nil
}

// GenerateDocument requests document generation from a transcript
// Reference: https://docs.corti.ai/api-reference/documents/generate-document
// Required fields:
// - templateKey: template identifier (e.g., "corti-soap")
// - context: ARRAY of context objects (Facts, Transcript, or String type)
// - outputLanguage: BCP-47 language tag (e.g., "en")
func (c *AsyncClient) GenerateDocument(interactionID string, templateKey string) (*GenerateDocumentResponse, error) {
	log.Printf("Generating document for interaction: %s with template: %s", interactionID, templateKey)

	context, err := c.getContextForInteraction(interactionID)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]interface{}{
		"templateKey":    templateKey,
		"outputLanguage": "en",
		"context":        context,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// POST /v2/interactions/{interactionId}/documents/
	url := fmt.Sprintf("%s/v2/interactions/%s/documents/", c.baseURL, interactionID)
	log.Printf("Document generation URL: %s", url)
	log.Printf("Document generation payload length: %d bytes", len(jsonBody))

	respBody, headers, err := c.doRequestWithHeaders("POST", url, jsonBody, "application/json")
	if err != nil {
		return nil, err
	}

	resp := &GenerateDocumentResponse{
		Status: "processing",
	}

	// Check for Location header (async processing)
	if location := headers.Get("Location"); location != "" {
		log.Printf("Async document processing - Location: %s", location)
		// Extract document ID from location
		parts := strings.Split(strings.TrimSuffix(location, "/"), "/")
		if len(parts) > 0 {
			resp.DocumentID = parts[len(parts)-1]
		}
	}

	// Parse response body if present
	if len(respBody) > 0 {
		log.Printf("Document generation response: %s", string(respBody))
		var docResp struct {
			ID       string            `json:"id"`
			Status   string            `json:"status"`
			Sections []DocumentSection `json:"sections"`
		}
		if err := json.Unmarshal(respBody, &docResp); err == nil {
			if docResp.ID != "" {
				resp.DocumentID = docResp.ID
			}
			if docResp.Status != "" {
				resp.Status = docResp.Status
			}
		}
	}

	log.Printf("Document generation initiated: documentId=%s, status=%s", resp.DocumentID, resp.Status)
	return resp, nil
}

// GenerateDocumentWithVerbosity generates a document with configurable verbosity level
// This uses Corti's new template customization feature with writingStyleOverride
// Reference: https://docs.corti.ai/api-reference/documents/generate-document
func (c *AsyncClient) GenerateDocumentWithVerbosity(interactionID string, verbosity OutputVerbosity) (*GenerateDocumentResponse, error) {
	log.Printf("Generating document for interaction: %s with verbosity: %s", interactionID, verbosity)

	context, err := c.getContextForInteraction(interactionID)
	if err != nil {
		return nil, err
	}

	log.Printf("Generating document with verbosity: %s", verbosity)

	// Get verbosity presets
	presets, ok := VerbosityPresets[verbosity]
	if !ok {
		presets = VerbosityPresets[VerbosityConcise] // Default to concise
	}

	// Build sections based on verbosity
	sections := []map[string]interface{}{
		{
			"key":                            SectionSubjective,
			"nameOverride":                   "Subjective",
			"writingStyleOverride":           presets["subjective"],
			"formatRuleOverride":             getFormatRuleForVerbosity(verbosity, "subjective"),
			"additionalInstructionsOverride": getAdditionalInstructions(verbosity, "subjective"),
		},
	}

	// For detailed mode, add Past Medical History as a separate focus within Objective
	// The objective section will handle PMH with subheadings
	sections = append(sections, map[string]interface{}{
		"key":                            SectionObjective,
		"nameOverride":                   "Objective",
		"writingStyleOverride":           presets["objective"],
		"formatRuleOverride":             getFormatRuleForVerbosity(verbosity, "objective"),
		"additionalInstructionsOverride": getAdditionalInstructions(verbosity, "objective"),
	})

	sections = append(sections, map[string]interface{}{
		"key":                            SectionAssessment,
		"nameOverride":                   "Assessment",
		"writingStyleOverride":           presets["assessment"],
		"formatRuleOverride":             getFormatRuleForVerbosity(verbosity, "assessment"),
		"additionalInstructionsOverride": getAdditionalInstructions(verbosity, "assessment"),
	})

	sections = append(sections, map[string]interface{}{
		"key":                            SectionPlan,
		"nameOverride":                   "Plan",
		"writingStyleOverride":           presets["plan"],
		"formatRuleOverride":             getFormatRuleForVerbosity(verbosity, "plan"),
		"additionalInstructionsOverride": getAdditionalInstructions(verbosity, "plan"),
	})

	// Build custom template request with section overrides
	// Reference: Corti support guidance on template customization
	reqBody := map[string]interface{}{
		"context": context,
		"template": map[string]interface{}{
			"description":                    getVerbosityDescription(verbosity),
			"additionalInstructionsOverride": getGlobalAdditionalInstructions(verbosity),
			"sections":                       sections,
		},
		"outputLanguage":    "en",
		"disableGuardrails": false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// POST /v2/interactions/{interactionId}/documents/
	url := fmt.Sprintf("%s/v2/interactions/%s/documents/", c.baseURL, interactionID)
	log.Printf("Document generation URL: %s", url)
	log.Printf("Document generation with verbosity=%s, payload length: %d bytes", verbosity, len(jsonBody))

	respBody, headers, err := c.doRequestWithHeaders("POST", url, jsonBody, "application/json")
	if err != nil {
		return nil, err
	}

	resp := &GenerateDocumentResponse{
		Status: "processing",
	}

	// Check for Location header (async processing)
	if location := headers.Get("Location"); location != "" {
		log.Printf("Async document processing - Location: %s", location)
		parts := strings.Split(strings.TrimSuffix(location, "/"), "/")
		if len(parts) > 0 {
			resp.DocumentID = parts[len(parts)-1]
		}
	}

	// Parse response body if present
	if len(respBody) > 0 {
		log.Printf("Document generation response: %s", string(respBody))
		var docResp struct {
			ID       string            `json:"id"`
			Status   string            `json:"status"`
			Sections []DocumentSection `json:"sections"`
		}
		if err := json.Unmarshal(respBody, &docResp); err == nil {
			if docResp.ID != "" {
				resp.DocumentID = docResp.ID
			}
			if docResp.Status != "" {
				resp.Status = docResp.Status
			}
		}
	}

	log.Printf("Document generation initiated: documentId=%s, status=%s, verbosity=%s", resp.DocumentID, resp.Status, verbosity)
	return resp, nil
}

// GenerateSOAPWithOverrides generates a SOAP note with user-customizable section overrides
// This allows users to customize writing style, format rules, and additional instructions per section
func (c *AsyncClient) GenerateSOAPWithOverrides(interactionID string, overrides *SOAPSectionOverrides) (*GenerateDocumentResponse, error) {
	log.Printf("Generating SOAP document for interaction: %s with custom section overrides", interactionID)

	context, err := c.getContextForInteraction(interactionID)
	if err != nil {
		return nil, err
	}

	// Build sections with user-provided overrides
	sections := []map[string]interface{}{}

	// Subjective section
	subjSection := map[string]interface{}{
		"key":          SectionSubjective,
		"nameOverride": "Subjective",
	}
	if overrides.Subjective.WritingStyle != "" {
		subjSection["writingStyleOverride"] = overrides.Subjective.WritingStyle
	}
	if overrides.Subjective.FormatRule != "" {
		subjSection["formatRuleOverride"] = overrides.Subjective.FormatRule
	}
	if overrides.Subjective.AdditionalInstructions != "" {
		subjSection["additionalInstructionsOverride"] = overrides.Subjective.AdditionalInstructions
	}
	sections = append(sections, subjSection)

	// Objective section
	objSection := map[string]interface{}{
		"key":          SectionObjective,
		"nameOverride": "Objective",
	}
	if overrides.Objective.WritingStyle != "" {
		objSection["writingStyleOverride"] = overrides.Objective.WritingStyle
	}
	if overrides.Objective.FormatRule != "" {
		objSection["formatRuleOverride"] = overrides.Objective.FormatRule
	}
	if overrides.Objective.AdditionalInstructions != "" {
		objSection["additionalInstructionsOverride"] = overrides.Objective.AdditionalInstructions
	}
	sections = append(sections, objSection)

	// Assessment section
	assessSection := map[string]interface{}{
		"key":          SectionAssessment,
		"nameOverride": "Assessment",
	}
	if overrides.Assessment.WritingStyle != "" {
		assessSection["writingStyleOverride"] = overrides.Assessment.WritingStyle
	}
	if overrides.Assessment.FormatRule != "" {
		assessSection["formatRuleOverride"] = overrides.Assessment.FormatRule
	}
	if overrides.Assessment.AdditionalInstructions != "" {
		assessSection["additionalInstructionsOverride"] = overrides.Assessment.AdditionalInstructions
	}
	sections = append(sections, assessSection)

	// Plan section
	planSection := map[string]interface{}{
		"key":          SectionPlan,
		"nameOverride": "Plan",
	}
	if overrides.Plan.WritingStyle != "" {
		planSection["writingStyleOverride"] = overrides.Plan.WritingStyle
	}
	if overrides.Plan.FormatRule != "" {
		planSection["formatRuleOverride"] = overrides.Plan.FormatRule
	}
	if overrides.Plan.AdditionalInstructions != "" {
		planSection["additionalInstructionsOverride"] = overrides.Plan.AdditionalInstructions
	}
	sections = append(sections, planSection)

	// Build custom template request with section overrides
	reqBody := map[string]interface{}{
		"context": context,
		"template": map[string]interface{}{
			"description": "Clinical SOAP Note with Custom Section Overrides",
			"sections":    sections,
		},
		"outputLanguage":    "en",
		"disableGuardrails": false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// POST /v2/interactions/{interactionId}/documents/
	url := fmt.Sprintf("%s/v2/interactions/%s/documents/", c.baseURL, interactionID)
	log.Printf("Document generation URL: %s", url)
	log.Printf("SOAP generation with custom overrides, payload length: %d bytes", len(jsonBody))

	respBody, headers, err := c.doRequestWithHeaders("POST", url, jsonBody, "application/json")
	if err != nil {
		return nil, err
	}

	resp := &GenerateDocumentResponse{
		Status: "processing",
	}

	// Check for Location header (async processing)
	if location := headers.Get("Location"); location != "" {
		log.Printf("Async document processing - Location: %s", location)
		parts := strings.Split(strings.TrimSuffix(location, "/"), "/")
		if len(parts) > 0 {
			resp.DocumentID = parts[len(parts)-1]
		}
	}

	// Parse response body if present
	if len(respBody) > 0 {
		log.Printf("Document generation response: %s", string(respBody))
		var docResp struct {
			ID       string            `json:"id"`
			Status   string            `json:"status"`
			Sections []DocumentSection `json:"sections"`
		}
		if err := json.Unmarshal(respBody, &docResp); err == nil {
			if docResp.ID != "" {
				resp.DocumentID = docResp.ID
			}
			if docResp.Status != "" {
				resp.Status = docResp.Status
			}
		}
	}

	log.Printf("SOAP document generation initiated: documentId=%s, status=%s", resp.DocumentID, resp.Status)
	return resp, nil
}

// getVerbosityDescription returns a description based on verbosity level
func getVerbosityDescription(verbosity OutputVerbosity) string {
	switch verbosity {
	case VerbosityDetailed:
		return "Detailed Clinical SOAP Note - Comprehensive documentation with bullet points"
	case VerbosityStandard:
		return "Standard Clinical SOAP Note"
	default:
		return "Concise Clinical SOAP Note"
	}
}

// getGlobalAdditionalInstructions returns global instructions that apply to the entire document
func getGlobalAdditionalInstructions(verbosity OutputVerbosity) string {
	switch verbosity {
	case VerbosityDetailed:
		return `Generate a structured clinical note with bullet points. Follow these rules:
1. Use bullet points (starting with "- ") for all lists and multiple findings
2. Keep each bullet point focused on ONE distinct piece of information
3. Write in third person (e.g., "Reports the lump is painful" not "I have pain")
4. Include Past Medical History as a subheading within the Objective section if relevant
5. Be specific and detailed - capture all clinical information mentioned
6. For the Plan section, include all counselling, advice, recommendations, and follow-up discussed
7. Use clear, professional medical language
8. Do not use paragraph format for lists - use bullet points instead`
	case VerbosityStandard:
		return "Generate a well-structured clinical note. Use bullet points for lists and multiple items. Write in third person."
	default:
		return ""
	}
}

// GenerateGPLetter generates a detailed GP referral letter with summary
// This uses a custom template structure to produce formal GP letters
func (c *AsyncClient) GenerateGPLetter(interactionID string, config *GPLetterConfig) (*GenerateDocumentResponse, error) {
	log.Printf("Generating GP Letter for interaction: %s", interactionID)

	context, err := c.getContextForInteraction(interactionID)
	if err != nil {
		return nil, err
	}

	// Set defaults if not provided
	patientName := "[patient name]"
	if config != nil && config.PatientName != "" {
		patientName = config.PatientName
	}

	recipientDoctor := "[clinician name or title]"
	if config != nil && config.RecipientDoctor != "" {
		recipientDoctor = config.RecipientDoctor
	}

	senderDoctor := "[clinician name]"
	senderTitle := "General Practitioner"
	if config != nil && config.SenderDoctor != "" {
		senderDoctor = config.SenderDoctor
	}
	if config != nil && config.PracticeName != "" {
		senderTitle = config.PracticeName
	}

	// GP Letter Template Structure
	// This follows a formal format with prose paragraphs
	gpLetterTemplate := fmt.Sprintf(`Generate a formal GP referral letter following this EXACT structure. This is a formal letter - do NOT use bullet points, lists, sentence fragments, or shorthand. Write in full sentences and structure each paragraph to read as a cohesive narrative using professional medical language. Always refer to the patient by name or pronoun throughout the note. Never come up with your own patient details - use ONLY information from the transcript.

EXACT STRUCTURE TO FOLLOW:

[GP recipient details]
(If known, insert clinic name and address. Otherwise omit.)

Dear %s,

Re: %s, DOB: [date of birth if mentioned, otherwise omit DOB]

[Introductory statement acknowledging the referral]
Begin with a courteous expression of thanks to the referring clinician. Mention the patient's name and the context of referral if known. This sentence should be neutral and applicable across all specialties.

[Brief overview of the patient's demographics and presenting complaint]
Begin by referring to the patient by name. Include the patient's age, relevant background and occupation if mentioned, and a concise description of the presenting issue. Write in complete sentences. Only include information explicitly mentioned.

[Detailed description of the presenting complaint and relevant history]
Describe the symptoms in narrative form, always referring to the patient's name or pronoun. Include onset, progression, severity, associated symptoms, any previous episodes, and relevant past medical, surgical, or family history. Write in full sentences as one coherent paragraph. Only include information explicitly mentioned.

[Clinical findings on examination]
Write in a single paragraph using full sentences, referring to the patient by name or pronoun. Include relevant positive and negative clinical findings observed during the physical examination. Structure the paragraph in a logical flow from general appearance through system-specific findings. Do NOT use bullet points or listing format. Only include if explicitly mentioned.

[Summary of clinical reasoning and discussion with the patient]
Summarise the clinician's interpretation of the presentation and any discussion held with the patient. Use full sentences to cover diagnostic considerations, patient education, and treatment planning. Only include if explicitly mentioned.

**Diagnoses:**
1. [First diagnosis]
2. [Second diagnosis if applicable]
(List each diagnosis as a separate numbered item. Include all active diagnoses mentioned. Only include if explicitly mentioned.)

**Investigations:**
1. [Investigation with date/year if provided]
2. [Investigation with date/year if provided]
(List each investigation as a separate numbered item. Include imaging, pathology, or other tests mentioned. Maintain chronological format. Only include if explicitly mentioned.)

**Medications:**
1. [Medication name, dose, frequency]
2. [Medication name, dose, frequency]
(List each medication as a separate numbered item. Include name, dose, and frequency. If dosing is flexible, note the range. Only include if explicitly mentioned.)

Note: [Medication changes or weaning details if applicable]
(Include only if there has been a change in medication. Mention the date of change if provided. Otherwise omit this line.)

**Suggested Management Plan:**
1. [General lifestyle and self-management recommendations]
2. [Planned investigations or follow-up actions with timeframes]
3. ACTIONS FOR SECRETARY: [Administrative tasks if applicable, otherwise omit]

[Detailed description of planned management approach and follow-up plan]
Write a paragraph summarising the agreed next steps including medications, procedures, referrals, or discharge planning. Mention anticipated side effects or restrictions. Only include if explicitly mentioned.

Yours sincerely,

%s
%s`, recipientDoctor, patientName, senderDoctor, senderTitle)

	// Build custom template for GP Letter
	reqBody := map[string]interface{}{
		"context": context,
		"template": map[string]interface{}{
			"description":                    "GP Letter with Summary",
			"additionalInstructionsOverride": gpLetterTemplate,
			"sections": []map[string]interface{}{
				{
					"key":                            SectionSubjective,
					"nameOverride":                   "Letter Content",
					"writingStyleOverride":           "Write the complete GP letter as one cohesive document. Use formal prose paragraphs. Do NOT use bullet points except in the Diagnoses, Investigations, Medications, and Management Plan sections which use numbered lists.",
					"formatRuleOverride":             "Formal letter format with paragraphs. Numbered lists only for Diagnoses, Investigations, Medications, and Suggested Management Plan sections.",
					"additionalInstructionsOverride": gpLetterTemplate,
				},
			},
		},
		"outputLanguage":    "en",
		"disableGuardrails": false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v2/interactions/%s/documents/", c.baseURL, interactionID)
	log.Printf("GP Letter generation URL: %s", url)
	log.Printf("GP Letter payload length: %d bytes", len(jsonBody))

	respBody, headers, err := c.doRequestWithHeaders("POST", url, jsonBody, "application/json")
	if err != nil {
		return nil, err
	}

	resp := &GenerateDocumentResponse{
		Status: "processing",
	}

	if location := headers.Get("Location"); location != "" {
		log.Printf("Async GP letter processing - Location: %s", location)
		parts := strings.Split(strings.TrimSuffix(location, "/"), "/")
		if len(parts) > 0 {
			resp.DocumentID = parts[len(parts)-1]
		}
	}

	if len(respBody) > 0 {
		log.Printf("GP Letter generation response: %s", string(respBody))
		var docResp struct {
			ID       string            `json:"id"`
			Status   string            `json:"status"`
			Sections []DocumentSection `json:"sections"`
		}
		if err := json.Unmarshal(respBody, &docResp); err == nil {
			if docResp.ID != "" {
				resp.DocumentID = docResp.ID
			}
			if docResp.Status != "" {
				resp.Status = docResp.Status
			}
		}
	}

	log.Printf("GP Letter generation initiated: documentId=%s, status=%s", resp.DocumentID, resp.Status)
	return resp, nil
}

// ========================================
// Custom Template Document Generation
// Reference: Corti API runtime section assembly approach
// ========================================

// CustomTemplateConfig represents a custom template configuration loaded from storage
// This structure follows Corti's recommended format for runtime section assembly
type CustomTemplateConfig struct {
	Key               string          `json:"key"`
	Name              string          `json:"name"`
	Category          string          `json:"category"`
	Description       string          `json:"description"`
	DocumentationMode string          `json:"documentationMode,omitempty"` // e.g., "routed_parallel"
	OutputLanguage    string          `json:"outputLanguage"`
	Template          CustomTemplate  `json:"template"`
	LetterConfig      *GPLetterConfig `json:"-"` // Runtime config for letter placeholders
	IsActive          *bool           `json:"isActive,omitempty"`         // Whether template is active/published (default: true)
	CreatedAt         string          `json:"createdAt,omitempty"`
	UpdatedAt         string          `json:"updatedAt,omitempty"`
}

// GenerateDocumentWithCustomTemplate generates a document using a custom template configuration
// This method follows Corti's official runtime section assembly approach
// Reference: Corti API documentation for custom templates
func (c *AsyncClient) GenerateDocumentWithCustomTemplate(interactionID string, templateConfig *CustomTemplateConfig) (*GenerateDocumentResponse, error) {
	log.Printf("Generating document with custom template '%s' for interaction: %s", templateConfig.Name, interactionID)

	// First, try to get facts from the interaction
	// We use facts-based context as recommended by Corti for ambient/streaming sessions
	// For async file uploads, facts may not be available, so we fall back to transcript
	facts, err := c.GetInteractionFacts(interactionID)
	if err != nil {
		log.Printf("Warning: Could not get facts for interaction %s: %v. Falling back to transcript.", interactionID, err)
		return c.generateDocumentWithTranscriptContext(interactionID, templateConfig)
	}

	// If facts are empty (common for async file uploads), fall back to transcript context
	if len(facts) == 0 {
		log.Printf("No facts available for interaction %s. Falling back to transcript context.", interactionID)
		return c.generateDocumentWithTranscriptContext(interactionID, templateConfig)
	}

	log.Printf("Using %d facts for custom template document generation", len(facts))

	// Build the sections array with all override fields
	sections := make([]map[string]interface{}, len(templateConfig.Template.Sections))
	for i, section := range templateConfig.Template.Sections {
		sectionMap := map[string]interface{}{
			"key": section.Key,
		}

		// Apply all overrides if present
		if section.NameOverride != "" {
			sectionMap["nameOverride"] = c.replacePlaceholders(section.NameOverride, templateConfig.LetterConfig)
		}
		if section.ContentOverride != "" {
			sectionMap["contentOverride"] = c.replacePlaceholders(section.ContentOverride, templateConfig.LetterConfig)
		}
		if section.WritingStyleOverride != "" {
			sectionMap["writingStyleOverride"] = section.WritingStyleOverride
		}
		if section.FormatRuleOverride != "" {
			sectionMap["formatRuleOverride"] = section.FormatRuleOverride
		}
		if section.AdditionalInstructionsOverride != "" {
			sectionMap["additionalInstructionsOverride"] = c.replacePlaceholders(section.AdditionalInstructionsOverride, templateConfig.LetterConfig)
		}

		sections[i] = sectionMap
	}

	// Build the request body following Corti's official format
	reqBody := map[string]interface{}{
		"context": []map[string]interface{}{
			{
				"type": "facts",
				"data": facts,
			},
		},
		"outputLanguage": templateConfig.OutputLanguage,
		"name":           templateConfig.Name,
		"template": map[string]interface{}{
			"sections": sections,
		},
	}

	// Add documentation mode if specified
	if templateConfig.DocumentationMode != "" {
		reqBody["documentationMode"] = templateConfig.DocumentationMode
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom template request: %w", err)
	}

	url := fmt.Sprintf("%s/v2/interactions/%s/documents/", c.baseURL, interactionID)
	log.Printf("Custom template document generation URL: %s", url)
	log.Printf("Custom template: %s, sections: %d, payload: %d bytes", templateConfig.Name, len(sections), len(jsonBody))

	respBody, headers, err := c.doRequestWithHeaders("POST", url, jsonBody, "application/json")
	if err != nil {
		return nil, err
	}

	return c.parseDocumentResponse(respBody, headers, templateConfig.Name)
}

// generateDocumentWithTranscriptContext is a fallback when facts are not available
func (c *AsyncClient) generateDocumentWithTranscriptContext(interactionID string, templateConfig *CustomTemplateConfig) (*GenerateDocumentResponse, error) {
	log.Printf("Using transcript-based context for custom template '%s'", templateConfig.Name)

	// Get transcript
	transcript, err := c.GetTranscript(interactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transcript: %w", err)
	}

	if transcript.Text == "" {
		return nil, fmt.Errorf("transcript is empty, cannot generate document")
	}

	// Build sections
	sections := make([]map[string]interface{}, len(templateConfig.Template.Sections))
	for i, section := range templateConfig.Template.Sections {
		sectionMap := map[string]interface{}{
			"key": section.Key,
		}

		if section.NameOverride != "" {
			sectionMap["nameOverride"] = c.replacePlaceholders(section.NameOverride, templateConfig.LetterConfig)
		}
		if section.ContentOverride != "" {
			sectionMap["contentOverride"] = c.replacePlaceholders(section.ContentOverride, templateConfig.LetterConfig)
		}
		if section.WritingStyleOverride != "" {
			sectionMap["writingStyleOverride"] = section.WritingStyleOverride
		}
		if section.FormatRuleOverride != "" {
			sectionMap["formatRuleOverride"] = section.FormatRuleOverride
		}
		if section.AdditionalInstructionsOverride != "" {
			sectionMap["additionalInstructionsOverride"] = c.replacePlaceholders(section.AdditionalInstructionsOverride, templateConfig.LetterConfig)
		}

		sections[i] = sectionMap
	}

	// Build request with string context (transcript)
	reqBody := map[string]interface{}{
		"context": []map[string]interface{}{
			{
				"type": "string",
				"data": transcript.Text,
			},
		},
		"outputLanguage": templateConfig.OutputLanguage,
		"name":           templateConfig.Name,
		"template": map[string]interface{}{
			"sections": sections,
		},
	}

	if templateConfig.DocumentationMode != "" {
		reqBody["documentationMode"] = templateConfig.DocumentationMode
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v2/interactions/%s/documents/", c.baseURL, interactionID)
	log.Printf("Custom template (transcript context) URL: %s", url)

	respBody, headers, err := c.doRequestWithHeaders("POST", url, jsonBody, "application/json")
	if err != nil {
		return nil, err
	}

	return c.parseDocumentResponse(respBody, headers, templateConfig.Name)
}

// GenerateDocumentFromTranscriptText generates a document directly from transcript text
// This creates a temporary interaction, uses the transcript text directly, and generates the document
// Useful for testing templates without needing to upload audio files
func (c *AsyncClient) GenerateDocumentFromTranscriptText(transcriptText string, templateConfig *CustomTemplateConfig) (*GenerateDocumentResponse, error) {
	log.Printf("Generating document from transcript text using template '%s'", templateConfig.Name)

	if transcriptText == "" {
		return nil, fmt.Errorf("transcript text is empty")
	}

	// Create a new interaction for this document generation
	interactionReq := &CreateInteractionRequest{}
	interaction, err := c.CreateInteraction(interactionReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create interaction: %w", err)
	}

	log.Printf("Created temporary interaction: %s for transcript-based document generation", interaction.InteractionID)

	// Build sections with placeholder replacements
	sections := make([]map[string]interface{}, len(templateConfig.Template.Sections))
	for i, section := range templateConfig.Template.Sections {
		sectionMap := map[string]interface{}{
			"key": section.Key,
		}

		if section.NameOverride != "" {
			sectionMap["nameOverride"] = c.replacePlaceholders(section.NameOverride, templateConfig.LetterConfig)
		}
		if section.ContentOverride != "" {
			sectionMap["contentOverride"] = c.replacePlaceholders(section.ContentOverride, templateConfig.LetterConfig)
		}
		if section.WritingStyleOverride != "" {
			sectionMap["writingStyleOverride"] = section.WritingStyleOverride
		}
		if section.FormatRuleOverride != "" {
			sectionMap["formatRuleOverride"] = section.FormatRuleOverride
		}
		if section.AdditionalInstructionsOverride != "" {
			sectionMap["additionalInstructionsOverride"] = c.replacePlaceholders(section.AdditionalInstructionsOverride, templateConfig.LetterConfig)
		}

		sections[i] = sectionMap
	}

	// Build request with string context (transcript text)
	reqBody := map[string]interface{}{
		"context": []map[string]interface{}{
			{
				"type": "string",
				"data": transcriptText,
			},
		},
		"outputLanguage": templateConfig.OutputLanguage,
		"name":           templateConfig.Name,
		"template": map[string]interface{}{
			"sections": sections,
		},
	}

	if templateConfig.DocumentationMode != "" {
		reqBody["documentationMode"] = templateConfig.DocumentationMode
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v2/interactions/%s/documents/", c.baseURL, interaction.InteractionID)
	log.Printf("Generating document from transcript text, URL: %s", url)

	respBody, headers, err := c.doRequestWithHeaders("POST", url, jsonBody, "application/json")
	if err != nil {
		return nil, err
	}

	response, err := c.parseDocumentResponse(respBody, headers, templateConfig.Name)
	if err != nil {
		return nil, err
	}

	// Add the interaction ID to the response for reference
	response.InteractionID = interaction.InteractionID

	return response, nil
}

// GetInteractionFacts retrieves the extracted facts from an interaction
func (c *AsyncClient) GetInteractionFacts(interactionID string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/v2/interactions/%s/facts/", c.baseURL, interactionID)
	log.Printf("Getting facts for interaction: %s", interactionID)

	respBody, _, err := c.doRequestWithHeaders("GET", url, nil, "")
	if err != nil {
		return nil, err
	}

	var factsResponse struct {
		Facts []map[string]interface{} `json:"facts"`
	}

	if err := json.Unmarshal(respBody, &factsResponse); err != nil {
		// Try parsing as array directly
		var facts []map[string]interface{}
		if err2 := json.Unmarshal(respBody, &facts); err2 != nil {
			return nil, fmt.Errorf("failed to parse facts response: %w", err)
		}
		return facts, nil
	}

	log.Printf("Retrieved %d facts for interaction %s", len(factsResponse.Facts), interactionID)
	return factsResponse.Facts, nil
}

// replacePlaceholders replaces template placeholders with actual values
func (c *AsyncClient) replacePlaceholders(text string, config *GPLetterConfig) string {
	if config == nil {
		return text
	}

	result := text

	// Replace patient name placeholders
	if config.PatientName != "" {
		result = strings.ReplaceAll(result, "{{PATIENT_NAME}}", config.PatientName)
		result = strings.ReplaceAll(result, "[patient name]", config.PatientName)
		result = strings.ReplaceAll(result, "[Patient Name]", config.PatientName)
	}

	// Replace recipient doctor placeholders
	if config.RecipientDoctor != "" {
		result = strings.ReplaceAll(result, "{{RECIPIENT_DOCTOR}}", config.RecipientDoctor)
		result = strings.ReplaceAll(result, "[clinician name or title]", config.RecipientDoctor)
		result = strings.ReplaceAll(result, "[Recipient]", config.RecipientDoctor)
	}

	// Replace sender doctor placeholders
	if config.SenderDoctor != "" {
		result = strings.ReplaceAll(result, "{{SENDER_DOCTOR}}", config.SenderDoctor)
		result = strings.ReplaceAll(result, "[clinician name]", config.SenderDoctor)
		result = strings.ReplaceAll(result, "[Sender Name]", config.SenderDoctor)
	}

	// Replace sender title placeholders
	if config.PracticeName != "" {
		result = strings.ReplaceAll(result, "{{SENDER_TITLE}}", config.PracticeName)
		result = strings.ReplaceAll(result, "[clinician title]", config.PracticeName)
		result = strings.ReplaceAll(result, "[Sender Title]", config.PracticeName)
	}

	return result
}

// CortiTemplateSection represents a template section from the Corti API
// Reference: https://docs.corti.ai/api-reference/templates/list-template-sections
type CortiTemplateSection struct {
	Name                   string                        `json:"name"`
	AlternateName          string                        `json:"alternateName,omitempty"`
	Key                    string                        `json:"key"`
	Description            string                        `json:"description"`
	DefaultWritingStyle    CortiWritingStyle             `json:"defaultWritingStyle"`
	DefaultFormatRule      *CortiFormatRule              `json:"defaultFormatRule,omitempty"`
	AdditionalInstructions string                        `json:"additionalInstructions,omitempty"`
	Content                string                        `json:"content,omitempty"`
	DocumentationMode      string                        `json:"documentationMode,omitempty"`
	Type                   string                        `json:"type"`
	Translations           []CortiSectionTranslation     `json:"translations"`
	UpdatedAt              string                        `json:"updatedAt,omitempty"`
}

type CortiWritingStyle struct {
	Name string `json:"name"`
}

type CortiFormatRule struct {
	Name string `json:"name"`
}

type CortiSectionTranslation struct {
	LanguageID  string `json:"languageId"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// GetTemplateSections fetches the list of template sections from Corti API
// Reference: https://docs.corti.ai/api-reference/templates/list-template-sections
func (c *AsyncClient) GetTemplateSections(lang string) ([]CortiTemplateSection, error) {
	url := fmt.Sprintf("%s/v2/templateSections/", c.baseURL)
	if lang != "" {
		url = fmt.Sprintf("%s?lang=%s", url, lang)
	}

	log.Printf("Fetching template sections from Corti API: %s", url)

	respBody, _, err := c.doRequestWithHeaders("GET", url, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template sections: %w", err)
	}

	var response struct {
		Data []CortiTemplateSection `json:"data"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse template sections response: %w", err)
	}

	log.Printf("Fetched %d template sections from Corti API", len(response.Data))
	return response.Data, nil
}

// parseDocumentResponse parses the API response and extracts document info
func (c *AsyncClient) parseDocumentResponse(respBody []byte, headers http.Header, templateName string) (*GenerateDocumentResponse, error) {
	resp := &GenerateDocumentResponse{
		Status:       "processing",
		TemplateName: templateName,
	}

	// Check for Location header (async processing)
	if location := headers.Get("Location"); location != "" {
		log.Printf("Async document processing - Location: %s", location)
		parts := strings.Split(strings.TrimSuffix(location, "/"), "/")
		if len(parts) > 0 {
			resp.DocumentID = parts[len(parts)-1]
		}
	}

	// Parse response body if present
	if len(respBody) > 0 {
		var docResp struct {
			ID       string            `json:"id"`
			Status   string            `json:"status"`
			Sections []DocumentSection `json:"sections"`
		}
		if err := json.Unmarshal(respBody, &docResp); err == nil {
			if docResp.ID != "" {
				resp.DocumentID = docResp.ID
			}
			if docResp.Status != "" {
				resp.Status = docResp.Status
			}
			if len(docResp.Sections) > 0 {
				resp.Sections = docResp.Sections
			}
		}
	}

	log.Printf("Custom template '%s' document generation initiated: documentId=%s, status=%s", templateName, resp.DocumentID, resp.Status)
	return resp, nil
}

// getFormatRuleForVerbosity returns format rules based on verbosity and section
func getFormatRuleForVerbosity(verbosity OutputVerbosity, section string) string {
	switch verbosity {
	case VerbosityDetailed:
		// Detailed: All sections use bullet points
		switch section {
		case "subjective":
			return "Use bullet point format. Each point starts with '- ' (dash and space). One finding per bullet. Do not use paragraphs."
		case "objective":
			return "Use bullet point format with optional subheadings like 'Past Medical History:'. Each finding starts with '- ' (dash and space)."
		case "assessment":
			return "Use bullet point format if multiple diagnoses. Single diagnosis can be plain text."
		case "plan":
			return "Use bullet point format. Each action/recommendation starts with '- ' (dash and space). One action per bullet. Include specific details like dosages, timelines, and advice given."
		default:
			return "Bullet point format with '- ' prefix for each item"
		}
	case VerbosityStandard:
		switch section {
		case "subjective":
			return "Use bullet points for key findings. Start each with '- '"
		case "plan":
			return "Use bullet points for action items. Start each with '- '"
		default:
			return "Clear format, use bullet points where appropriate"
		}
	default: // Concise
		switch section {
		case "plan":
			return "Brief bulleted list with '- ' prefix"
		default:
			return "Concise paragraph format"
		}
	}
}

// getAdditionalInstructions returns additional instructions based on verbosity and section
func getAdditionalInstructions(verbosity OutputVerbosity, section string) string {
	switch verbosity {
	case VerbosityDetailed:
		switch section {
		case "subjective":
			return `IMPORTANT: Format as bullet points, each starting with "- ". Include:
- Reason for visit/chief complaint
- Symptom characteristics (what, when, where, how long)
- Changes since last visit if follow-up
- Impact on activities or quality of life
- Associated symptoms or factors
- Patient's concerns or questions
Do NOT use paragraphs. Each distinct finding should be a separate bullet.`
		case "objective":
			return `Format as bullet points starting with "- ". If there is past medical history, use a subheading "Past Medical History:" followed by bullet points. Include:
- Examination findings (what was observed/measured)
- Vital signs if mentioned
- Any test results or measurements
Keep each bullet focused on one finding.`
		case "assessment":
			return `State the diagnosis clearly. If multiple diagnoses, use bullet points. Include:
- Primary diagnosis/clinical impression
- Any secondary diagnoses
- Status of condition (new, ongoing, resolved)`
		case "plan":
			return `IMPORTANT: Format as bullet points, each starting with "- ". Each bullet should be ONE complete action or recommendation. Include:
- Counselling and education provided to patient
- Treatment recommendations with specific details
- Medications with dose, frequency, and duration if mentioned
- Procedures or referrals recommended
- Follow-up plans and timeline
- Safety netting advice (when to return, warning signs)
- Lifestyle recommendations
Be specific about what was discussed or recommended.`
		}
	case VerbosityStandard:
		switch section {
		case "subjective":
			return "Include relevant history and context. Use bullet points for multiple findings."
		case "objective":
			return "Include all significant findings. Use bullet points if multiple findings."
		case "assessment":
			return "Include primary diagnosis and key reasoning"
		case "plan":
			return "Include specific action items with details. Use bullet points for each action."
		}
	}
	return ""
}

// GetDocument retrieves a generated document
// Reference: https://docs.corti.ai/api-reference/documents/get-document
func (c *AsyncClient) GetDocument(interactionID, documentID string) (*DocumentResult, error) {
	log.Printf("Getting document %s for interaction: %s", documentID, interactionID)

	// GET /v2/interactions/{interactionId}/documents/{documentId}
	// Note: No trailing slash per API docs
	url := fmt.Sprintf("%s/v2/interactions/%s/documents/%s", c.baseURL, interactionID, documentID)

	respBody, err := c.doRequest("GET", url, nil, "")
	if err != nil {
		return nil, err
	}

	if len(respBody) == 0 {
		return &DocumentResult{
			DocumentID: documentID,
			Status:     "processing",
		}, nil
	}

	log.Printf("Document response length: %d bytes", len(respBody))
	if len(respBody) < 2000 {
		log.Printf("Document response: %s", string(respBody))
	}

	var resp DocumentResult
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse document response: %w (body: %s)", err, string(respBody))
	}

	if resp.DocumentID == "" {
		resp.DocumentID = documentID
	}

	// Map Text to Content for frontend compatibility
	for i := range resp.Sections {
		if resp.Sections[i].Content == "" && resp.Sections[i].Text != "" {
			resp.Sections[i].Content = resp.Sections[i].Text
		}
	}

	// Set status based on sections presence
	if len(resp.Sections) > 0 {
		resp.Status = "completed"
	} else {
		resp.Status = "processing"
	}

	log.Printf("Retrieved document: %s with %d sections", resp.DocumentID, len(resp.Sections))
	return &resp, nil
}

// ListDocuments lists all documents for an interaction
func (c *AsyncClient) ListDocuments(interactionID string) ([]DocumentResult, error) {
	log.Printf("Listing documents for interaction: %s", interactionID)

	// GET /v2/interactions/{interactionId}/documents/
	url := fmt.Sprintf("%s/v2/interactions/%s/documents/", c.baseURL, interactionID)

	respBody, err := c.doRequest("GET", url, nil, "")
	if err != nil {
		return nil, err
	}

	if len(respBody) == 0 {
		return []DocumentResult{}, nil
	}

	var listResp struct {
		Documents []DocumentResult `json:"documents"`
	}

	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse documents list: %w (body: %s)", err, string(respBody))
	}

	log.Printf("Found %d documents for interaction", len(listResp.Documents))
	return listResp.Documents, nil
}

// doRequest performs an authenticated HTTP request
func (c *AsyncClient) doRequest(method, url string, body []byte, contentType string) ([]byte, error) {
	respBody, _, err := c.doRequestWithHeaders(method, url, body, contentType)
	return respBody, err
}

// doRequestWithHeaders performs an authenticated HTTP request and returns headers
func (c *AsyncClient) doRequestWithHeaders(method, url string, body []byte, contentType string) ([]byte, http.Header, error) {
	token, err := c.tokenManager.GetToken()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get access token: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewBuffer(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Tenant-Name", c.tenantName)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Log request details
	log.Printf("API Request: %s %s", method, url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("API Response: %d %s (body: %d bytes)", resp.StatusCode, resp.Status, len(respBody))
	if len(respBody) > 0 && len(respBody) < 1000 {
		log.Printf("Response body: %s", string(respBody))
	}

	// Handle errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized {
			log.Println("Token expired, retrying with fresh token...")
			c.tokenManager.InvalidateToken()
			return c.doRequestWithNewToken(method, url, body, contentType)
		}

		return nil, nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	return respBody, resp.Header, nil
}

// doRequestWithNewToken retries with a fresh token
func (c *AsyncClient) doRequestWithNewToken(method, url string, body []byte, contentType string) ([]byte, http.Header, error) {
	token, err := c.tokenManager.GetToken()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get fresh token: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewBuffer(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Tenant-Name", c.tenantName)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	return respBody, resp.Header, nil
}
