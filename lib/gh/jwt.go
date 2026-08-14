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

package gh

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidPrivateKey is returned when the PEM block cannot be parsed as an RSA private key.
	ErrInvalidPrivateKey = errors.New("invalid rsa private key")

	// ErrEmptyAppID is returned when app ID is empty.
	ErrEmptyAppID = errors.New("app id must not be empty")
)

// MintAppJWT generates an RS256-signed JSON Web Token for authenticating as a GitHub App.
// Claims:
// - iat: now - 60s (handles clock skew)
// - exp: now + 10m (GitHub maximum validity)
// - iss: appID.
func MintAppJWT(appID string, privateKeyPEM []byte) (string, error) {
	if appID == "" {
		return "", ErrEmptyAppID
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidPrivateKey, err)
	}

	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		Issuer:    appID,
		Subject:   "",
		Audience:  nil,
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
		NotBefore: nil,
		IssuedAt:  jwt.NewNumericDate(now.Add(-60 * time.Second)),
		ID:        "",
	})

	signedStr, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("failed to sign jwt with rsa private key: %w", err)
	}

	return signedStr, nil
}
