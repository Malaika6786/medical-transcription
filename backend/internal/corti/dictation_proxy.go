// Package corti provides the Dictation WebSocket proxy implementation
// Reference: https://docs.corti.ai/quickstart/dictation
package corti

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	fiberws "github.com/gofiber/websocket/v2"
	gorillaws "github.com/gorilla/websocket"

	"corti-backend/internal/utils"
)

// ========================================
// Dictation Types
// Reference: https://docs.corti.ai/quickstart/dictation
// ========================================

// DictationConfig represents the configuration message sent to Corti
// Reference: https://docs.corti.ai/quickstart/dictation#required-within-10s-of-opening-connection
type DictationConfig struct {
	PrimaryLanguage      string               `json:"primaryLanguage"`
	SpokenPunctuation    bool                 `json:"spokenPunctuation,omitempty"`
	AutomaticPunctuation bool                 `json:"automaticPunctuation,omitempty"`
	Commands             []DictationCommand   `json:"commands,omitempty"`
	Formatting           *DictationFormatting `json:"formatting,omitempty"`
}

// DictationCommand represents a voice command configuration
type DictationCommand struct {
	ID        string                     `json:"id"`
	Phrases   []string                   `json:"phrases"`
	Variables []DictationCommandVariable `json:"variables,omitempty"`
}

// DictationCommandVariable represents a variable in a command
type DictationCommandVariable struct {
	Key  string   `json:"key"`
	Type string   `json:"type"` // "enum"
	Enum []string `json:"enum,omitempty"`
}

// DictationFormatting represents formatting options
type DictationFormatting struct {
	Dates         string `json:"dates,omitempty"`         // "long_text", "short_text", etc.
	Times         string `json:"times,omitempty"`         // "h24", "h12"
	Numbers       string `json:"numbers,omitempty"`       // "numerals_above_nine", etc.
	Measurements  string `json:"measurements,omitempty"`  // "abbreviated", "full"
	NumericRanges string `json:"numericRanges,omitempty"` // "numerals", etc.
	Ordinals      string `json:"ordinals,omitempty"`      // "numerals", etc.
}

// DictationTranscriptData represents transcript data from the stream
// Reference: https://docs.corti.ai/quickstart/dictation#handle-responses
type DictationTranscriptData struct {
	Text              string  `json:"text"`
	RawTranscriptText string  `json:"rawTranscriptText,omitempty"`
	Start             float64 `json:"start"`
	End               float64 `json:"end"`
	IsFinal           bool    `json:"isFinal"`
}

// DictationCommandData represents command data from the stream
type DictationCommandData struct {
	ID        string            `json:"id"`
	Variables map[string]string `json:"variables,omitempty"`
}

// DictationMessage represents a message from Corti dictation WebSocket
type DictationMessage struct {
	Type    string          `json:"type"` // "transcript", "command", "usage", "error", "CONFIG_ACCEPTED", "ended", "flushed"
	Data    interface{}     `json:"data,omitempty"`
	Error   *DictationError `json:"error,omitempty"`
	Credits float64         `json:"credits,omitempty"`
}

// DictationError represents an error from the stream
type DictationError struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  int    `json:"status"`
	Details string `json:"details"`
}

// DictationServerMessage represents a message sent to the client
type DictationServerMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// Message types for dictation
const (
	DictationMsgTypeTranscript = "transcript"
	DictationMsgTypeCommand    = "command"
	DictationMsgTypeUsage      = "usage"
	DictationMsgTypeError      = "error"
	DictationMsgTypeStatus     = "status"
	DictationMsgTypePong       = "pong"
)

// ========================================
// Dictation Proxy Implementation
// ========================================

// DictationProxy manages dictation WebSocket connections
type DictationProxy struct {
	config       *utils.Config
	tokenManager *TokenManager
	sessions     map[string]*DictationSession
	mu           sync.RWMutex
}

// DictationSession represents an active dictation session
type DictationSession struct {
	SessionID string
	Language  string
	// AudioFormat, when non-empty, is declared to Corti as this session's raw
	// PCM format (mobile clients) — see ambient_proxy.go's AudioFormat for
	// the same pattern.
	AudioFormat string
	ClientConn  *fiberws.Conn
	CortiConn   *gorillaws.Conn
	IsActive    bool
	ConfigSent  bool
	stopChan    chan struct{}
	mu          sync.Mutex
	transcripts []DictationTranscriptData
	commands    []DictationCommandData
}

