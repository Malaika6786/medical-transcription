// Package corti provides client implementations for Corti API integration
package corti

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	fiberws "github.com/gofiber/websocket/v2"
	gorillaws "github.com/gorilla/websocket"

	"corti-backend/internal/utils"
)

// AmbientProxy handles WebSocket proxy connections between clients and Corti
// Reference: https://docs.corti.ai/workflows/ambient-rt
type AmbientProxy struct {
	config       *utils.Config
	tokenManager *TokenManager
	asyncClient  *AsyncClient

	// Active sessions tracking
	mu       sync.RWMutex
	sessions map[string]*ProxySession

	// Store Corti WebSocket URLs for each interaction
	wsURLs   map[string]string
	wsURLsMu sync.RWMutex
}

// ProxySession represents an active WebSocket proxy session
type ProxySession struct {
	InteractionID string
	Language      string
	// AudioFormat, when non-empty, is declared to Corti as this session's raw
	// PCM format (mobile clients). Empty means the client sends a
	// self-describing container (WebM/Opus, web clients) that Corti
	// auto-detects.
	AudioFormat string
	ClientConn    *fiberws.Conn   // Fiber/fasthttp WebSocket
	CortiConn     *gorillaws.Conn // Gorilla WebSocket to Corti
	StartedAt     time.Time
	IsActive      bool
	ConfigSent    bool // Track if initial config has been sent to Corti

	mu            sync.Mutex
	stopChan      chan struct{}
	transcripts   []StreamTranscriptEvent
	medicalEvents []StreamMedicalEvent
	facts         []CortiFactMessage
}

// NewAmbientProxy creates a new AmbientProxy instance
func NewAmbientProxy(config *utils.Config, tokenManager *TokenManager, asyncClient *AsyncClient) *AmbientProxy {
	return &AmbientProxy{
		config:       config,
		tokenManager: tokenManager,
		asyncClient:  asyncClient,
		sessions:     make(map[string]*ProxySession),
		wsURLs:       make(map[string]string),
	}
}

// StoreCortiWSURL stores the Corti WebSocket URL for an interaction
func (p *AmbientProxy) StoreCortiWSURL(interactionID, wsURL string) {
	p.wsURLsMu.Lock()
	defer p.wsURLsMu.Unlock()
	p.wsURLs[interactionID] = wsURL
	log.Printf("Stored Corti WebSocket URL for %s: %s", interactionID, wsURL)
}

// GetCortiWSURL retrieves the stored Corti WebSocket URL for an interaction
func (p *AmbientProxy) GetCortiWSURL(interactionID string) (string, bool) {
	p.wsURLsMu.RLock()
	defer p.wsURLsMu.RUnlock()
	url, exists := p.wsURLs[interactionID]
	return url, exists
}

// StartSession creates a new ambient session and returns the interaction details
// Reference: https://docs.corti.ai/workflows/ambient-rt
func (p *AmbientProxy) StartSession(language string, metadata map[string]string) (*CreateInteractionResponse, error) {
	log.Printf("Starting ambient session with language: %s", language)

	// Create interaction - Corti returns websocketUrl for real-time streaming
	req := &CreateInteractionRequest{
		Patient: &PatientInfo{
			Identifier: fmt.Sprintf("patient-%d", time.Now().UnixNano()),
		},
		Encounter: &EncounterInfo{
			Identifier: fmt.Sprintf("encounter-%d", time.Now().UnixNano()),
			Status:     "in-progress",
			Type:       "consultation",
			Period: &EncounterPeriod{
				StartedAt: time.Now().Format(time.RFC3339),
			},
		},
	}

	resp, err := p.asyncClient.CreateInteraction(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create ambient interaction: %w", err)
	}

	log.Printf("Created ambient interaction: %s, Corti WebSocket URL: %s", resp.InteractionID, resp.WebSocketURL)
	return resp, nil
}

