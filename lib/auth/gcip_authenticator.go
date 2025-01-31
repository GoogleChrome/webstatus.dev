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

	firebaseauth "firebase.google.com/go/v4/auth"
)

type GCIPAuthenticator struct {
	FirebaseAuthClient
}

type FirebaseAuthClient interface {
	VerifyIDToken(context.Context, string) (*firebaseauth.Token, error)
}

func NewGCIPAuthenticator(client FirebaseAuthClient) *GCIPAuthenticator {
	return &GCIPAuthenticator{
		client,
	}
}

func (a GCIPAuthenticator) Authenticate(ctx context.Context, idToken string) (*User, error) {
	token, err := a.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, err
	}

	return &User{
		ID: token.UID,
	}, nil
}
