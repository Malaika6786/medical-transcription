// Package handlers provides HTTP request handlers for the Corti backend service
package handlers

import (
	"errors"
	"log"
	"sort"
	"time"

	"github.com/gofiber/fiber/v2"

	"corti-backend/internal/corti"
	"corti-backend/internal/pgstore"
	"corti-backend/internal/utils"
)

// TemplateStorage persists Corti custom templates (Postgres-backed in
// production; see internal/pgstore).
type TemplateStorage interface {
	GetTemplate(key string) (*corti.CustomTemplateConfig, error)
	ListTemplates() ([]*corti.CustomTemplateConfig, error)
	TemplateExists(key string) (bool, error)
	SaveTemplate(tpl *corti.CustomTemplateConfig) error
	DeleteTemplate(key string) error
}

// AsyncHandler handles async transcription requests
type AsyncHandler struct {
	asyncClient *corti.AsyncClient
	templates   TemplateStorage
}

// NewAsyncHandler creates a new AsyncHandler instance
func NewAsyncHandler(asyncClient *corti.AsyncClient, templates TemplateStorage) *AsyncHandler {
	return &AsyncHandler{
		asyncClient: asyncClient,
		templates:   templates,
	}
}

// UploadTranscriptionRequest represents the request for file upload transcription
type UploadTranscriptionRequest struct {
	Language   string            `json:"language"`
	ExternalID string            `json:"external_id"`
	Metadata   map[string]string `json:"metadata"`
}

// UploadTranscriptionResponse represents the response from file upload transcription
type UploadTranscriptionResponse struct {
	Success       bool                    `json:"success"`
	InteractionID string                  `json:"interaction_id"`
	RecordingID   string                  `json:"recording_id,omitempty"`
	TranscriptID  string                  `json:"transcript_id,omitempty"`
	Status        string                  `json:"status"`
	Message       string                  `json:"message,omitempty"`
	// Populated when Corti returns the full transcript synchronously (short audio)
	Transcript    *corti.TranscriptResult `json:"transcript,omitempty"`
}

// HandleUpload handles audio file upload and transcription initiation
// POST /api/transcribe/upload
// Supports multipart form with:
//   - audio: the audio file (required)
//   - diarization: "true" or "false" to enable speaker identification (optional, default: false)
func (h *AsyncHandler) HandleUpload(c *fiber.Ctx) error {
	requestID, _ := c.Locals("requestID").(string)
	log.Printf("[%s] Processing audio upload request", requestID)

	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to parse multipart form",
		})
	}

	// Get uploaded file
	files := form.File["audio"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "No audio file provided. Use 'audio' field name.",
		})
	}

	file := files[0]

	// Check for diarization option
	enableDiarization := false
	if diarizationValues, ok := form.Value["diarization"]; ok && len(diarizationValues) > 0 {
		enableDiarization = diarizationValues[0] == "true"
	}
	log.Printf("[%s] Diarization enabled: %v", requestID, enableDiarization)

	// Validate file
	if err := utils.ValidateAudioFile(file); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	if err := utils.ValidateAudioFileSize(file.Size); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	// Read file content
	audioData, err := utils.ReadAudioFile(file)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to read audio file",
		})
	}

	// Step 1: Create interaction
	// Based on working example from Corti support
	interactionReq := &corti.CreateInteractionRequest{}

	interaction, err := h.asyncClient.CreateInteraction(interactionReq)
	if err != nil {
		log.Printf("[%s] Failed to create interaction: %v", requestID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":     false,
			"error":       "Failed to create interaction",
			"corti_error": err.Error(),
		})
	}

	log.Printf("[%s] Created interaction: %s", requestID, interaction.InteractionID)

	// Step 2: Upload recording
	// Based on working example: Content-Type: application/octet-stream
	recording, err := h.asyncClient.UploadRecording(
		interaction.InteractionID,
		file.Filename,
		audioData,
		"application/octet-stream",
	)
	if err != nil {
		log.Printf("[%s] Failed to upload recording: %v", requestID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":        false,
			"error":          "Failed to upload recording",
			"corti_error":    err.Error(),
			"interaction_id": interaction.InteractionID,
		})
	}

	log.Printf("[%s] Uploaded recording: %s", requestID, recording.RecordingID)

	// Step 3: Create transcript with diarization option
	// Based on working example: { recordingId, primaryLanguage: "en" }
	transcriptReq := &corti.RequestTranscriptRequest{
		RecordingID: recording.RecordingID,
		Diarize:     enableDiarization,
	}

	log.Printf("[%s] Creating transcript with recordingId: %s, diarization: %v", requestID, recording.RecordingID, enableDiarization)
	transcript, err := h.asyncClient.RequestTranscript(interaction.InteractionID, transcriptReq)
	if err != nil {
		log.Printf("[%s] Failed to create transcript: %v", requestID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":        false,
			"error":          "Failed to create transcript",
			"corti_error":    err.Error(),
			"interaction_id": interaction.InteractionID,
			"recording_id":   recording.RecordingID,
		})
	}

	log.Printf("[%s] Transcript created: %s (status: %s)", requestID, transcript.TranscriptID, transcript.Status)

	message := "Transcription in progress. Poll for results using the interaction ID."
	if enableDiarization {
		message = "Transcription with speaker diarization in progress. Poll for results using the interaction ID."
	}

	resp := UploadTranscriptionResponse{
		Success:       true,
		InteractionID: interaction.InteractionID,
		RecordingID:   recording.RecordingID,
		TranscriptID:  transcript.TranscriptID,
		Status:        transcript.Status,
		Message:       message,
	}

	// When Corti returns the full transcript synchronously (short audio files),
	// include it directly so the frontend can skip polling and avoid the
	// GetTranscriptByID call that returns 403 on the EU region.
	if transcript.Status == "completed" && transcript.Text != "" {
		log.Printf("[%s] Transcript already complete — including in upload response", requestID)
		resp.Transcript = &corti.TranscriptResult{
			TranscriptID: transcript.TranscriptID,
			Status:       "completed",
			Text:         transcript.Text,
			Segments:     transcript.Segments,
		}
	}

	return c.Status(fiber.StatusAccepted).JSON(resp)
}

