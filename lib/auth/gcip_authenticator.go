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
	"log/slog"

	firebaseauth "firebase.google.com/go/v4/auth"
)

// GCIPAuthenticator is a struct that authenticates users using Firebase Auth,
// which is part of Google Cloud Identity Platform (GCIP).
// It implements the UserAuthClient interface.
type GCIPAuthenticator struct {
	UserAuthClient
}

// UserAuthClient is an interface that defines the methods for interacting with a user authentication provider.
type UserAuthClient interface {
	VerifyIDToken(context.Context, string) (*firebaseauth.Token, error)
}

// NewGCIPAuthenticator creates a new GCIPAuthenticator instance.
func NewGCIPAuthenticator(client UserAuthClient) *GCIPAuthenticator {
	return &GCIPAuthenticator{
		client,
	}
}

func extractGitHubUserIDFromToken(token *firebaseauth.Token) (*string, bool) {
	if token.Firebase.Identities == nil {
		return nil, false
	}

	rawGithubIdentities, ok := token.Firebase.Identities["github.com"]
	if !ok {
		return nil, false
	}

	// Have to convert to []interface{} first.
	// Cannot directly convert to []string.
	githubIDs, ok := rawGithubIdentities.([]interface{})
	if !ok || len(githubIDs) == 0 {
		return nil, false
	}

	if githubID, ok := githubIDs[0].(string); ok {
		return &githubID, true
	}

	return nil, false
}

// Authenticate authenticates a user using the provided ID token.
func (a GCIPAuthenticator) Authenticate(ctx context.Context, idToken string) (*User, error) {
	// Verify the ID token using the Firebase Auth client.
	token, err := a.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, err
	}

	gitHubUserID, found := extractGitHubUserIDFromToken(token)
	if !found {
		slog.WarnContext(ctx, "github user ID not found in token", "identities", token.Firebase.Identities,
			"uid", token.UID)
	}

	// Create a new user object using the user's ID from the token.
	return &User{
		ID:           token.UID,
		GitHubUserID: gitHubUserID,
	}, nil
}
