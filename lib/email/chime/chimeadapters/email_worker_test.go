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

package chimeadapters

import (
	"context"
	"errors"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/email/chime"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

// mockChimeSender is a mock implementation of the EmailSender for testing.
type mockChimeSender struct {
	sendErr error
}

func (m *mockChimeSender) Send(_ context.Context, _ string, _ string, _ string, _ string) error {
	return m.sendErr
}

var errTest = errors.New("test error")

func TestEmailWorkerChimeAdapter_Send(t *testing.T) {
	ctx := context.Background()
	testCases := []struct {
		name          string
		chimeError    error
		expectedError error
	}{
		{
			name:          "Success",
			chimeError:    nil,
			expectedError: nil,
		},
		{
			name:          "Permanent User Error",
			chimeError:    chime.ErrPermanentUser,
			expectedError: workertypes.ErrUnrecoverableUserFailureEmailSending,
		},
		{
			name:          "Permanent System Error",
			chimeError:    chime.ErrPermanentSystem,
			expectedError: workertypes.ErrUnrecoverableSystemFailureEmailSending,
		},
		{
			name:          "Duplicate Error",
			chimeError:    chime.ErrDuplicate,
			expectedError: workertypes.ErrUnrecoverableSystemFailureEmailSending,
		},
		{
			name:          "Transient Error",
			chimeError:    chime.ErrTransient,
			expectedError: chime.ErrTransient, // Should be passed through
		},
		{
			name:          "Other Error",
			chimeError:    errTest,
			expectedError: errTest, // Should be passed through
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			mockSender := &mockChimeSender{sendErr: tc.chimeError}
			adapter := NewEmailWorkerChimeAdapter(mockSender)

			// Execute
			err := adapter.Send(ctx, "test-id", "to@example.com", "Test Subject", "<p>Hello</p>")

			// Verify
			if tc.expectedError != nil {
				if err == nil {
					t.Fatal("Expected an error, but got nil")
				}
				if !errors.Is(err, tc.expectedError) {
					t.Errorf("Expected error wrapping %v, but got %v", tc.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got %v", err)
				}
			}
		})
	}
}