// HandleWebSocket handles WebSocket upgrade and proxy logic
// This implements the Corti Ambient AI real-time streaming workflow
// Reference: https://docs.corti.ai/api-reference/stream
//
// According to official Corti documentation:
// "The authentication for the WSS stream requires in addition to the tenant-name
// parameter a token parameter to pass in the Bearer access token."
//
// Flow:
// 1. Browser connects to our backend WebSocket
// 2. Backend connects to Corti WebSocket with token as URL query parameter
// 3. Backend sends stream configuration and waits for CONFIG_ACCEPTED
// 4. Backend proxies audio frames (binary) from browser to Corti
// 5. Backend proxies transcription events from Corti to browser
func (p *AmbientProxy) HandleWebSocket(c *fiberws.Conn, interactionID string, language string, audioFormat string) error {
	log.Printf("=== Starting Corti Ambient WebSocket Proxy ===")
	log.Printf("Interaction ID: %s", interactionID)
	log.Printf("Language: %s", language)
	if audioFormat != "" {
		log.Printf("Audio format: %s", audioFormat)
	}

	// Step 1: Get fresh access token
	// Token TTL is ~300 seconds, ensure it's fresh
	token, err := p.tokenManager.GetToken()
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}
	log.Printf("Got access token (length: %d chars)", len(token))

	// Step 2: Get the Corti WebSocket URL (returned from interaction creation)
	// URL format: wss://api.us.corti.app/audio-bridge/v2/interactions/{id}/streams?tenant-name=base
	cortiWSURL, exists := p.GetCortiWSURL(interactionID)
	if !exists || cortiWSURL == "" {
		return fmt.Errorf("no WebSocket URL found for interaction %s - StartSession must be called first", interactionID)
	}

	// Step 3: Add token as URL query parameter
	// According to https://docs.corti.ai/quickstart/ambient-rt#python-app-example:
	// The Python example shows: ws_url += f"&token={quote(f'Bearer {token}')}"
	// This means the token value must be "Bearer <token>" (with Bearer prefix), URL-encoded
	//
	// Reference: https://docs.corti.ai/quickstart/ambient-rt
	// Format: wss://{stream-url}&token=Bearer {access_token}
	//
	// The token value should be "Bearer <jwt>" URL-encoded
	encodedToken := encodeBearerToken(token)
	if strings.Contains(cortiWSURL, "?") {
		cortiWSURL = cortiWSURL + "&token=" + encodedToken
	} else {
		cortiWSURL = cortiWSURL + "?token=" + encodedToken
	}

	// Log URL without exposing full token
	urlWithoutToken := strings.Split(cortiWSURL, "&token=")[0]
	log.Printf("Corti WebSocket URL: %s&token=[%d chars]", urlWithoutToken, len(token))

	// Step 4: Configure WebSocket dialer
	// No special subprotocols needed - authentication is via URL token parameter
	dialer := newCortiDialer()

	// Step 5: Set HTTP headers (minimal - auth is in URL)
	header := http.Header{}

	log.Printf("Connecting to Corti WebSocket...")
	log.Printf("  URL: %s&token=[REDACTED]", urlWithoutToken)
	log.Printf("  Auth: token parameter in URL (per Corti docs)")

	// Step 5: Connect to Corti WebSocket
	cortiConn, resp, err := dialer.Dial(cortiWSURL, header)
	if err != nil {
		log.Printf("=== WebSocket Connection FAILED ===")
		if resp != nil {
			log.Printf("HTTP Status: %d", resp.StatusCode)

			// Read error response body
			if resp.Body != nil {
				bodyBytes := make([]byte, 4096)
				n, _ := resp.Body.Read(bodyBytes)
				if n > 0 {
					log.Printf("Response Body: %s", string(bodyBytes[:n]))
				}
				resp.Body.Close()
			}

			// Log response headers for debugging
			log.Printf("Response Headers:")
			for key, values := range resp.Header {
				log.Printf("  %s: %v", key, values)
			}

			// Provide specific error guidance
			if resp.StatusCode == 401 {
				log.Printf("ERROR: 401 Unauthorized - Check if:")
				log.Printf("  1. Access token is valid and not expired")
				log.Printf("  2. Token has permission for streaming/ambient features")
				log.Printf("  3. Tenant-Name header matches your Corti tenant")
				log.Printf("  4. The interaction was created with proper encounter data")
			} else if resp.StatusCode == 200 {
				log.Printf("ERROR: Got 200 instead of 101 - WebSocket upgrade failed")
				log.Printf("  This may indicate the endpoint doesn't support WebSocket")
			}
		} else {
			log.Printf("Connection failed with no HTTP response: %v", err)
		}
		return fmt.Errorf("failed to connect to Corti WebSocket: %w", err)
	}

	log.Printf("=== Connected to Corti WebSocket Successfully! ===")
	log.Printf("Negotiated subprotocol: %s", cortiConn.Subprotocol())

	// Create session
	session := &ProxySession{
		InteractionID: interactionID,
		Language:      language,
		AudioFormat:   audioFormat,
		ClientConn:    c,
		CortiConn:     cortiConn,
		StartedAt:     time.Now(),
		IsActive:      true,
		ConfigSent:    false,
		stopChan:      make(chan struct{}),
		transcripts:   make([]StreamTranscriptEvent, 0),
		medicalEvents: make([]StreamMedicalEvent, 0),
		facts:         make([]CortiFactMessage, 0),
	}

	// Register session
	p.mu.Lock()
	p.sessions[interactionID] = session
	p.mu.Unlock()

	defer func() {
		session.cleanup()
		p.mu.Lock()
		delete(p.sessions, interactionID)
		p.mu.Unlock()
		log.Printf("WebSocket proxy session ended for interaction: %s", interactionID)
	}()

	// Send initial config to Corti
	if err := session.sendStreamConfig(); err != nil {
		log.Printf("Failed to send stream config: %v", err)
		return fmt.Errorf("failed to send stream config: %w", err)
	}

	// Send connected message to client
	connectedMsg := ServerWSMessage{
		Type:      WSMessageTypeConnected,
		Timestamp: time.Now().UnixMilli(),
		Data: map[string]interface{}{
			"interaction_id": interactionID,
			"status":         "connected",
			"language":       language,
		},
	}
	if err := session.sendToClient(connectedMsg); err != nil {
		log.Printf("Failed to send connected message: %v", err)
	}

	// Start goroutines for bidirectional communication
	errChan := make(chan error, 2)

	// Client -> Corti (audio frames)
	go func() {
		errChan <- session.proxyClientToCorti()
	}()

	// Corti -> Client (transcripts, events, facts)
	go func() {
		errChan <- session.proxyCortiToClient()
	}()

	// Wait for either direction to complete/error
	err = <-errChan

	// Signal stop to both goroutines
	close(session.stopChan)

	if err != nil {
		log.Printf("WebSocket proxy error for interaction %s: %v", interactionID, err)
	}

	return err
}

