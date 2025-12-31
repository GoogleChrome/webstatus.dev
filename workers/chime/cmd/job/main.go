package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// ChimeEnv type for environment selection
type ChimeEnv int

const (
	// EnvAutopush uses the autopush environment
	EnvAutopush ChimeEnv = iota
	// EnvStaging uses the staging environment
	EnvStaging
	// EnvProd uses the production environment
	EnvProd
)

var chimeBaseURLs = map[ChimeEnv]string{
	EnvAutopush: "https://autopush-notifications-pa-googleapis.sandbox.google.com",
	EnvProd:     "https://notifications-pa.googleapis.com",
}

// ClientID and other constants
const (
	clientID         = "webstatus_dev"
	notificationType = "SUBSCRIPTION_NOTIFICATION"
	defaultFromAddr  = "noreply-webstatus-dev@google.com"
)

// Sentinel Errors
var (
	ErrPermanentUser   = errors.New("permanent error due to user/target issue")
	ErrPermanentSystem = errors.New("permanent error due to system/config issue")
	ErrTransient       = errors.New("transient error, can be retried")
	ErrDuplicate       = errors.New("duplicate notification")
)

// EmailSender Interface
type EmailSender interface {
	Send(ctx context.Context, id string, to string, subject string, htmlBody string) error
}

// HTTPClient interface to allow mocking http.Client
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ChimeSender Struct
type ChimeSender struct {
	bcc         []string
	tokenSource oauth2.TokenSource
	httpClient  HTTPClient // Use the interface
	fromAddress string
	baseURL     string
}

// NewChimeSender creates a new ChimeSender instance
func NewChimeSender(ctx context.Context, env ChimeEnv, bcc []string, fromAddr string, customHTTPClient HTTPClient) (*ChimeSender, error) {
	baseURL, ok := chimeBaseURLs[env]
	if !ok {
		return nil, fmt.Errorf("%w: invalid ChimeEnv: %v", ErrPermanentSystem, env)
	}

	ts, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/notifications")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to find default credentials: %v", ErrPermanentSystem, err)
	}

	var httpClient HTTPClient = customHTTPClient
	if httpClient == nil {
		client := oauth2.NewClient(ctx, ts.TokenSource)
		client.Timeout = 30 * time.Second
		httpClient = client
	}

	if fromAddr == "" {
		fromAddr = defaultFromAddr
	}

	return &ChimeSender{
		bcc:         bcc,
		tokenSource: ts.TokenSource,
		httpClient:  httpClient,
		fromAddress: fromAddr,
		baseURL:     baseURL,
	}, nil
}

// --- Structs for JSON payload ---
type NotifyTargetSyncRequest struct {
	Notification Notification `json:"notification"`
	Target       Target       `json:"target"`
}
type Notification struct {
	ClientID   string  `json:"client_id"`
	ExternalID string  `json:"external_id"`
	TypeID     string  `json:"type_id"`
	Payload    Payload `json:"payload"`
}
type Source struct {
	SystemName string `json:"system_name"`
}
type Payload struct {
	TypeURL      string       `json:"@type"`
	EmailMessage EmailMessage `json:"email_message"`
}
type EmailMessage struct {
	FromAddress  string     `json:"from_address"`
	Subject      string     `json:"subject"`
	BodyPart     []BodyPart `json:"body_part"`
	BccRecipient []string   `json:"bcc_recipient,omitempty"`
}
type BodyPart struct {
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}
type Target struct {
	ChannelType     string          `json:"channel_type"`
	DeliveryAddress DeliveryAddress `json:"delivery_address"`
}
type DeliveryAddress struct {
	EmailAddress EmailAddress `json:"email_address"`
}
type EmailAddress struct {
	ToAddress string `json:"to_address"`
}
type NotifyTargetSyncResponse struct {
	ExternalId string `json:"externalId"`
	Identifier string `json:"identifier"`
	Details    struct {
		Outcome string `json:"outcome"`
		Reason  string `json:"reason"`
	} `json:"details"`
}

// --- Send method and its helpers ---

