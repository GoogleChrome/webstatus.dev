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
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type jwtClaims struct {
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	Issuer    string `json:"iss"`
}

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
	header := jwtHeader{
		Alg: "RS256",
		Typ: "JWT",
	}
	claims := jwtClaims{
		IssuedAt:  now.Add(-60 * time.Second).Unix(),
		ExpiresAt: now.Add(10 * time.Minute).Unix(),
		Issuer:    appID,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal jwt header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal jwt claims: %w", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := headerB64 + "." + claimsB64

	hashed := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign jwt with rsa private key: %w", err)
	}

	sigB64 := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + sigB64, nil
}