// sendStreamConfig sends the initial configuration to Corti WebSocket
// Reference: https://docs.corti.ai/api-reference/stream#stream-configuration
//
// The configuration must be sent immediately after connection and the client
// must wait for CONFIG_ACCEPTED before sending audio data.
func (s *ProxySession) sendStreamConfig() error {
	// Extract language code (e.g., "en-US" -> "en")
	langCode := primaryLanguageCode(s.Language)

	// Build configuration message per Corti documentation
	// Reference: https://docs.corti.ai/api-reference/stream#stream-configuration
	configMsg := StreamConfigMessage{
		Type: "config",
		Configuration: StreamConfiguration{
			Transcription: StreamTranscriptionConfig{
				PrimaryLanguage: langCode,
				IsDiarization:   false, // Don't identify individual speakers
				IsMultichannel:  false, // Mono audio from microphone
				Participants: []StreamParticipant{
					{Channel: 0, Role: "multiple"}, // Single channel, multiple speakers
				},
			},
			Mode: StreamModeConfig{
				Type:         "facts",  // Enable FactsR™ extraction
				OutputLocale: langCode, // Output language for facts
			},
			AudioFormat: s.AudioFormat,
		},
	}

	configData, err := json.Marshal(configMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal stream config: %w", err)
	}

	log.Printf("Sending stream config to Corti: %s", string(configData))

	if err := s.CortiConn.WriteMessage(gorillaws.TextMessage, configData); err != nil {
		return fmt.Errorf("failed to send stream config: %w", err)
	}

	// Wait for CONFIG_ACCEPTED response (with timeout)
	// Per docs: "Clients must send a stream configuration message and wait for
	// a response of type CONFIG_ACCEPTED before transmitting other data."
	s.CortiConn.SetReadDeadline(time.Now().Add(15 * time.Second))
	defer s.CortiConn.SetReadDeadline(time.Time{}) // Reset deadline

	for {
		_, msg, err := s.CortiConn.ReadMessage()
		if err != nil {
			return fmt.Errorf("failed to read config response: %w", err)
		}

		var response CortiStreamMessage
		if err := json.Unmarshal(msg, &response); err != nil {
			log.Printf("Failed to parse config response: %v (raw: %s)", err, string(msg))
			continue
		}

		log.Printf("Received config response: type=%s", response.Type)

		switch response.Type {
		case "CONFIG_ACCEPTED":
			s.ConfigSent = true
			log.Printf("Stream config accepted by Corti")
			return nil
		case "CONFIG_DENIED":
			return fmt.Errorf("stream config denied by Corti: %s", response.Reason)
		case "CONFIG_MISSING", "CONFIG_NOT_PROVIDED":
			return fmt.Errorf("stream config missing: %s", response.Type)
		case "CONFIG_ALREADY_RECEIVED":
			// Config was already sent, treat as success
			s.ConfigSent = true
			log.Printf("Config already received (treated as success)")
			return nil
		default:
			log.Printf("Unexpected message type while waiting for config response: %s", response.Type)
		}
	}
}