// Send implements the EmailSender interface for ChimeSender
func (s *ChimeSender) Send(ctx context.Context, id string, to string, subject string, htmlBody string) error {
	if id == "" {
		return fmt.Errorf("%w: id (externalID) cannot be empty", ErrPermanentSystem)
	}

	reqBodyData, err := s.buildRequestBody(id, to, subject, htmlBody)
	if err != nil {
		return err
	}

	httpReq, err := s.createHTTPRequest(ctx, reqBodyData)
	if err != nil {
		return err
	}

	resp, bodyBytes, err := s.executeRequest(httpReq)
	if err != nil {
		return err // errors from executeRequest are already wrapped
	}
	defer resp.Body.Close()

	return s.handleResponse(resp, bodyBytes, id)
}

func (s *ChimeSender) buildRequestBody(id string, to string, subject string, htmlBody string) ([]byte, error) {
	reqBody := NotifyTargetSyncRequest{
		Notification: Notification{
			ClientID:   clientID,
			ExternalID: id,
			TypeID:     notificationType,
			Payload: Payload{
				TypeURL: "type.googleapis.com/notifications.backend.common.message.RenderedMessage",
				EmailMessage: EmailMessage{
					FromAddress: s.fromAddress,
					Subject:     subject,
					BodyPart: []BodyPart{
						{Content: htmlBody, ContentType: "text/html"},
					},
					BccRecipient: s.bcc,
				},
			},
		},
		Target: Target{
			ChannelType: "EMAIL",
			DeliveryAddress: DeliveryAddress{
				EmailAddress: EmailAddress{ToAddress: to},
			},
		},
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal request body: %v", ErrPermanentSystem, err)
	}
	return jsonData, nil
}

func (s *ChimeSender) createHTTPRequest(ctx context.Context, body []byte) (*http.Request, error) {
	apiURL := fmt.Sprintf("%s/v1/notifytargetsync", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create HTTP request: %v", ErrPermanentSystem, err)
	}

	token, err := s.tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to retrieve access token: %v", ErrPermanentSystem, err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (s *ChimeSender) executeRequest(req *http.Request) (*http.Response, []byte, error) {
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: network error sending to Chime: %v", ErrTransient, err)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close() // Close body if ReadAll fails
		return nil, nil, fmt.Errorf("%w: failed to read response body: %v", ErrTransient, err)
	}
	return resp, bodyBytes, nil
}

func (s *ChimeSender) handleResponse(resp *http.Response, bodyBytes []byte, externalID string) error {
	bodyStr := string(bodyBytes)

	if resp.StatusCode == http.StatusConflict { // 409
		return fmt.Errorf("%w: external_id %s: %s", ErrDuplicate, externalID, bodyStr)
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return classifyHTTPClientError(resp.StatusCode, bodyStr)
	} else if resp.StatusCode >= 500 {
		return fmt.Errorf("%w: Chime server error (%d): %s", ErrTransient, resp.StatusCode, bodyStr)
	}

	var responseBody NotifyTargetSyncResponse
	if err := json.Unmarshal(bodyBytes, &responseBody); err != nil {
		// Chime accepted it, but response is not what we expected. Log and treat as success.
		fmt.Printf("Chime call OK (ExternalID: %s), but failed to parse response body: %v. Body: %s\n", externalID, err, bodyStr)
		return nil
	}

	return classifyChimeOutcome(externalID, responseBody)
}

func classifyHTTPClientError(statusCode int, bodyStr string) error {
	switch statusCode {
	case http.StatusBadRequest: // 400
		return fmt.Errorf("%w: bad request (400): %s", ErrPermanentSystem, bodyStr)
	case http.StatusUnauthorized: // 401
		return fmt.Errorf("%w: unauthorized (401): %s", ErrPermanentSystem, bodyStr)
	case http.StatusForbidden: // 403
		return fmt.Errorf("%w: forbidden (403): %s", ErrPermanentSystem, bodyStr)
	default:
		return fmt.Errorf("%w: client error (%d): %s", ErrPermanentSystem, statusCode, bodyStr)
	}
}

