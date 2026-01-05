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

package smtpsender

import (
	"errors"
	"net/smtp"
	"testing"
)

func TestNewSMTPClient(t *testing.T) {
	t.Parallel()
	var emptyCfg SMTPClientConfig

	testCases := []struct {
		name      string
		config    SMTPClientConfig
		expectErr error
	}{
		{
			name:      "Valid config",
			config:    SMTPClientConfig{Host: "localhost", Port: 1025, Username: "", Password: ""},
			expectErr: nil,
		},
		{
			name:      "Missing host",
			config:    SMTPClientConfig{Host: "", Port: 1025, Username: "", Password: ""},
			expectErr: ErrSMTPConfig,
		},
		{
			name:      "Missing port",
			config:    SMTPClientConfig{Host: "localhost", Port: 0, Username: "", Password: ""},
			expectErr: ErrSMTPConfig,
		},
		{
			name:      "Empty config",
			config:    emptyCfg,
			expectErr: ErrSMTPConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewClient(tc.config, "from@example.com")
			if !errors.Is(err, tc.expectErr) {
				t.Errorf("Expected error wrapping %v, but got %v", tc.expectErr, err)
			}
			if err == nil && client == nil {
				t.Fatal("Expected client, but got nil")
			}
		})
	}
}

func TestSMTPClient_SendMail(t *testing.T) {
	cfg := SMTPClientConfig{Host: "localhost", Port: 1025, Username: "fake", Password: "fake"}

	testCases := []struct {
		name          string
		mockSendMail  func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
		expectedError error
	}{
		{
			name: "Successful send",
			mockSendMail: func(_ string, _ smtp.Auth, _ string, _ []string, _ []byte) error {
				return nil
			},
			expectedError: nil,
		},
		{
			name: "Some error",
			mockSendMail: func(_ string, _ smtp.Auth, _ string, _ []string, _ []byte) error {
				return errors.New("connection refused")
			},
			expectedError: ErrSMTPFailedSend,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewClient(cfg, "from@example.com")
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			client.send = tc.mockSendMail

			err = client.SendMail([]string{"to@example.com"}, []byte("body"))

			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Expected error wrapping %v, but got %v", tc.expectedError, err)
			}
		})
	}
}
