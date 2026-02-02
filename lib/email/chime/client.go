// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package chime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Env type for environment selection.
type Env int

const (
	// EnvAutopush uses the autopush environment.
	EnvAutopush Env = iota
	// EnvProd uses the production environment.
	EnvProd
)

func getChimeURL(env Env) string {
	switch env {
	case EnvAutopush:
		return "https://autopush-notifications-pa-googleapis.sandbox.google.com"
	case EnvProd:
		return "https://notifications-pa.googleapis.com"
	default:
		return ""
	}
}

// ClientID and other constants.
const (
	clientID         = "webstatus_dev"
	notificationType = "SUBSCRIPTION_NOTIFICATION"
	defaultFromAddr  = "noreply-webstatus-dev@google.com"
)

// Sentinel Errors.
var (
	ErrPermanentUser   = errors.New("permanent error due to user/target issue")
	ErrPermanentSystem = errors.New("permanent error due to system/config issue")
	ErrTransient       = errors.New("transient error, can be retried")
	ErrDuplicate       = errors.New("duplicate notification")
)

// HTTPClient interface to allow mocking http.Client.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Sender struct {
	bcc         []string
	tokenSource oauth2.TokenSource
	httpClient  HTTPClient
	fromAddress string
	baseURL     string
}

// NewChimeSender creates a new ChimeSender instance.
func NewChimeSender(ctx context.Context, env Env, bcc []string, fromAddr string,
	customHTTPClient HTTPClient) (*Sender, error) {
	baseURL := getChimeURL(env)
	if baseURL == "" {
		return nil, fmt.Errorf("%w: invalid ChimeEnv: %v", ErrPermanentSystem, env)
	}

	ts, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/notifications")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to find default credentials: %w", ErrPermanentSystem, err)
	}

	httpClient := customHTTPClient
	if httpClient == nil {
		client := oauth2.NewClient(ctx, ts.TokenSource)
		client.Timeout = 30 * time.Second
		httpClient = client
	}

	if fromAddr == "" {
		fromAddr = defaultFromAddr
	}

	return &Sender{
		bcc:         bcc,
		tokenSource: ts.TokenSource,
		httpClient:  httpClient,
		fromAddress: fromAddr,
		baseURL:     baseURL,
	}, nil
}

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
	ExternalID string `json:"externalId"`
	Identifier string `json:"identifier"`
	Details    struct {
		Outcome string `json:"outcome"`
		Reason  string `json:"reason"`
	} `json:"details"`
}

// --- Send method and its helpers ---

func (s *Sender) Send(ctx context.Context, id string, to string, subject string, htmlBody string) error {
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

	err = s.handleResponse(ctx, resp, bodyBytes, id)
	handleSendResult(ctx, err, id)

	return err
}

func (s *Sender) buildRequestBody(id string, to string, subject string, htmlBody string) ([]byte, error) {
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
		return nil, fmt.Errorf("%w: failed to marshal request body: %w", ErrPermanentSystem, err)
	}

	return jsonData, nil
}

func (s *Sender) createHTTPRequest(ctx context.Context, body []byte) (*http.Request, error) {
	apiURL := fmt.Sprintf("%s/v1/notifytargetsync", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create HTTP request: %w", ErrPermanentSystem, err)
	}

	token, err := s.tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to retrieve access token: %w", ErrPermanentSystem, err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (s *Sender) executeRequest(req *http.Request) (*http.Response, []byte, error) {
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: network error sending to Chime: %w", ErrTransient, err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: failed to read response body: %w", ErrTransient, err)
	}

	return resp, bodyBytes, nil
}

func (s *Sender) handleResponse(ctx context.Context,
	resp *http.Response, bodyBytes []byte, externalID string) error {
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
		slog.WarnContext(ctx, "Chime call OK, but failed to parse response body",
			"externalID", externalID, "error", err, "body", bodyStr)

		return nil
	}

	return classifyChimeOutcome(ctx, externalID, responseBody)
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

func classifyChimeOutcome(ctx context.Context, externalID string, responseBody NotifyTargetSyncResponse) error {
	outcome := responseBody.Details.Outcome
	reason := responseBody.Details.Reason
	chimeID := responseBody.Identifier
	slog.DebugContext(ctx, "Chime Response", "externalID", externalID,
		"chimeID", chimeID, "outcome", outcome, "reason", reason)

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
		}

		return fmt.Errorf("%w: outcome %s, reason: %s", ErrTransient, outcome, reason)
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

func handleSendResult(ctx context.Context, err error, externalID string) {
	if err == nil {
		slog.InfoContext(ctx, "Email sending process initiated and reported as SENT.", "externalID", externalID)

		return
	}
	slog.ErrorContext(ctx, "Error sending email", "externalID", externalID, "error", err)
	if errors.Is(err, ErrDuplicate) {
		slog.ErrorContext(ctx, "Result: This was a DUPLICATE send.", "externalID", externalID)
	} else if errors.Is(err, ErrPermanentUser) {
		slog.ErrorContext(ctx, "Result: PERMANENT error due to USER issue.", "externalID", externalID)
	} else if errors.Is(err, ErrPermanentSystem) {
		slog.ErrorContext(ctx, "Result: PERMANENT error due to SYSTEM issue.", "externalID", externalID)
	} else if errors.Is(err, ErrTransient) {
		slog.ErrorContext(ctx, "Result: TRANSIENT error.", "externalID", externalID)
	} else {
		slog.ErrorContext(ctx, "Result: Unknown error type.", "externalID", externalID)
	}
}