func classifyChimeOutcome(externalID string, responseBody NotifyTargetSyncResponse) error {
	outcome := responseBody.Details.Outcome
	reason := responseBody.Details.Reason
	chimeID := responseBody.Identifier
	fmt.Printf("Chime Response: ExternalID: %s, ChimeID: %s, Outcome: %s, Reason: %s\n", externalID, chimeID, outcome, reason)

	switch outcome {
	case "SENT":
		return nil // Success
	case "PREFERENCE_DROPPED", "INVALID_AUTH_SUB_TOKEN_DROPPED":
		return fmt.Errorf("%w: outcome %s, reason: %s", ErrPermanentUser, outcome, reason)
	case "EXPLICITLY_DROPPED", "MESSAGE_TOO_LARGE_DROPPED", "INVALID_REQUEST_DROPPED":
		return fmt.Errorf("%w: outcome %s, reason: %s", ErrPermanentSystem, outcome, reason)
	case "DELIVERY_FAILURE_DROPPED":
		if isUserCausedDeliveryFailure(reason) {
			return fmt.Errorf("%w: outcome %s, reason: %s", ErrPermanentUser, outcome, reason)
		} else if isSystemCausedDeliveryFailure(reason) {
			return fmt.Errorf("%w: outcome %s, reason: %s", ErrPermanentSystem, outcome, reason)
		} else {
			return fmt.Errorf("%w: outcome %s, reason: %s", ErrTransient, outcome, reason)
		}
	case "QUOTA_DROPPED":
		return fmt.Errorf("%w: outcome %s, reason: %s", ErrTransient, outcome, reason)
	default: // Unknown outcome
		return fmt.Errorf("%w: unknown outcome %s, reason: %s", ErrTransient, outcome, reason)
	}
}

func isUserCausedDeliveryFailure(reason string) bool {
	userKeywords := []string{"invalid_mailbox", "no such user", "invalid_domain", "domain not found", "unroutable address"}
	lowerReason := strings.ToLower(reason)
	for _, kw := range userKeywords {
		if strings.Contains(lowerReason, kw) {
			return true
		}
	}
	return strings.Contains(lowerReason, "perm_fail") && !isSystemCausedDeliveryFailure(reason)
}

func isSystemCausedDeliveryFailure(reason string) bool {
	systemKeywords := []string{"perm_fail_sender_denied", "mail loop"}
	lowerReason := strings.ToLower(reason)
	for _, kw := range systemKeywords {
		if strings.Contains(lowerReason, kw) {
			return true
		}
	}
	return false
}

func handleSendResult(err error) {
	if err != nil {
		fmt.Printf("\nError sending email: %v\n", err)
		if errors.Is(err, ErrDuplicate) {
			fmt.Println("Result: This was a DUPLICATE send.")
		} else if errors.Is(err, ErrPermanentUser) {
			fmt.Println("Result: PERMANENT error due to USER issue.")
		} else if errors.Is(err, ErrPermanentSystem) {
			fmt.Println("Result: PERMANENT error due to SYSTEM issue.")
		} else if errors.Is(err, ErrTransient) {
			fmt.Println("Result: TRANSIENT error.")
		} else {
			fmt.Println("Result: Unknown error type.")
		}
	} else {
		fmt.Println("\nEmail sending process initiated and reported as SENT.")
	}
}

// --- Main function for demonstration ---
func main() {
	ctx := context.Background()

	// Initialize ChimeSender
	bccList := []string{}                                             // Add BCC addresses if needed
	sender, err := NewChimeSender(ctx, EnvAutopush, bccList, "", nil) // Use default from addr, no custom HTTP client
	if err != nil {
		fmt.Printf("Failed to create ChimeSender: %v\n", err)
		return
	}

	// Example Send
	myExternalID := uuid.New().String()
	to := ""
	subject := "Test from Refactored ChimeSender"
	htmlEmail := "<h1>Hello from Refactored ChimeSender!</h1><p>This email was sent using the refactored ChimeSender struct.</p>"

	fmt.Println("--- First Send Attempt ---")
	err = sender.Send(ctx, myExternalID, to, subject, htmlEmail)
	handleSendResult(err)

	// Example of a duplicate send attempt
	fmt.Println("\n--- Second Send Attempt (Duplicate) ---")
	// Using the SAME myExternalID, to, subject, htmlBody
	err = sender.Send(ctx, myExternalID, to, subject, htmlEmail)
	handleSendResult(err)
}
