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
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/email/smtpsender"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

// mockSMTPSenderClient is a mock implementation of Sender for testing.
type mockSMTPSenderClient struct {
	sendMailErr error
}

func (m *mockSMTPSenderClient) SendMail(_ []string, _ []byte) error {
	return m.sendMailErr
}

func (m *mockSMTPSenderClient) From() string {
	return "from@example.com"
}

func TestEmailWorkerSmtpAdapter_Send(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name          string
		smtpSendErr   error
		expectedError error
	}{
		{
			name:          "Success",
			smtpSendErr:   nil,
			expectedError: nil,
		},
		{
			name:          "SMTP Failed Send Error",
			smtpSendErr:   smtpsender.ErrSMTPFailedSend,
			expectedError: workertypes.ErrUnrecoverableSystemFailureEmailSending,
		},
		{
			name:          "SMTP Config Error",
			smtpSendErr:   smtpsender.ErrSMTPConfig,
			expectedError: workertypes.ErrUnrecoverableSystemFailureEmailSending,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockSMTPSenderClient{sendMailErr: tc.smtpSendErr}
			adapter := NewEmailWorkerSMTPAdapter(mockClient)

			err := adapter.Send(ctx, "test-id", "to@example.com", "Test Subject", "<p>Hello</p>")
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Expected error wrapping %v, but got %v (raw: %v)", tc.expectedError, err, errors.Unwrap(err))
			}
		})
	}
}
