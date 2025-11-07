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

package auth

import (
	"context"
	"errors"
	"reflect"
	"testing"

	firebaseauth "firebase.google.com/go/v4/auth"
)

func createTestFirebaseToken(uid string, githubID *string) *firebaseauth.Token {
	identities := map[string]interface{}{}
	if githubID != nil {
		identities["github.com"] = []interface{}{*githubID}
	}
	// nolint:exhaustruct
	return &firebaseauth.Token{
		UID: uid,
		Firebase: firebaseauth.FirebaseInfo{
			Identities: identities,
		},
	}
}

// TestGCIPAuthenticator tests the GCIPAuthenticator functions.
func TestGCIPAuthenticator(t *testing.T) {
	tests := []struct {
		name          string
		idToken       string
		mockVerifyFn  func(context.Context, string) (*firebaseauth.Token, error)
		expectedUser  *User
		expectedError bool
	}{
		{
			name:    "Successful authentication",
			idToken: "valid_id_token",
			mockVerifyFn: func(_ context.Context, _ string) (*firebaseauth.Token, error) {
				return createTestFirebaseToken("123", valuePtr("id2")), nil
			},
			expectedUser:  &User{ID: "123", GitHubUserID: valuePtr("id2")},
			expectedError: false,
		},
		{
			name:    "Authentication failure",
			idToken: "invalid_id_token",
			mockVerifyFn: func(_ context.Context, _ string) (*firebaseauth.Token, error) {
				return nil, errors.New("invalid ID token")
			},
			expectedUser:  nil,
			expectedError: true,
		},
		{
			name:    "Missing GitHub user ID",
			idToken: "invalid_id_token",
			mockVerifyFn: func(_ context.Context, _ string) (*firebaseauth.Token, error) {
				return createTestFirebaseToken("123", nil), nil
			},
			expectedUser:  &User{ID: "123", GitHubUserID: nil},
			expectedError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock UserAuthClient.
			mockUserAuthClient := &MockUserAuthClient{
				verifyIDTokenFn: tc.mockVerifyFn,
			}

			// Create a GCIPAuthenticator using the mock client.
			authenticator := NewGCIPAuthenticator(mockUserAuthClient)

			// Authenticate the user.
			user, err := authenticator.Authenticate(context.Background(), tc.idToken)

			// Check if the error matches the expected outcome.
			if tc.expectedError && err == nil {
				t.Fatal("Expected authentication to fail, but it succeeded")
			} else if !tc.expectedError && err != nil {
				t.Fatalf("Failed to authenticate: %v", err)
			}

			// Check if the user matches the expected value.
			if !reflect.DeepEqual(tc.expectedUser, user) {
				t.Errorf("Expected user to be '%+v', got '%+v'", tc.expectedUser, user)
			}
		})
	}
}

func TestExtractGitHubUserIDFromToken(t *testing.T) {
	tests := []struct {
		name          string
		token         *firebaseauth.Token
		expectedID    *string
		expectedFound bool
	}{
		{
			name:          "Success",
			token:         createTestFirebaseToken("test", valuePtr("12345")),
			expectedID:    valuePtr("12345"),
			expectedFound: true,
		},
		{
			name: "Nil Identities",
			// nolint:exhaustruct
			// Keep inline for specific nil case
			token:         &firebaseauth.Token{Firebase: firebaseauth.FirebaseInfo{Identities: nil}},
			expectedID:    nil,
			expectedFound: false,
		},
		{
			name: "Missing github.com identity",
			// nolint:exhaustruct
			token: &firebaseauth.Token{Firebase: firebaseauth.FirebaseInfo{
				Identities: map[string]interface{}{"google.com": []interface{}{"some-id"}},
			}},
			expectedID:    nil,
			expectedFound: false,
		},
		{
			name: "Non-slice identity",
			// nolint:exhaustruct
			token: &firebaseauth.Token{Firebase: firebaseauth.FirebaseInfo{
				Identities: map[string]interface{}{"github.com": "not-a-slice"},
			}},
			expectedID:    nil,
			expectedFound: false,
		},
		{
			name: "Empty identity slice",
			// nolint:exhaustruct
			token: &firebaseauth.Token{Firebase: firebaseauth.FirebaseInfo{
				Identities: map[string]interface{}{"github.com": []interface{}{}},
			}},
			expectedID:    nil,
			expectedFound: false,
		},
		{
			name: "Non-string identity",
			// nolint:exhaustruct
			token: &firebaseauth.Token{Firebase: firebaseauth.FirebaseInfo{
				Identities: map[string]interface{}{"github.com": []interface{}{12345}},
			}},
			expectedID:    nil,
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := extractGitHubUserIDFromToken(tt.token)
			if found != tt.expectedFound {
				t.Errorf("extractGitHubUserIDFromToken() found = %v, want %v", found, tt.expectedFound)
			}
			if !reflect.DeepEqual(got, tt.expectedID) {
				t.Errorf("extractGitHubUserIDFromToken() got = %v, want %v", got, tt.expectedID)
			}
		})
	}
}

// MockUserAuthClient is a mock implementation of the UserAuthClient interface.
type MockUserAuthClient struct {
	verifyIDTokenFn func(context.Context, string) (*firebaseauth.Token, error)
}

// VerifyIDToken verifies an ID token.
func (m *MockUserAuthClient) VerifyIDToken(ctx context.Context, idToken string) (*firebaseauth.Token, error) {
	if m.verifyIDTokenFn == nil {
		panic("verifyIDTokenFn not set")
	}

	return m.verifyIDTokenFn(ctx, idToken)
}