// proxyClientToCorti forwards messages from client to Corti
func (s *ProxySession) proxyClientToCorti() error {
	for {
		select {
		case <-s.stopChan:
			return nil
		default:
			// Read message from client (Fiber WebSocket)
			msgType, msg, err := s.ClientConn.ReadMessage()
			if err != nil {
				if fiberws.IsCloseError(err, fiberws.CloseNormalClosure, fiberws.CloseGoingAway) {
					return nil
				}
				return fmt.Errorf("client read error: %w", err)
			}

			// Handle different message types
			if msgType == fiberws.TextMessage {
				// Parse control message
				var clientMsg ClientWSMessage
				if err := json.Unmarshal(msg, &clientMsg); err != nil {
					log.Printf("Failed to parse client message: %v", err)
					continue
				}

				switch clientMsg.Type {
				case WSMessageTypePing:
					// Respond with pong
					s.sendToClient(ServerWSMessage{
						Type:      WSMessageTypePong,
						Timestamp: time.Now().UnixMilli(),
					})
					continue
				case WSMessageTypeStop:
					log.Printf("Client requested stop for interaction: %s", s.InteractionID)
					return nil
				case WSMessageTypeConfig:
					// Forward config to Corti (client may send updated config)
					log.Printf("Forwarding config message to Corti")
					if err := s.CortiConn.WriteMessage(gorillaws.TextMessage, msg); err != nil {
						return fmt.Errorf("corti write error: %w", err)
					}
				default:
					log.Printf("Unknown client message type: %s", clientMsg.Type)
				}
			} else if msgType == fiberws.BinaryMessage {
				// Forward binary audio data to Corti
				if err := s.CortiConn.WriteMessage(gorillaws.BinaryMessage, msg); err != nil {
					return fmt.Errorf("corti write error: %w", err)
				}
			}
		}
	}
}

// proxyCortiToClient forwards messages from Corti to client
func (s *ProxySession) proxyCortiToClient() error {
	for {
		select {
		case <-s.stopChan:
			return nil
		default:
			// Read message from Corti (Gorilla WebSocket)
			msgType, msg, err := s.CortiConn.ReadMessage()
			if err != nil {
				if gorillaws.IsCloseError(err, gorillaws.CloseNormalClosure, gorillaws.CloseGoingAway) {
					return nil
				}
				return fmt.Errorf("corti read error: %w", err)
			}

			if msgType == gorillaws.TextMessage {
				log.Printf("Received message from Corti: %s", string(msg))

				// Parse Corti message
				var cortiMsg CortiStreamMessage
				if err := json.Unmarshal(msg, &cortiMsg); err != nil {
					log.Printf("Failed to parse Corti message: %v", err)
					// Try parsing as generic map
					var genericMsg map[string]interface{}
					if err := json.Unmarshal(msg, &genericMsg); err == nil {
						s.processGenericCortiMessage(genericMsg)
						serverMsg := ServerWSMessage{
							Type:      determineMessageType(genericMsg),
							Data:      genericMsg,
							Timestamp: time.Now().UnixMilli(),
						}
						if err := s.sendToClient(serverMsg); err != nil {
							return fmt.Errorf("client write error: %w", err)
						}
					}
					continue
				}

				// Process and store the message
				s.processCortiStreamMessage(cortiMsg)

				// Forward to client with appropriate type
				serverMsg := s.cortiMessageToServerMessage(cortiMsg)
				if err := s.sendToClient(serverMsg); err != nil {
					return fmt.Errorf("client write error: %w", err)
				}
			}
		}
	}
}

