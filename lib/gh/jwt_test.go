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
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"strings"
	"testing"
	"time"
)

func generateTestRSAPEM(t *testing.T) (*rsa.PrivateKey, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey failed: %v", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(key),
	})

	return key, pemBytes
}

func TestMintAppJWT(t *testing.T) {
	t.Parallel()

	privKey, pemBytes := generateTestRSAPEM(t)
	appID := "webstatus-github-app-123"

	tokenStr, err := MintAppJWT(appID, pemBytes)
	if err != nil {
		t.Fatalf("MintAppJWT failed: %v", err)
	}

	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT parts, got %d", len(parts))
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("failed to decode header: %v", err)
	}
	var header map[string]string
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		t.Fatalf("failed to unmarshal header: %v", err)
	}
	if header["alg"] != "RS256" || header["typ"] != "JWT" {
		t.Errorf("unexpected header: %v", header)
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	var claims struct {
		Iss string `json:"iss"`
		Iat int64  `json:"iat"`
		Exp int64  `json:"exp"`
	}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if claims.Iss != appID {
		t.Errorf("iss = %s, want %s", claims.Iss, appID)
	}
	now := time.Now().Unix()
	if claims.Exp-claims.Iat != 660 {
		t.Errorf("token lifespan = %d, want 660s", claims.Exp-claims.Iat)
	}
	if claims.Iat > now || claims.Exp < now {
		t.Errorf("invalid token timestamps: iat=%d, exp=%d, now=%d", claims.Iat, claims.Exp, now)
	}

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("failed to decode signature: %v", err)
	}
	signedContent := parts[0] + "." + parts[1]
	hasher := sha256.New()
	hasher.Write([]byte(signedContent))
	hashed := hasher.Sum(nil)

	if err := rsa.VerifyPKCS1v15(&privKey.PublicKey, crypto.SHA256, hashed, sigBytes); err != nil {
		t.Errorf("RSA PKCS1v15 signature verification failed: %v", err)
	}
}

func TestMintAppJWTErrors(t *testing.T) {
	t.Parallel()

	_, validPEM := generateTestRSAPEM(t)

	tests := []struct {
		name      string
		appID     string
		pemData   []byte
		targetErr error
	}{
		{
			name:      "empty app ID",
			appID:     "",
			pemData:   validPEM,
			targetErr: ErrEmptyAppID,
		},
		{
			name:      "invalid PEM format",
			appID:     "123",
			pemData:   []byte("not a pem block"),
			targetErr: ErrInvalidPrivateKey,
		},
		{
			name:  "corrupted PEM block bytes",
			appID: "123",
			pemData: pem.EncodeToMemory(&pem.Block{
				Type:    "RSA PRIVATE KEY",
				Headers: nil,
				Bytes:   []byte("garbage"),
			}),
			targetErr: ErrInvalidPrivateKey,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := MintAppJWT(tc.appID, tc.pemData)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, tc.targetErr) {
				t.Errorf("expected error %v, got %v", tc.targetErr, err)
			}
		})
	}
}
