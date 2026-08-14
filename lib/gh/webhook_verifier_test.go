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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestWebhookVerifier_VerifySignature(t *testing.T) {
	t.Parallel()

	validSecret := []byte("this-is-a-secure-webhook-secret-1234")
	payload := []byte(`{"action":"push","repository":{"id":12345,"full_name":"org/repo"}}`)

	mac := hmac.New(sha256.New, validSecret)
	mac.Write(payload)
	validSig := fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil)))

	tests := []struct {
		name      string
		payload   []byte
		headerSig string
		secret    []byte
		wantValid bool
	}{
		{
			name:      "valid signature",
			payload:   payload,
			headerSig: validSig,
			secret:    validSecret,
			wantValid: true,
		},
		{
			name:      "tampered payload",
			payload:   []byte(`{"action":"tampered"}`),
			headerSig: validSig,
			secret:    validSecret,
			wantValid: false,
		},
		{
			name:      "wrong secret",
			payload:   payload,
			headerSig: validSig,
			secret:    []byte("different-secret-key-1234567890"),
			wantValid: false,
		},
		{
			name:      "empty secret",
			payload:   payload,
			headerSig: validSig,
			secret:    []byte(""),
			wantValid: false,
		},
		{
			name:      "nil secret",
			payload:   payload,
			headerSig: validSig,
			secret:    nil,
			wantValid: false,
		},
		{
			name:      "missing sha256 prefix",
			payload:   payload,
			headerSig: hex.EncodeToString(mac.Sum(nil)),
			secret:    validSecret,
			wantValid: false,
		},
		{
			name:      "sha1 prefix instead of sha256",
			payload:   payload,
			headerSig: "sha1=abcdef123456",
			secret:    validSecret,
			wantValid: false,
		},
		{
			name:      "invalid hex in signature",
			payload:   payload,
			headerSig: "sha256=zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			secret:    validSecret,
			wantValid: false,
		},
		{
			name:      "truncated hex signature",
			payload:   payload,
			headerSig: "sha256=abcdef",
			secret:    validSecret,
			wantValid: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			verifier := NewWebhookVerifier(tc.secret)
			got := verifier.VerifySignature(tc.payload, tc.headerSig)
			if got != tc.wantValid {
				t.Errorf("VerifySignature() = %v, want %v", got, tc.wantValid)
			}
		})
	}

	t.Run("nil verifier returns false", func(t *testing.T) {
		t.Parallel()
		var nilVerifier *WebhookVerifier
		if nilVerifier.VerifySignature(payload, validSig) {
			t.Errorf("nil verifier should return false")
		}
	})
}

func FuzzWebhookVerifier_VerifySignature(f *testing.F) {
	f.Add([]byte("sample payload"), "sha256=abcdef0123456789", []byte("a-secret-longer-than-16-bytes"))
	f.Add([]byte(""), "invalid", []byte(""))

	f.Fuzz(func(_ *testing.T, payload []byte, header string, secret []byte) {
		verifier := NewWebhookVerifier(secret)
		_ = verifier.VerifySignature(payload, header)
	})
}