// processCortiStreamMessage processes and stores Corti stream messages
// Reference: https://docs.corti.ai/api-reference/stream#handshake-responses
func (s *ProxySession) processCortiStreamMessage(msg CortiStreamMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch msg.Type {
	case "transcript":
		// Transcript format per docs:
		// { "type": "transcript", "data": [{ "id": "...", "transcript": "...", "final": true/false, ... }] }
		if dataArr, ok := msg.Data.([]interface{}); ok {
			for _, item := range dataArr {
				if dataMap, ok := item.(map[string]interface{}); ok {
					transcript, _ := dataMap["transcript"].(string)
					isFinal, _ := dataMap["final"].(bool)
					speakerId := -1
					if sid, ok := dataMap["speakerId"].(float64); ok {
						speakerId = int(sid)
					}

					event := StreamTranscriptEvent{
						Type:    msg.Type,
						Text:    transcript,
						IsFinal: isFinal,
						Speaker: fmt.Sprintf("speaker_%d", speakerId),
					}
					s.transcripts = append(s.transcripts, event)
					log.Printf("Stored transcript: final=%v, speakerId=%d, text=%s", isFinal, speakerId, truncateText(transcript, 50))
				}
			}
		}

	case "facts":
		// Facts format per docs:
		// { "type": "facts", "fact": [{ "id": "...", "text": "...", "group": "...", "isDiscarded": false, ... }] }
		log.Printf("Processing facts message - typed facts: %d, Data field present: %v", len(msg.Fact), msg.Data != nil)

		factsProcessed := 0
		factsSkipped := 0

		// First try the typed Fact array
		if len(msg.Fact) > 0 {
			for _, factData := range msg.Fact {
				// Skip discarded facts
				if factData.IsDiscarded {
					factsSkipped++
					log.Printf("Skipping discarded fact: id=%s, text=%s", factData.ID, truncateText(factData.Text, 30))
					continue
				}

				// Skip empty facts
				if factData.Text == "" {
					factsSkipped++
					continue
				}

				fact := CortiFactMessage{
					ID:       factData.ID,
					Text:     factData.Text,
					Group:    factData.Group,
					GroupID:  factData.GroupID,
					Source:   factData.Source,
					Category: factData.Group, // Map group to category for backward compat
					Value:    factData.Text,
				}
				s.facts = append(s.facts, fact)
				factsProcessed++

				// Create medical event for backward compatibility
				event := StreamMedicalEvent{
					Type:     "fact",
					Category: factData.Group,
					Value:    factData.Text,
				}
				s.medicalEvents = append(s.medicalEvents, event)
				log.Printf("Stored fact [typed]: group=%s, id=%s, text=%s", factData.Group, factData.ID, truncateText(factData.Text, 50))
			}
		}

		// Always check Data field as fallback (Corti may send facts in either format)
		if msg.Data != nil {
			log.Printf("Checking Data field for additional facts")
			s.processFactsFromDataField(msg.Data, &factsProcessed, &factsSkipped)
		}

		log.Printf("Facts processing complete: %d stored, %d skipped, total facts now: %d",
			factsProcessed, factsSkipped, len(s.facts))

	case "usage":
		log.Printf("Usage report: credits=%.2f", msg.Credits)

	case "ENDED":
		log.Printf("Corti stream ended")

	case "error":
		if msg.Error != nil {
			log.Printf("Corti stream error: [%d] %s - %s", msg.Error.Status, msg.Error.Title, msg.Error.Details)
		}

	case "CONFIG_ACCEPTED", "CONFIG_DENIED", "CONFIG_MISSING", "CONFIG_NOT_PROVIDED", "CONFIG_ALREADY_RECEIVED":
		// Config responses handled in sendStreamConfig
		log.Printf("Config response: %s", msg.Type)
	}
}

// processFactsFromDataField extracts facts from the Data field of a message
// This handles various formats Corti may send facts in
func (s *ProxySession) processFactsFromDataField(data interface{}, processed, skipped *int) {
	// Try as array directly
	if factArr, ok := data.([]interface{}); ok {
		for _, item := range factArr {
			s.processFactItem(item, processed, skipped)
		}
		return
	}

	// Try as map with nested arrays
	if dataMap, ok := data.(map[string]interface{}); ok {
		// Check for "fact" key
		if factArr, ok := dataMap["fact"].([]interface{}); ok {
			for _, item := range factArr {
				s.processFactItem(item, processed, skipped)
			}
		}
		// Check for "facts" key
		if factArr, ok := dataMap["facts"].([]interface{}); ok {
			for _, item := range factArr {
				s.processFactItem(item, processed, skipped)
			}
		}
		// Check for direct fact data
		if text, hasText := dataMap["text"].(string); hasText && text != "" {
			s.processFactItem(dataMap, processed, skipped)
		}
	}
}