// TranscriptResponse represents the response for transcript retrieval
type TranscriptResponse struct {
	Success    bool                    `json:"success"`
	Status     string                  `json:"status"`
	Transcript *corti.TranscriptResult `json:"transcript,omitempty"`
	IsComplete bool                    `json:"is_complete"`
	Error      string                  `json:"error,omitempty"`
}

// HandleGetTranscript handles transcript retrieval by interaction ID
// GET /api/transcribe/:interactionId
func (h *AsyncHandler) HandleGetTranscript(c *fiber.Ctx) error {
	requestID, _ := c.Locals("requestID").(string)
	interactionID := c.Params("interactionId")

	if interactionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Interaction ID is required",
		})
	}

	log.Printf("[%s] Getting transcript for interaction: %s", requestID, interactionID)

	// Check if wait parameter is provided for long polling
	waitParam := c.Query("wait", "false")
	shouldWait := waitParam == "true"

	var result *corti.TranscriptResult
	var err error

	if shouldWait {
		// Poll with timeout
		timeout := 60 * time.Second
		interval := 2 * time.Second
		result, err = h.asyncClient.PollTranscript(interactionID, timeout, interval)
	} else {
		result, err = h.asyncClient.GetTranscript(interactionID)
	}

	if err != nil {
		log.Printf("[%s] Failed to get transcript: %v", requestID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":     false,
			"error":       "Failed to retrieve transcript",
			"corti_error": err.Error(),
		})
	}

	isComplete := corti.TranscriptStatus(result.Status) == corti.TranscriptStatusCompleted

	return c.JSON(TranscriptResponse{
		Success:    true,
		Status:     result.Status,
		Transcript: result,
		IsComplete: isComplete,
	})
}

