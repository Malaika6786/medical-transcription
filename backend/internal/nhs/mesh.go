package nhs

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// MESH (Message Exchange for Social Care and Health) is the NHS transport
// GP Connect: Send Document rides on — a mailbox-to-mailbox file transfer
// service, not a synchronous request/response API. This client implements
// the publicly documented MESH REST API v2 request shape (mailbox-to-
// mailbox send, custom "NHSMESH" authorization scheme). It CANNOT reach a
// real NHS mailbox without: (1) a MESH mailbox ID + password issued to a
// registered organisation, and (2) the organisation's registered shared
// key. Treat every header name/format below as needing verification
// against NHS Digital's current MESH API specification
// (https://digital.nhs.uk/services/message-exchange-for-social-care-and-health-mesh)
// before sending anything to the real MESH integration or live
// environment — this is flagged explicitly, not guessed silently.
//
// WorkflowGPFedConsultReport is the specific MESH workflow ID GP Connect:
// Send Document uses for a consultation report (per NHS Digital's
// published MESH workflow ID list) — the receiving practice's system
// (SystmOne, EMIS, Vision) routes an inbound message to the right handling
// purely off this ID.
const WorkflowGPFedConsultReport = "GPFED_CONSULT_REPORT"

// MESHConfig holds the mailbox credentials for one sending organisation.
// All of these are issued by NHS Digital/TPP during onboarding — see
// docs/regulatory/PREREQUISITES.md — and have no meaningful default.
type MESHConfig struct {
	BaseURL         string // e.g. the MESH integration-test or live API root
	MailboxID       string
	MailboxPassword string
	SharedKey       string // the "shared secret" issued alongside the mailbox
}

// MESHClient sends messages over MESH. It does not implement inbox
// polling/download — this project only ever needs to *send* a document
// (GP Connect: Send Document is one-way), never receive.
type MESHClient struct {
	cfg        MESHConfig
	HTTPClient *http.Client
}

func NewMESHClient(cfg MESHConfig) *MESHClient {
	return &MESHClient{cfg: cfg, HTTPClient: &http.Client{Timeout: 30 * time.Second}}
}

// SendMessage POSTs one file to recipientMailboxID under workflowID. Returns
// the MESH-assigned message ID on success. filename is informational
// (surfaces in the receiving practice's inbox); NHS Digital's ITK3
// convention for Send Document is a ".json" or ".xml" FHIR message body —
// this project sends the FHIR Bundle JSON built by BuildSendDocumentBundle.
func (c *MESHClient) SendMessage(ctx context.Context, recipientMailboxID, workflowID, filename string, body []byte) (messageID string, err error) {
	if c.cfg.MailboxID == "" || c.cfg.MailboxPassword == "" || c.cfg.SharedKey == "" {
		return "", fmt.Errorf("nhs: MESH is not configured (MESH_MAILBOX_ID/MESH_MAILBOX_PASSWORD/MESH_SHARED_KEY) — cannot send to a real mailbox without NHS-issued credentials, see docs/regulatory/PREREQUISITES.md")
	}

	endpoint := fmt.Sprintf("%s/messageexchange/%s/outbox/%s/%s", c.cfg.BaseURL, c.cfg.MailboxID, recipientMailboxID, workflowID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Mex-From", c.cfg.MailboxID)
	req.Header.Set("Mex-To", recipientMailboxID)
	req.Header.Set("Mex-WorkflowID", workflowID)
	req.Header.Set("Mex-FileName", filename)
	req.Header.Set("Mex-ClientVersion", "xstek-nhs-integration/0.1")
	req.Header.Set("Authorization", c.authorizationHeader())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("nhs: MESH send failed (is MESH_BASE_URL reachable?): %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("nhs: MESH rejected the message (%d): %s", resp.StatusCode, truncate(string(respBody), 500))
	}
	return resp.Header.Get("Mex-MessageID"), nil
}

// authorizationHeader implements MESH's documented "NHSMESH" scheme:
//
//	Authorization: NHSMESH <mailboxID>:<nonce>:<nonceCount>:<hmac>
//
// where hmac = HMAC-SHA256(sharedKey, mailboxID + ':' + nonce + ':' +
// mailboxPassword + ':' + nonceCount + ':' + timestamp), hex-encoded. This
// mirrors the publicly described MESH v2 auth scheme; confirm the exact
// field order/format against NHS Digital's current MESH API spec before
// relying on it against a real environment — auth-header formats are
// exactly the kind of detail that drifts between API versions.
func (c *MESHClient) authorizationHeader() string {
	nonce := randomNonce()
	nonceCount := "0"
	timestamp := time.Now().UTC().Format("200601021504")
	message := fmt.Sprintf("%s:%s:%s:%s:%s", c.cfg.MailboxID, nonce, c.cfg.MailboxPassword, nonceCount, timestamp)
	mac := hmac.New(sha256.New, []byte(c.cfg.SharedKey))
	mac.Write([]byte(message))
	signature := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("NHSMESH %s:%s:%s:%s:%s", c.cfg.MailboxID, nonce, nonceCount, timestamp, signature)
}

func randomNonce() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36) + strconv.Itoa(rand.Intn(1_000_000))
}