// processFactItem processes a single fact item from various formats
func (s *ProxySession) processFactItem(item interface{}, processed, skipped *int) {
	factMap, ok := item.(map[string]interface{})
	if !ok {
		return
	}

	// Check if discarded
	if isDiscarded, ok := factMap["isDiscarded"].(bool); ok && isDiscarded {
		*skipped++
		return
	}

	// Extract fields with fallbacks
	text := ""
	if t, ok := factMap["text"].(string); ok {
		text = t
	} else if v, ok := factMap["value"].(string); ok {
		text = v
	}

	if text == "" {
		*skipped++
		return
	}

	group := ""
	if g, ok := factMap["group"].(string); ok {
		group = g
	} else if c, ok := factMap["category"].(string); ok {
		group = c
	}

	id := ""
	if i, ok := factMap["id"].(string); ok {
		id = i
	}

	groupID := ""
	if gi, ok := factMap["groupId"].(string); ok {
		groupID = gi
	}

	source := ""
	if src, ok := factMap["source"].(string); ok {
		source = src
	}

	// Check for duplicates before adding
	isDuplicate := false
	for _, existing := range s.facts {
		if (id != "" && existing.ID == id) || (existing.Value == text && existing.Category == group) {
			isDuplicate = true
			break
		}
	}

	if isDuplicate {
		log.Printf("Skipping duplicate fact: group=%s, text=%s", group, truncateText(text, 30))
		*skipped++
		return
	}

	fact := CortiFactMessage{
		ID:       id,
		Text:     text,
		Group:    group,
		GroupID:  groupID,
		Source:   source,
		Category: group,
		Value:    text,
	}
	s.facts = append(s.facts, fact)
	*processed++

	// Create medical event for backward compatibility
	event := StreamMedicalEvent{
		Type:     "fact",
		Category: group,
		Value:    text,
	}
	s.medicalEvents = append(s.medicalEvents, event)

	log.Printf("Stored fact [data]: group=%s, id=%s, text=%s", group, id, truncateText(text, 50))
}

// extractFactsFromData extracts facts from the Data field for forwarding to client
func (s *ProxySession) extractFactsFromData(data interface{}) []map[string]interface{} {
	facts := []map[string]interface{}{}

	extractFromItem := func(item interface{}) {
		factMap, ok := item.(map[string]interface{})
		if !ok {
			return
		}

		// Check if discarded
		if isDiscarded, ok := factMap["isDiscarded"].(bool); ok && isDiscarded {
			return
		}

		// Extract text
		text := ""
		if t, ok := factMap["text"].(string); ok {
			text = t
		} else if v, ok := factMap["value"].(string); ok {
			text = v
		}
		if text == "" {
			return
		}

		// Extract group/category
		group := ""
		if g, ok := factMap["group"].(string); ok {
			group = g
		} else if c, ok := factMap["category"].(string); ok {
			group = c
		}

		facts = append(facts, map[string]interface{}{
			"id":          factMap["id"],
			"text":        text,
			"group":       group,
			"groupId":     factMap["groupId"],
			"isDiscarded": false,
			"source":      factMap["source"],
			"createdAt":   factMap["createdAt"],
			"updatedAt":   factMap["updatedAt"],
		})
	}

	// Try as array directly
	if factArr, ok := data.([]interface{}); ok {
		for _, item := range factArr {
			extractFromItem(item)
		}
	}

	// Try as map with nested arrays
	if dataMap, ok := data.(map[string]interface{}); ok {
		if factArr, ok := dataMap["fact"].([]interface{}); ok {
			for _, item := range factArr {
				extractFromItem(item)
			}
		}
		if factArr, ok := dataMap["facts"].([]interface{}); ok {
			for _, item := range factArr {
				extractFromItem(item)
			}
		}
		// Check if the map itself is a fact
		if _, hasText := dataMap["text"].(string); hasText {
			extractFromItem(dataMap)
		}
	}

	return facts
}

