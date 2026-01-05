// Copyright 2025 Google LLC
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

package smtpsenderadapters

import (
	"context"
	"errors"
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

// EmailWorkerSMTPAdapter implements the interface for the email worker
// using the SMTP client.
type EmailWorkerSMTPAdapter struct {
	sender Sender
}

type Sender interface {
	SendMail(to []string, msg []byte) error
	From() string
}

// NewEmailWorkerSMTPAdapter creates a new adapter for the email worker to use SMTP.
func NewEmailWorkerSMTPAdapter(client Sender) *EmailWorkerSMTPAdapter {
	return &EmailWorkerSMTPAdapter{
		sender: client,
	}
}

// Send implements the EmailSender interface for the email worker.
func (a *EmailWorkerSMTPAdapter) Send(ctx context.Context, id string,
	to string,
	subject string,
	htmlBody string) error {

	slog.InfoContext(ctx, "sending email via SMTP", "to", to, "id", id)

	msg := []byte("To: " + to + "\r\n" +
		"From: " + a.sender.From() + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" + htmlBody)

	err := a.sender.SendMail([]string{to}, msg)
	if err != nil {
		return errors.Join(workertypes.ErrUnrecoverableSystemFailureEmailSending, err)

	}

	return nil
}