// HandlePollTranscript handles long-polling for transcript completion
// GET /api/transcribe/:interactionId/poll
func (h *AsyncHandler) HandlePollTranscript(c *fiber.Ctx) error {
	requestID, _ := c.Locals("requestID").(string)
	interactionID := c.Params("interactionId")

	if interactionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Interaction ID is required",
		})
	}

	log.Printf("[%s] Long polling transcript for interaction: %s", requestID, interactionID)

	// Poll with 60 second timeout, 2 second interval
	result, err := h.asyncClient.PollTranscript(interactionID, 60*time.Second, 2*time.Second)
	if err != nil {
		log.Printf("[%s] Transcript polling failed: %v", requestID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(TranscriptResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	return c.JSON(TranscriptResponse{
		Success:    true,
		Status:     result.Status,
		Transcript: result,
		IsComplete: corti.TranscriptStatus(result.Status) == corti.TranscriptStatusCompleted,
	})
}

// DocumentResponse represents the response for document generation
type DocumentResponse struct {
	Success    bool                  `json:"success"`
	Status     string                `json:"status"`
	DocumentID string                `json:"document_id,omitempty"`
	Document   *corti.DocumentResult `json:"document,omitempty"`
	Error      string                `json:"error,omitempty"`
}

// SectionOverride represents customizable overrides for a single SOAP section
type SectionOverride struct {
	WritingStyle           string `json:"writingStyle,omitempty"`
	FormatRule             string `json:"formatRule,omitempty"`
	AdditionalInstructions string `json:"additionalInstructions,omitempty"`
}

// SOAPSectionOverrides contains overrides for all SOAP sections
type SOAPSectionOverrides struct {
	Subjective SectionOverride `json:"subjective,omitempty"`
	Objective  SectionOverride `json:"objective,omitempty"`
	Assessment SectionOverride `json:"assessment,omitempty"`
	Plan       SectionOverride `json:"plan,omitempty"`
}

// GenerateDocumentRequest represents the request body for document generation
type GenerateDocumentRequestBody struct {
	TemplateKey      string                `json:"templateKey"`
	Verbosity        string                `json:"verbosity,omitempty"`        // "concise", "standard", or "detailed"
	SectionOverrides *SOAPSectionOverrides `json:"sectionOverrides,omitempty"` // Custom section overrides for SOAP
	PatientName      string                `json:"patientName,omitempty"`      // For GP Letter template
	RecipientDoctor  string                `json:"recipientDoctor,omitempty"`  // For GP Letter template - "Dear Dr..."
	SenderDoctor     string                `json:"senderDoctor,omitempty"`     // For GP Letter template - sign-off name
	SenderTitle      string                `json:"senderTitle,omitempty"`      // For GP Letter template - e.g., "General Practitioner"
}

// HandleGenerateDocument handles clinical document generation
// POST /api/transcribe/:interactionId/document
// Supports optional verbosity parameter: "concise" (default), "standard", or "detailed"
func (h *AsyncHandler) HandleGenerateDocument(c *fiber.Ctx) error {
	requestID, _ := c.Locals("requestID").(string)
	interactionID := c.Params("interactionId")

	if interactionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Interaction ID is required",
		})
	}

	// Parse request body
	var req GenerateDocumentRequestBody
	if err := c.BodyParser(&req); err != nil {
		// Default to SOAP template if no body provided
		req.TemplateKey = corti.TemplateSOAP
	}

	// Validate template key
	if req.TemplateKey == "" {
		req.TemplateKey = corti.TemplateSOAP
	}

	log.Printf("[%s] Generating document for interaction: %s with template: %s, verbosity: %s", requestID, interactionID, req.TemplateKey, req.Verbosity)

	var result *corti.GenerateDocumentResponse
	var err error

	// First, check if this is a custom template
	customTemplate, loadErr := h.templates.GetTemplate(req.TemplateKey)
	if loadErr == nil && customTemplate != nil {
		log.Printf("[%s] Using custom template: %s", requestID, req.TemplateKey)

		// Set letter config for placeholder replacement
		customTemplate.LetterConfig = &corti.GPLetterConfig{
			PatientName:     req.PatientName,
			RecipientDoctor: req.RecipientDoctor,
			SenderDoctor:    req.SenderDoctor,
			PracticeName:    req.SenderTitle,
		}

		result, err = h.asyncClient.GenerateDocumentWithCustomTemplate(interactionID, customTemplate)
	} else if req.TemplateKey == "gp-letter-with-summary" {
		// Legacy GP Letter template (built-in)
		log.Printf("[%s] Using legacy GP Letter template", requestID)
		gpConfig := &corti.GPLetterConfig{
			PatientName:     req.PatientName,
			RecipientDoctor: req.RecipientDoctor,
			SenderDoctor:    req.SenderDoctor,
			PracticeName:    req.SenderTitle,
		}
		result, err = h.asyncClient.GenerateGPLetter(interactionID, gpConfig)
	} else if req.TemplateKey == corti.TemplateSOAP && req.SectionOverrides != nil {
		// SOAP template with custom section overrides from user
		log.Printf("[%s] Using SOAP template with custom section overrides", requestID)

		// Convert handler overrides to corti overrides
		cortiOverrides := &corti.SOAPSectionOverrides{
			Subjective: corti.SectionOverride{
				WritingStyle:           req.SectionOverrides.Subjective.WritingStyle,
				FormatRule:             req.SectionOverrides.Subjective.FormatRule,
				AdditionalInstructions: req.SectionOverrides.Subjective.AdditionalInstructions,
			},
			Objective: corti.SectionOverride{
				WritingStyle:           req.SectionOverrides.Objective.WritingStyle,
				FormatRule:             req.SectionOverrides.Objective.FormatRule,
				AdditionalInstructions: req.SectionOverrides.Objective.AdditionalInstructions,
			},
			Assessment: corti.SectionOverride{
				WritingStyle:           req.SectionOverrides.Assessment.WritingStyle,
				FormatRule:             req.SectionOverrides.Assessment.FormatRule,
				AdditionalInstructions: req.SectionOverrides.Assessment.AdditionalInstructions,
			},
			Plan: corti.SectionOverride{
				WritingStyle:           req.SectionOverrides.Plan.WritingStyle,
				FormatRule:             req.SectionOverrides.Plan.FormatRule,
				AdditionalInstructions: req.SectionOverrides.Plan.AdditionalInstructions,
			},
		}

		result, err = h.asyncClient.GenerateSOAPWithOverrides(interactionID, cortiOverrides)
	} else if req.Verbosity != "" && req.Verbosity != "concise" {
		// Fallback for verbosity without custom overrides (uses preset defaults)
		verbosity := corti.OutputVerbosity(req.Verbosity)
		// Validate verbosity value
		if verbosity != corti.VerbosityStandard && verbosity != corti.VerbosityDetailed {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid verbosity. Use 'concise', 'standard', or 'detailed'",
			})
		}
		log.Printf("[%s] Using custom template with verbosity: %s", requestID, verbosity)
		result, err = h.asyncClient.GenerateDocumentWithVerbosity(interactionID, verbosity)
	} else {
		// Default: use standard template
		result, err = h.asyncClient.GenerateDocument(interactionID, req.TemplateKey)
	}

	if err != nil {
		log.Printf("[%s] Failed to generate document: %v", requestID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":     false,
			"error":       "Failed to generate document",
			"corti_error": err.Error(),
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(DocumentResponse{
		Success:    true,
		Status:     result.Status,
		DocumentID: result.DocumentID,
	})
}

// HandleGetDocument retrieves a generated document
// GET /api/transcribe/:interactionId/document/:documentId
func (h *AsyncHandler) HandleGetDocument(c *fiber.Ctx) error {
	requestID, _ := c.Locals("requestID").(string)
	interactionID := c.Params("interactionId")
	documentID := c.Params("documentId")

	if interactionID == "" || documentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Interaction ID and Document ID are required",
		})
	}

	log.Printf("[%s] Getting document %s for interaction: %s", requestID, documentID, interactionID)

	result, err := h.asyncClient.GetDocument(interactionID, documentID)
	if err != nil {
		log.Printf("[%s] Failed to get document: %v", requestID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":     false,
			"error":       "Failed to get document",
			"corti_error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"document": result,
	})
}