// processGenericCortiMessage processes messages as generic map (fallback)
// This handles messages that don't parse cleanly into the typed struct
func (s *ProxySession) processGenericCortiMessage(msg map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msgType, _ := msg["type"].(string)

	switch msgType {
	case "transcript":
		// Handle transcript data array
		if dataArr, ok := msg["data"].([]interface{}); ok {
			for _, item := range dataArr {
				if dataMap, ok := item.(map[string]interface{}); ok {
					event := StreamTranscriptEvent{
						Type: msgType,
					}
					if transcript, ok := dataMap["transcript"].(string); ok {
						event.Text = transcript
					}
					if isFinal, ok := dataMap["final"].(bool); ok {
						event.IsFinal = isFinal
					}
					if speakerId, ok := dataMap["speakerId"].(float64); ok {
						event.Speaker = fmt.Sprintf("speaker_%d", int(speakerId))
					}
					s.transcripts = append(s.transcripts, event)
				}
			}
		}

	case "facts":
		// Handle facts array
		if factArr, ok := msg["fact"].([]interface{}); ok {
			for _, item := range factArr {
				if factMap, ok := item.(map[string]interface{}); ok {
					event := StreamMedicalEvent{Type: "fact"}
					if group, ok := factMap["group"].(string); ok {
						event.Category = group
					}
					if text, ok := factMap["text"].(string); ok {
						event.Value = text
					}
					s.medicalEvents = append(s.medicalEvents, event)

					// Store as fact
					fact := CortiFactMessage{
						Category: event.Category,
						Value:    event.Value,
					}
					if text, ok := factMap["text"].(string); ok {
						fact.Text = text
					}
					if group, ok := factMap["group"].(string); ok {
						fact.Group = group
					}
					s.facts = append(s.facts, fact)
				}
			}
		}

	case "usage":
		if credits, ok := msg["credits"].(float64); ok {
			log.Printf("Usage credits: %.2f", credits)
		}

	case "ENDED":
		log.Printf("Stream ended")

	case "CONFIG_ACCEPTED", "CONFIG_DENIED", "flushed":
		log.Printf("Stream control: %s", msgType)

	case "error":
		if errData, ok := msg["error"].(map[string]interface{}); ok {
			title, _ := errData["title"].(string)
			details, _ := errData["details"].(string)
			log.Printf("Stream error: %s - %s", title, details)
		}

	default:
		// Legacy format handling
		if text, ok := msg["text"].(string); ok {
			event := StreamTranscriptEvent{
				Type: msgType,
				Text: text,
			}
			if isFinal, ok := msg["isFinal"].(bool); ok {
				event.IsFinal = isFinal
			}
			if speaker, ok := msg["speaker"].(string); ok {
				event.Speaker = speaker
			}
			s.transcripts = append(s.transcripts, event)
		}

		// Legacy fact handling
		if category, ok := msg["category"].(string); ok {
			event := StreamMedicalEvent{Type: msgType}
			event.Category = category
			if value, ok := msg["value"].(string); ok {
				event.Value = value
			}
			s.medicalEvents = append(s.medicalEvents, event)

			fact := CortiFactMessage{
				Category: event.Category,
				Value:    event.Value,
			}
			s.facts = append(s.facts, fact)
		}
	}
}