// NewDictationProxy creates a new dictation proxy instance
func NewDictationProxy(config *utils.Config, tokenManager *TokenManager) *DictationProxy {
	return &DictationProxy{
		config:       config,
		tokenManager: tokenManager,
		sessions:     make(map[string]*DictationSession),
	}
}

// CreateSession creates a new dictation session and connects to Corti
func (p *DictationProxy) CreateSession(sessionID, language, audioFormat string, clientConn *fiberws.Conn) (*DictationSession, error) {
	// Step 1: Get fresh access token
	token, err := p.tokenManager.GetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Step 2: Build WebSocket URL
	// Reference: https://docs.corti.ai/quickstart/dictation#initiate-real-time-bi-directional-communication
	// URL format: wss://api.{environment}.corti.app/audio-bridge/v2/transcribe?tenant-name={tenant}&token=Bearer%20{token}
	cortiWSURL := fmt.Sprintf("wss://api.%s.corti.app/audio-bridge/v2/transcribe", p.config.CortiEnvironment)

	// Add query parameters
	encodedToken := encodeBearerToken(token)
	cortiWSURL = fmt.Sprintf("%s?tenant-name=%s&token=%s", cortiWSURL, p.config.CortiTenant, encodedToken)

	log.Printf("Connecting to Corti Dictation WebSocket: %s", cortiWSURL[:len(cortiWSURL)-50]+"...")

	// Step 3: Connect to Corti WebSocket
	dialer := newCortiDialer()

	cortiConn, resp, err := dialer.Dial(cortiWSURL, nil)
	if err != nil {
		if resp != nil {
			log.Printf("Dictation WebSocket connection failed - Status: %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("failed to connect to Corti dictation WebSocket: %w", err)
	}

	log.Printf("Connected to Corti Dictation WebSocket - Status: %d", resp.StatusCode)

	// Step 4: Create session
	session := &DictationSession{
		SessionID:   sessionID,
		Language:    language,
		AudioFormat: audioFormat,
		ClientConn:  clientConn,
		CortiConn:   cortiConn,
		IsActive:    true,
		stopChan:    make(chan struct{}),
	}

	// Step 5: IMMEDIATELY send config after connection
	// CRITICAL: Must be within 10 seconds of WebSocket open
	// Reference: https://docs.corti.ai/quickstart/dictation#required-within-10s-of-opening-connection
	if err := session.sendDictationConfig(); err != nil {
		cortiConn.Close()
		return nil, fmt.Errorf("failed to send config: %w", err)
	}

	// Store session
	p.mu.Lock()
	p.sessions[sessionID] = session
	p.mu.Unlock()

	return session, nil
}

// HandleWebSocket handles the WebSocket connection for dictation
func (p *DictationProxy) HandleWebSocket(clientConn *fiberws.Conn, language, audioFormat string) error {
	// Generate session ID
	sessionID := fmt.Sprintf("dict-%d", time.Now().UnixNano())

	log.Printf("Starting dictation session: %s, language: %s", sessionID, language)

	// Create session and connect to Corti
	session, err := p.CreateSession(sessionID, language, audioFormat, clientConn)
	if err != nil {
		log.Printf("Failed to create dictation session: %v", err)
		return err
	}

	defer session.cleanup()
	defer p.removeSession(sessionID)

	// Config was already sent in CreateSession (immediately after connection)
	// Now wait for CONFIG_ACCEPTED before proceeding
	if err := session.waitForConfigAccepted(); err != nil {
		log.Printf("Config not accepted: %v", err)
		return err
	}

	// Send connected message to client after config is accepted
	session.sendToClient(DictationServerMessage{
		Type: "connected",
		Data: map[string]interface{}{
			"session_id": sessionID,
			"status":     "ready",
		},
		Timestamp: time.Now().UnixMilli(),
	})

	// Start bidirectional proxying
	errChan := make(chan error, 2)

	go func() {
		errChan <- session.proxyClientToCorti()
	}()

	go func() {
		errChan <- session.proxyCortiToClient()
	}()

	// Wait for either goroutine to finish
	err = <-errChan
	if err != nil {
		log.Printf("Dictation proxy error: %v", err)
	}

	return err
}

// sendDictationConfig sends the configuration message to Corti
// Reference: https://docs.corti.ai/api-reference/transcribe#configuration
// IMPORTANT: Config must be sent within 10 seconds of opening the WebSocket connection
//
// Per the API documentation, the config message format is:
//
//	{
//	  "type": "config",
//	  "configuration": {
//	    "primaryLanguage": "en",
//	    "spokenPunctuation": true,
//	    ...
//	  }
//	}
func (s *DictationSession) sendDictationConfig() error {
	// Extract 2-letter language code
	langCode := primaryLanguageCode(s.Language)

	// Build configuration per Corti API Reference:
	// https://docs.corti.ai/api-reference/transcribe#configuration
	// Must have "type": "config" wrapper with nested "configuration" object
	//
	// Punctuation options (mutually exclusive):
	// - spokenPunctuation: User says "period", "comma" etc. for punctuation
	// - automaticPunctuation: System auto-adds punctuation
	//
	// Commands: Voice commands for navigation, editing, and control
	// Reference: https://docs.corti.ai/stt/commands
	config := map[string]interface{}{
		"type": "config",
		"configuration": map[string]interface{}{
			"primaryLanguage": langCode,
			//"spokenPunctuation": true,
			"automaticPunctuation": true,
			"formatting": map[string]string{
				"dates":         "long_text",
				"times":         "h24",
				"numbers":       "numerals",
				"measurements":  "abbreviated",
				"numericRanges": "numerals",
				"ordinals":      "numerals",
			},
			// Voice Commands per https://docs.corti.ai/stt/commands
			// IMPORTANT: Commands must be spoken EXACTLY as defined in phrases
			// Use action verbs as first word per Corti best practices
			"commands": []map[string]interface{}{
				// Navigation command - move between clinical note sections
				// Multiple phrase patterns for flexibility
				{
					"id": "go_to_section",
					"phrases": []string{
						"go to {section_key}",
						"go to {section_key} section",
						"navigate to {section_key}",
						"move to {section_key}",
					},
					"variables": []map[string]interface{}{
						{
							"key":  "section_key",
							"type": "enum",
							"enum": []string{
								// Navigation
								"next",
								"previous",
								// GP Letter sections (formal letter format)
								"recipient",
								"recipient details",
								"address",
								"reference",
								"patient reference",
								"introduction",
								"overview",
								"patient overview",
								"history",
								"presenting history",
								"findings",
								"clinical findings",
								"examination",
								"reasoning",
								"clinical reasoning",
								"diagnoses",
								"diagnosis",
								"investigations",
								"medications",
								"management",
								"management plan",
								"follow up",
								"sign off",
								// SOAP sections
								"subjective",
								"objective",
								"assessment",
								"plan",
								// NHS sections
								"presenting complaint",
								"clinical impression",
							},
						},
					},
				},
				// Delete text command
				// Say: "delete that", "delete last word", "delete everything"
				{
					"id": "delete_range",
					"phrases": []string{
						"delete {delete_range}",
						"remove {delete_range}",
						"erase {delete_range}",
					},
					"variables": []map[string]interface{}{
						{
							"key":  "delete_range",
							"type": "enum",
							"enum": []string{"everything", "all", "the last word", "last word", "the last sentence", "last sentence", "that", "this"},
						},
					},
				},
				// Select text command
				// Say: "select all", "select last word"
				{
					"id": "select_range",
					"phrases": []string{
						"select {select_range}",
						"highlight {select_range}",
					},
					"variables": []map[string]interface{}{
						{
							"key":  "select_range",
							"type": "enum",
							"enum": []string{"all", "everything", "the last word", "last word", "the last sentence", "last sentence"},
						},
					},
				},
				// Insert template command - NHS-compliant templates
				// Say: "insert gp consultation template", "insert clinical note template"
				{
					"id": "insert_template",
					"phrases": []string{
						"insert {template_name} template",
						"insert {template_name}",
						"add {template_name} template",
						"use {template_name} template",
						"insert my {template_name} template",
						"insert my {template_name}",
						"start {template_name}",
					},
					"variables": []map[string]interface{}{
						{
							"key":  "template_name",
							"type": "enum",
							"enum": []string{
								// GP Letter with Summary (formal letter format)
								"gp letter with summary",
								"gp letter",
								"gp referral letter",
								// NHS-compliant templates
								"gp consultation",
								"clinical note",
								"discharge summary",
								"ae triage",
								"progress note",
								// Examination templates
								"physical exam",
								"normal exam",
								"review of systems",
								// Data templates
								"vital signs",
								"medication list",
								"allergy list",
								// SOAP (kept for compatibility)
								"soap note",
								"soap",
							},
						},
					},
				},
				// Undo/Redo commands
				// Say: "undo", "undo that", "redo"
				{
					"id":      "undo",
					"phrases": []string{"undo", "undo that", "undo last"},
				},
				{
					"id":      "redo",
					"phrases": []string{"redo", "redo that"},
				},
				// New line/paragraph commands
				// Say: "new line", "new paragraph"
				{
					"id":      "new_line",
					"phrases": []string{"new line", "next line", "line break"},
				},
				{
					"id":      "new_paragraph",
					"phrases": []string{"new paragraph", "next paragraph", "paragraph break"},
				},
				// New point/number commands for numbered lists
				// Say: "new point", "next number", "add point"
				{
					"id":      "new_point",
					"phrases": []string{"new point", "next point", "add point", "next number", "new number", "add number"},
				},
				// Bullet point command
				// Say: "bullet point", "new bullet", "add bullet"
				{
					"id":      "bullet_point",
					"phrases": []string{"bullet point", "new bullet", "add bullet", "bullet"},
				},
				// Clear all command
				// Say: "clear all", "start over"
				{
					"id":      "clear_all",
					"phrases": []string{"clear all", "clear everything", "start over", "clear text"},
				},
				// Stop/pause dictation
				// Say: "stop dictation", "stop listening"
				{
					"id":      "stop_dictation",
					"phrases": []string{"stop dictation", "pause dictation", "stop listening"},
				},
			},
		},
	}

	// AudioFormat, when set (mobile clients streaming raw PCM), must be
	// declared to Corti — see ambient_proxy.go's sendStreamConfig for the
	// same pattern and why it's needed.
	if s.AudioFormat != "" {
		if cfg, ok := config["configuration"].(map[string]interface{}); ok {
			cfg["audioFormat"] = s.AudioFormat
		}
	}

	configData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal dictation config: %w", err)
	}

	log.Printf("Sending dictation config to Corti: %s", string(configData))

	// Send using WriteMessage
	if err := s.CortiConn.WriteMessage(gorillaws.TextMessage, configData); err != nil {
		return fmt.Errorf("failed to send dictation config: %w", err)
	}

	s.ConfigSent = true
	log.Printf("Dictation config sent successfully")
	return nil
}

// waitForConfigAccepted waits for CONFIG_ACCEPTED response
func (s *DictationSession) waitForConfigAccepted() error {
	s.CortiConn.SetReadDeadline(time.Now().Add(15 * time.Second))
	defer s.CortiConn.SetReadDeadline(time.Time{})

	for {
		_, msg, err := s.CortiConn.ReadMessage()
		if err != nil {
			return fmt.Errorf("failed to read config response: %w", err)
		}

		log.Printf("Dictation config response: %s", string(msg))

		var response map[string]interface{}
		if err := json.Unmarshal(msg, &response); err != nil {
			continue
		}

		msgType, _ := response["type"].(string)
		switch msgType {
		case "CONFIG_ACCEPTED":
			log.Printf("Dictation config accepted by Corti")
			// Notify client
			s.sendToClient(DictationServerMessage{
				Type: DictationMsgTypeStatus,
				Data: map[string]interface{}{
					"type":   "CONFIG_ACCEPTED",
					"status": "ready",
				},
				Timestamp: time.Now().UnixMilli(),
			})
			return nil
		case "CONFIG_DENIED":
			reason, _ := response["reason"].(string)
			return fmt.Errorf("config denied: %s", reason)
		case "CONFIG_TIMEOUT":
			return fmt.Errorf("config timeout")
		}
	}
}

// proxyClientToCorti forwards messages from client to Corti
func (s *DictationSession) proxyClientToCorti() error {
	for {
		select {
		case <-s.stopChan:
			return nil
		default:
			msgType, msg, err := s.ClientConn.ReadMessage()
			if err != nil {
				if fiberws.IsCloseError(err, fiberws.CloseNormalClosure, fiberws.CloseGoingAway) {
					return nil
				}
				return fmt.Errorf("client read error: %w", err)
			}

			if msgType == fiberws.TextMessage {
				// Parse control message
				var clientMsg map[string]interface{}
				if err := json.Unmarshal(msg, &clientMsg); err == nil {
					msgTypeStr, _ := clientMsg["type"].(string)

					switch msgTypeStr {
					case "ping":
						s.sendToClient(DictationServerMessage{
							Type:      DictationMsgTypePong,
							Timestamp: time.Now().UnixMilli(),
						})
						continue
					case "flush":
						// Reference: https://docs.corti.ai/quickstart/dictation#force-results-to-be-returned-from-server
						log.Printf("Sending flush to Corti")
						if err := s.CortiConn.WriteMessage(gorillaws.TextMessage, msg); err != nil {
							return fmt.Errorf("corti write error: %w", err)
						}
						continue
					case "end", "stop":
						// Reference: https://docs.corti.ai/quickstart/dictation#sending-the-end-message
						log.Printf("Client requested end")
						endMsg := map[string]string{"type": "end"}
						endData, _ := json.Marshal(endMsg)
						if err := s.CortiConn.WriteMessage(gorillaws.TextMessage, endData); err != nil {
							return fmt.Errorf("corti write error: %w", err)
						}
						continue
					}
				}
			} else if msgType == fiberws.BinaryMessage {
				// Forward binary audio data to Corti
				// Reference: https://docs.corti.ai/quickstart/dictation#stream-audio-and-receive-transcripts
				if err := s.CortiConn.WriteMessage(gorillaws.BinaryMessage, msg); err != nil {
					return fmt.Errorf("corti write error: %w", err)
				}
			}
		}
	}
}

// proxyCortiToClient forwards messages from Corti to client
func (s *DictationSession) proxyCortiToClient() error {
	for {
		select {
		case <-s.stopChan:
			return nil
		default:
			msgType, msg, err := s.CortiConn.ReadMessage()
			if err != nil {
				if gorillaws.IsCloseError(err, gorillaws.CloseNormalClosure, gorillaws.CloseGoingAway) {
					return nil
				}
				return fmt.Errorf("corti read error: %w", err)
			}

			if msgType == gorillaws.TextMessage {
				log.Printf("Received from Corti dictation: %s", string(msg))

				// Parse message
				var cortiMsg map[string]interface{}
				if err := json.Unmarshal(msg, &cortiMsg); err != nil {
					log.Printf("Failed to parse Corti message: %v", err)
					continue
				}

				// Process and forward to client
				serverMsg := s.processCortiMessage(cortiMsg)
				if err := s.sendToClient(serverMsg); err != nil {
					return fmt.Errorf("client write error: %w", err)
				}

				// Check for session end
				if msgTypeStr, _ := cortiMsg["type"].(string); msgTypeStr == "ended" {
					log.Printf("Dictation session ended by Corti")
					return nil
				}
			}
		}
	}
}

// processCortiMessage processes a Corti message and creates a server message
func (s *DictationSession) processCortiMessage(msg map[string]interface{}) DictationServerMessage {
	msgType, _ := msg["type"].(string)

	switch msgType {
	case "transcript":
		// Reference: https://docs.corti.ai/quickstart/dictation#handle-responses
		return DictationServerMessage{
			Type:      DictationMsgTypeTranscript,
			Data:      msg["data"],
			Timestamp: time.Now().UnixMilli(),
		}

	case "command":
		return DictationServerMessage{
			Type:      DictationMsgTypeCommand,
			Data:      msg["data"],
			Timestamp: time.Now().UnixMilli(),
		}

	case "usage":
		credits, _ := msg["credits"].(float64)
		return DictationServerMessage{
			Type: DictationMsgTypeUsage,
			Data: map[string]interface{}{
				"credits": credits,
			},
			Timestamp: time.Now().UnixMilli(),
		}

	case "error":
		return DictationServerMessage{
			Type:      DictationMsgTypeError,
			Data:      msg["error"],
			Timestamp: time.Now().UnixMilli(),
		}

	case "flushed":
		return DictationServerMessage{
			Type: DictationMsgTypeStatus,
			Data: map[string]interface{}{
				"type":   "flushed",
				"status": "flushed",
			},
			Timestamp: time.Now().UnixMilli(),
		}

	case "ended":
		return DictationServerMessage{
			Type: DictationMsgTypeStatus,
			Data: map[string]interface{}{
				"type":   "ended",
				"status": "ended",
			},
			Timestamp: time.Now().UnixMilli(),
		}

	default:
		return DictationServerMessage{
			Type:      DictationMsgTypeStatus,
			Data:      msg,
			Timestamp: time.Now().UnixMilli(),
		}
	}
}

// sendToClient sends a message to the client
func (s *DictationSession) sendToClient(msg DictationServerMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.ClientConn.WriteMessage(fiberws.TextMessage, data)
}

// cleanup closes all connections
func (s *DictationSession) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.IsActive = false

	if s.CortiConn != nil {
		// Send end message if not already sent
		endMsg := map[string]string{"type": "end"}
		if endData, err := json.Marshal(endMsg); err == nil {
			s.CortiConn.WriteMessage(gorillaws.TextMessage, endData)
		}
		time.Sleep(200 * time.Millisecond)
		s.CortiConn.Close()
	}

	close(s.stopChan)
	log.Printf("Dictation session %s cleaned up", s.SessionID)
}

// removeSession removes a session from the proxy
func (p *DictationProxy) removeSession(sessionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.sessions, sessionID)
}

// GetStats returns statistics about active sessions
func (p *DictationProxy) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"active_sessions": len(p.sessions),
	}
}
