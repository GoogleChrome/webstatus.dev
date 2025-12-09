package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2/google"
)

// Chime API Configuration
const (
	chimeAPIKey = "" // API Key from your GCP project (if required, often not for service account auth)
	// Or use Autopush: "https://autopush-notifications-pa.sandbox.googleapis.com"
	chimeBaseURL     = "https://autopush-notifications-pa-googleapis.sandbox.google.com"
	clientID         = "webstatus_dev"
	notificationType = "SUBSCRIPTION_NOTIFICATION"
)

// Structs for JSON payload (matching Chime's NotifyTargetRequest structure)

type NotifyTargetSyncRequest struct {
	Notification Notification `json:"notification"`
	Target       Target       `json:"target"`
}

type Notification struct {
	ClientID string  `json:"client_id"`
	TypeID   string  `json:"type_id"`
	Payload  Payload `json:"payload"`
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
	CcRecipient  []string   `json:"cc_recipient,omitempty"`
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

// SendChimeEmail sends an email using Chime's NotifyTargetSync API
func SendChimeEmail(ctx context.Context, toAddress, fromAddress, subject, htmlBody, plainBody string, bcc []string) error {
	// 1. Get OAuth Token from Metadata Server
	scopes := []string{"https://www.googleapis.com/auth/notifications"}
	creds, err := google.FindDefaultCredentials(ctx, scopes...)
	if err != nil {
		return fmt.Errorf("failed to find default credentials: %w", err)
	}

	token, err := creds.TokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to retrieve access token: %w", err)
	}

	// 2. Construct the Request Body
	reqBody := NotifyTargetSyncRequest{
		Notification: Notification{
			ClientID: clientID,
			TypeID:   notificationType,
			Payload: Payload{
				TypeURL: "type.googleapis.com/notifications.backend.common.message.RenderedMessage",
				EmailMessage: EmailMessage{
					FromAddress: fromAddress, // Ensure this complies with GMR sender rules
					Subject:     subject,
					BodyPart: []BodyPart{
						{
							Content:     plainBody,
							ContentType: "text/plain",
						},
						{
							Content:     htmlBody,
							ContentType: "text/html",
						},
					},
					CcRecipient:  nil,
					BccRecipient: bcc,
				},
			},
		},
		Target: Target{
			ChannelType: "EMAIL",
			DeliveryAddress: DeliveryAddress{
				EmailAddress: EmailAddress{
					ToAddress: toAddress,
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 3. Make the HTTP POST Request
	apiURL := fmt.Sprintf("%s/v1/notifytargetsync", chimeBaseURL)
	if chimeAPIKey != "" {
		apiURL = fmt.Sprintf("%s?key=%s", apiURL, chimeAPIKey)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to Chime: %w", err)
	}
	defer resp.Body.Close()

	// 4. Handle the Response
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Chime API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	fmt.Println("Chime notification sent successfully to", toAddress)
	// Optionally parse the NotifyTargetSyncResponse JSON
	var responseBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err == nil {
		fmt.Printf("Response: %+v\n", responseBody)
	}
	return nil
}

func main() {
	ctx := context.Background()

	// Example Usage:
	to := ""
	from := "noreply-webstatus-dev@google.com"
	bcc := []string{""}
	subject := "Test Notification from Cloud Run"
	// body := "<h1>Hello from Chime!</h1><p>This is a test email sent via NotifyTargetSync from Cloud Run.</p>"

	if err := SendChimeEmail(ctx, to, from, subject, htmlEmail, textEmail, bcc); err != nil {
		fmt.Printf("Error sending email: %v\n", err)
	} else {
		fmt.Println("Email sending process initiated.")
	}
}
