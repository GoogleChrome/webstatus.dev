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

package gh

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/go-github/v75/github"
)

func TestUserGitHubClient_ListEmails(t *testing.T) {
	type mockListEmailsConfig struct {
		emails []*github.UserEmail
		err    error
	}
	tests := []struct {
		name           string
		cfg            *mockListEmailsConfig
		expectedEmails []*UserEmail
		expectedError  error
	}{
		{
			name: "Success - Single Page",
			cfg: &mockListEmailsConfig{
				emails: []*github.UserEmail{
					{Email: valuePtr("test1@example.com"), Verified: valuePtr(true)},
					{Email: valuePtr("test2@example.com"), Verified: valuePtr(false)},
				},
				err: nil,
			},
			expectedEmails: []*UserEmail{
				{Email: "test1@example.com", Verified: true},
				{Email: "test2@example.com", Verified: false},
			},
			expectedError: nil,
		},
		{
			name: "API Error",
			cfg: &mockListEmailsConfig{
				emails: nil,
				err:    errTestAPI,
			},
			expectedEmails: nil,
			expectedError:  errTestAPI,
		},
		{
			name: "Nil Email in list",
			cfg: &mockListEmailsConfig{
				emails: []*github.UserEmail{
					{Email: valuePtr("test1@example.com"), Verified: valuePtr(true)},
					nil,
				},
				err: nil,
			},
			expectedEmails: []*UserEmail{
				{Email: "test1@example.com", Verified: true},
			},
			expectedError: nil,
		},
		{
			name: "Nil Email Address",
			cfg: &mockListEmailsConfig{
				emails: []*github.UserEmail{
					{Email: nil, Verified: valuePtr(true)},
				},
				err: nil,
			},
			expectedEmails: nil,
			expectedError:  nil,
		},
		{
			name: "Nil Verified Status",
			cfg: &mockListEmailsConfig{
				emails: []*github.UserEmail{
					{Email: valuePtr("test@example.com"), Verified: nil},
				},
				err: nil,
			},
			expectedEmails: []*UserEmail{
				{Email: "test@example.com", Verified: false},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockUsersClient{
				ListEmailsFunc: func(_ context.Context, _ *github.ListOptions) ([]*github.UserEmail, *github.Response, error) {
					return tt.cfg.emails, createTestGithubResponse(0), tt.cfg.err
				},
				GetFunc: nil,
			}
			appClient := &UserGitHubClient{usersClient: mockClient}

			emails, err := appClient.ListEmails(context.Background())

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("ListEmails() error = %v, wantErr %v", err, tt.expectedError)
			}
			if !reflect.DeepEqual(emails, tt.expectedEmails) {
				t.Errorf("ListEmails() = %v, want %v", emails, tt.expectedEmails)
			}
		})
	}
}