// cortiMessageToServerMessage converts a Corti message to a server message
// Reference: https://docs.corti.ai/api-reference/stream#handshake-responses
func (s *ProxySession) cortiMessageToServerMessage(msg CortiStreamMessage) ServerWSMessage {
	var serverType string
	var data interface{}

	switch msg.Type {
	case "transcript":
		// Transcript data is in msg.Data as array
		serverType = WSMessageTypeTranscript
		transcripts := []map[string]interface{}{}
		if dataArr, ok := msg.Data.([]interface{}); ok {
			for _, item := range dataArr {
				if dataMap, ok := item.(map[string]interface{}); ok {
					transcripts = append(transcripts, dataMap)
				}
			}
		}
		data = map[string]interface{}{
			"type": msg.Type,
			"data": transcripts,
		}

	case "facts":
		// Facts data may be in msg.Fact array or msg.Data
		serverType = WSMessageTypeFact
		facts := []map[string]interface{}{}

		// First try typed Fact array
		for _, fact := range msg.Fact {
			// Skip discarded facts
			if fact.IsDiscarded {
				continue
			}
			if fact.Text != "" {
				facts = append(facts, map[string]interface{}{
					"id":          fact.ID,
					"text":        fact.Text,
					"group":       fact.Group,
					"groupId":     fact.GroupID,
					"isDiscarded": fact.IsDiscarded,
					"source":      fact.Source,
					"createdAt":   fact.CreatedAt,
					"updatedAt":   fact.UpdatedAt,
				})
			}
		}

		// Also check Data field for additional facts
		if msg.Data != nil {
			extractedFacts := s.extractFactsFromData(msg.Data)
			facts = append(facts, extractedFacts...)
		}

		log.Printf("Forwarding %d facts to client", len(facts))
		data = map[string]interface{}{
			"type": msg.Type,
			"fact": facts,
		}

	case "usage":
		serverType = WSMessageTypeStatus
		data = map[string]interface{}{
			"type":    msg.Type,
			"credits": msg.Credits,
		}

	case "ENDED":
		serverType = WSMessageTypeStatus
		data = map[string]interface{}{
			"type":   msg.Type,
			"status": "ended",
		}

	case "error":
		serverType = WSMessageTypeError
		if msg.Error != nil {
			data = map[string]interface{}{
				"id":      msg.Error.ID,
				"title":   msg.Error.Title,
				"status":  msg.Error.Status,
				"details": msg.Error.Details,
				"doc":     msg.Error.Doc,
			}
		} else {
			data = map[string]interface{}{
				"error": "Unknown error",
			}
		}

	case "flushed":
		serverType = WSMessageTypeStatus
		data = map[string]interface{}{
			"type":   msg.Type,
			"status": "flushed",
		}

	default:
		serverType = WSMessageTypeStatus
		data = map[string]interface{}{
			"type": msg.Type,
			"data": msg.Data,
		}
	}

	return ServerWSMessage{
		Type:      serverType,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
}

// sendToClient sends a message to the client WebSocket
func (s *ProxySession) sendToClient(msg ServerWSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.ClientConn.WriteMessage(fiberws.TextMessage, data)
}

// cleanup closes all connections gracefully
// Reference: https://docs.corti.ai/api-reference/stream#ending-the-session
func (s *ProxySession) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.IsActive = false

	if s.CortiConn != nil {
		// Send "end" message to Corti to properly close the session
		// This signals Corti to send remaining transcripts and facts
		endMsg := map[string]string{"type": "end"}
		if endData, err := json.Marshal(endMsg); err == nil {
			log.Printf("Sending 'end' message to Corti")
			s.CortiConn.WriteMessage(gorillaws.TextMessage, endData)

			// Give Corti time to send final messages (usage, ENDED)
			time.Sleep(500 * time.Millisecond)
		}

		s.CortiConn.WriteMessage(gorillaws.CloseMessage,
			gorillaws.FormatCloseMessage(gorillaws.CloseNormalClosure, ""))
		s.CortiConn.Close()
	}

	if s.ClientConn != nil {
		s.ClientConn.WriteMessage(fiberws.CloseMessage,
			fiberws.FormatCloseMessage(fiberws.CloseNormalClosure, ""))
		s.ClientConn.Close()
	}
}

// GetSession retrieves an active session by interaction ID
func (p *AmbientProxy) GetSession(interactionID string) (*ProxySession, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	session, exists := p.sessions[interactionID]
	return session, exists
}

// GetSessionState returns the current state of a session
func (p *AmbientProxy) GetSessionState(interactionID string) (*AmbientSessionState, error) {
	session, exists := p.GetSession(interactionID)
	if !exists {
		return nil, fmt.Errorf("session not found: %s", interactionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	status := "active"
	if !session.IsActive {
		status = "inactive"
	}

	return &AmbientSessionState{
		InteractionID: session.InteractionID,
		Status:        status,
		StartedAt:     session.StartedAt,
		Transcripts:   session.transcripts,
		MedicalEvents: session.medicalEvents,
	}, nil
}

// determineMessageType maps Corti message types to our server message types
func determineMessageType(msg map[string]interface{}) string {
	if msgType, ok := msg["type"].(string); ok {
		switch msgType {
		case "transcript", "partial", "final":
			return WSMessageTypeTranscript
		case "fact":
			return WSMessageTypeFact
		case "medical_event":
			return WSMessageTypeMedicalEvent
		case "error":
			return WSMessageTypeError
		}
	}
	return WSMessageTypeStatus
}

// ActiveSessionCount returns the number of active sessions
func (p *AmbientProxy) ActiveSessionCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.sessions)
}

// truncateText truncates text for logging purposes
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