// HandleListDocuments lists all documents for an interaction
// GET /api/transcribe/:interactionId/documents
func (h *AsyncHandler) HandleListDocuments(c *fiber.Ctx) error {
	requestID, _ := c.Locals("requestID").(string)
	interactionID := c.Params("interactionId")

	if interactionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Interaction ID is required",
		})
	}

	log.Printf("[%s] Listing documents for interaction: %s", requestID, interactionID)

	results, err := h.asyncClient.ListDocuments(interactionID)
	if err != nil {
		log.Printf("[%s] Failed to list documents: %v", requestID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":     false,
			"error":       "Failed to list documents",
			"corti_error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"documents": results,
	})
}

// GenerateFromTranscript generates a document directly from transcript text
// POST /api/transcribe/generate-from-transcript
// This is useful for testing templates without needing to upload audio files
func (h *AsyncHandler) GenerateFromTranscript(c *fiber.Ctx) error {
	requestID, _ := c.Locals("requestID").(string)
	log.Printf("[%s] Processing generate-from-transcript request", requestID)

	// Parse request body
	var req struct {
		Transcript       string `json:"transcript"`
		TemplateKey      string `json:"templateKey"`
		PatientName      string `json:"patientName,omitempty"`
		RecipientDoctor  string `json:"recipientDoctor,omitempty"`
		SenderDoctor     string `json:"senderDoctor,omitempty"`
		SenderTitle      string `json:"senderTitle,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	if req.Transcript == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Transcript text is required",
		})
	}

	if req.TemplateKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Template key is required",
		})
	}

	log.Printf("[%s] Generating document from transcript text using template: %s", requestID, req.TemplateKey)

	// Check if it's a custom template
	customTemplate, err := h.templates.GetTemplate(req.TemplateKey)
	if err != nil {
		// Not a custom template - use standard template
		log.Printf("[%s] Template '%s' is not a custom template, generating with standard template", requestID, req.TemplateKey)
		
		// For standard templates, we need to create an interaction and generate
		// This is a simplified flow for testing
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Direct transcript generation currently only supports custom templates. Standard templates require audio upload.",
		})
	}

	// Set letter config for placeholder replacement
	customTemplate.LetterConfig = &corti.GPLetterConfig{
		PatientName:     req.PatientName,
		RecipientDoctor: req.RecipientDoctor,
		SenderDoctor:    req.SenderDoctor,
		PracticeName:    req.SenderTitle,
	}

	// Generate document directly from transcript text
	response, err := h.asyncClient.GenerateDocumentFromTranscriptText(req.Transcript, customTemplate)
	if err != nil {
		log.Printf("[%s] Failed to generate document from transcript: %v", requestID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":     false,
			"error":       "Failed to generate document",
			"corti_error": err.Error(),
		})
	}

	log.Printf("[%s] Successfully generated document from transcript using template '%s'", requestID, req.TemplateKey)

	return c.JSON(fiber.Map{
		"success":       true,
		"interactionId": response.InteractionID,
		"document": fiber.Map{
			"name":     response.TemplateName,
			"sections": response.Sections,
		},
	})
}

// GetAvailableTemplates returns the available document templates
// GET /api/templates
func (h *AsyncHandler) GetAvailableTemplates(c *fiber.Ctx) error {
	// Built-in Corti templates (custom templates are loaded dynamically below)
	templates := []fiber.Map{
		{"key": corti.TemplateSOAP, "name": "SOAP Note", "description": "Subjective, Objective, Assessment, Plan"},
		{"key": corti.TemplateBriefClinicalNote, "name": "Brief Clinical Note", "description": "Quick clinical summary"},
		{"key": corti.TemplateHistoryAndPhysical, "name": "History & Physical", "description": "Initial patient evaluation"},
		{"key": corti.TemplateOutpatientVisit, "name": "Outpatient Visit Note", "description": "Office visit documentation"},
		{"key": corti.TemplateEmergencyNote, "name": "Emergency Note", "description": "Emergency department documentation"},
		{"key": corti.TemplateNursingNote, "name": "Nursing Note", "description": "Nursing assessment and care"},
		{"key": corti.TemplateEmergencyResponse, "name": "Emergency Response Note", "description": "EMS/First responder documentation"},
		{"key": corti.TemplateReferral, "name": "Referral", "description": "Specialist referral letter"},
		{"key": corti.TemplatePatientSummary, "name": "Patient Summary", "description": "Patient overview"},
	}

	// Verbosity options for template customization
	verbosityOptions := []fiber.Map{
		{
			"key":         "concise",
			"name":        "Concise",
			"description": "Brief, fact-based output focusing on key clinical findings (Corti default)",
		},
		{
			"key":         "standard",
			"name":        "Standard",
			"description": "Moderate detail with relevant history and context",
		},
		{
			"key":         "detailed",
			"name":        "Detailed",
			"description": "Comprehensive narrative output with full clinical documentation",
		},
	}

	// Load custom templates and add only ACTIVE ones to the list
	customTemplates, err := h.templates.ListTemplates()
	if err != nil {
		log.Printf("Could not list custom templates: %v", err)
	}
	for _, ct := range customTemplates {
		// Only include active templates (default to true if not set)
		isActive := ct.IsActive == nil || *ct.IsActive
		if isActive {
			templates = append(templates, fiber.Map{
				"key":         ct.Key,
				"name":        ct.Name,
				"description": ct.Description,
				"category":    ct.Category,
				"isCustom":    true,
			})
		}
	}

	// Sort templates alphabetically by name
	sort.Slice(templates, func(i, j int) bool {
		nameI, _ := templates[i]["name"].(string)
		nameJ, _ := templates[j]["name"].(string)
		return nameI < nameJ
	})

	return c.JSON(fiber.Map{
		"success":          true,
		"templates":        templates,
		"verbosityOptions": verbosityOptions,
	})
}

// ========================================
// Custom Template Management API Handlers
// (persisted in the corti_templates table — docs/adr/0001)
// ========================================

// GetCustomTemplates returns all custom templates with full details
// GET /api/templates/custom
func (h *AsyncHandler) GetCustomTemplates(c *fiber.Ctx) error {
	templates, err := h.templates.ListTemplates()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to load templates: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"templates": templates,
	})
}

// GetCustomTemplate returns a single custom template by key
// GET /api/templates/custom/:key
func (h *AsyncHandler) GetCustomTemplate(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Template key is required",
		})
	}

	template, err := h.templates.GetTemplate(key)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Template not found",
		})
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"template": template,
	})
}

// UpdateCustomTemplate updates a custom template
// PUT /api/templates/custom/:key
func (h *AsyncHandler) UpdateCustomTemplate(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Template key is required",
		})
	}

	// Parse the updated template
	var updatedTemplate corti.CustomTemplateConfig
	if err := c.BodyParser(&updatedTemplate); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid template data: " + err.Error(),
		})
	}

	// Ensure the key matches
	if updatedTemplate.Key != key {
		updatedTemplate.Key = key
	}

	// Only existing templates can be updated.
	exists, err := h.templates.TemplateExists(key)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to check template: " + err.Error(),
		})
	}
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Template not found",
		})
	}

	// Update timestamps
	updatedTemplate.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := h.templates.SaveTemplate(&updatedTemplate); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to save template: " + err.Error(),
		})
	}

	log.Printf("Updated custom template: %s", key)

	return c.JSON(fiber.Map{
		"success":  true,
		"message":  "Template updated successfully",
		"template": updatedTemplate,
	})
}

// CreateCustomTemplate creates a new custom template
// POST /api/templates/custom
func (h *AsyncHandler) CreateCustomTemplate(c *fiber.Ctx) error {
	var newTemplate corti.CustomTemplateConfig
	if err := c.BodyParser(&newTemplate); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid template data: " + err.Error(),
		})
	}

	if newTemplate.Key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Template key is required",
		})
	}

	// Check if template already exists
	exists, err := h.templates.TemplateExists(newTemplate.Key)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to check template: " + err.Error(),
		})
	}
	if exists {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"error":   "Template with this key already exists",
		})
	}

	// Set timestamps
	now := time.Now().Format(time.RFC3339)
	newTemplate.CreatedAt = now
	newTemplate.UpdatedAt = now

	if err := h.templates.SaveTemplate(&newTemplate); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to save template: " + err.Error(),
		})
	}

	log.Printf("Created custom template: %s", newTemplate.Key)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success":  true,
		"message":  "Template created successfully",
		"template": newTemplate,
	})
}

// DeleteCustomTemplate deletes a custom template
// DELETE /api/templates/custom/:key
func (h *AsyncHandler) DeleteCustomTemplate(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Template key is required",
		})
	}

	if err := h.templates.DeleteTemplate(key); err != nil {
		if errors.Is(err, pgstore.ErrTemplateNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "Template not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to delete template: " + err.Error(),
		})
	}

	log.Printf("Deleted custom template: %s", key)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Template deleted successfully",
	})
}

// GetAvailableSections returns the list of available Corti section keys
// GET /api/templates/sections
// Based on https://docs.corti.ai/textgen/templates-standard#standard-out-of-the-box-sections
func (h *AsyncHandler) GetAvailableSections(c *fiber.Ctx) error {
	sections := []fiber.Map{
		{"key": "corti-actions", "name": "Actions", "description": "Therapeutic actions performed"},
		{"key": "corti-actions-and-plan", "name": "Actions and Plan", "description": "Therapeutic and diagnostic procedures performed, and planned steps"},
		{"key": "corti-allergies", "name": "Allergies", "description": "Allergies and associated reactions"},
		{"key": "corti-assessment", "name": "Assessment", "description": "Assessment based on current clinical findings"},
		{"key": "corti-assessment-and-plan", "name": "Assessment and Plan", "description": "Diagnosis/evaluation and proposed treatment plan"},
		{"key": "corti-brief-clinical-note", "name": "Brief Clinical Note", "description": "Brief note summarizing encounter and key findings"},
		{"key": "corti-chief-complaint", "name": "Chief Complaint", "description": "Primary concern prompting the visit"},
		{"key": "corti-diagnoses", "name": "Diagnoses", "description": "List of diagnoses, past and current"},
		{"key": "corti-diagnostic-results", "name": "Diagnostic Results", "description": "Relevant lab and imaging results"},
		{"key": "corti-discharge-summary", "name": "Discharge Summary", "description": "Summary in discharge note format"},
		{"key": "corti-discussion-notes", "name": "Discussion Notes", "description": "Key discussion points of consultation"},
		{"key": "corti-family-history", "name": "Family History", "description": "Family medical conditions relevant to health risk"},
		{"key": "corti-hpi", "name": "History of Present Illness", "description": "Patient's description of present illness history"},
		{"key": "corti-interval-history", "name": "Interval History", "description": "Developments since last visit"},
		{"key": "corti-medical-decision-making", "name": "Medical Decision Making", "description": "Clinician's reasoning behind diagnoses and care"},
		{"key": "corti-medications", "name": "Medications", "description": "Record of patient's current medications"},
		{"key": "corti-objective", "name": "Objective", "description": "Physical examination and diagnostic data"},
		{"key": "corti-past-medical-history", "name": "Past Medical History", "description": "Past conditions, surgeries, and procedures"},
		{"key": "corti-patient-summary", "name": "Patient Summary", "description": "Summary for non-medical professional"},
		{"key": "corti-physical-exam-with-vitals", "name": "Physical Examination", "description": "Physical exam findings including vitals"},
		{"key": "corti-plan", "name": "Plan", "description": "Planned follow-up steps and actions"},
		{"key": "corti-referral", "name": "Referral", "description": "Referral to another medical professional"},
		{"key": "corti-review-of-systems", "name": "Review of Systems", "description": "Patient-reported symptoms by body system"},
		{"key": "corti-social-history", "name": "Social History", "description": "Social history including habits and risk factors"},
		{"key": "corti-subjective", "name": "Subjective", "description": "Detailed history, current and past"},
		{"key": "corti-vital-signs", "name": "Vital Signs", "description": "Patient vitals (BP, temp, pulse, resp rate, O2 sat)"},
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"sections": sections,
	})
}

// GetCortiTemplateSections fetches template sections directly from Corti API
// GET /api/templates/corti-sections
// This endpoint is for admin use only to view the official Corti section definitions
func (h *AsyncHandler) GetCortiTemplateSections(c *fiber.Ctx) error {
	requestID, _ := c.Locals("requestID").(string)
	log.Printf("[%s] Fetching Corti template sections from API", requestID)

	// Optional language filter
	lang := c.Query("lang", "")

	sections, err := h.asyncClient.GetTemplateSections(lang)
	if err != nil {
		log.Printf("[%s] Failed to fetch Corti template sections: %v", requestID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":     false,
			"error":       "Failed to fetch template sections from Corti API",
			"corti_error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"count":    len(sections),
		"sections": sections,
	})
}
