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
	"errors"
	"fmt"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/webhookverifiertypes"
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
		wantErr   error
	}{
		{
			name:      "valid signature",
			payload:   payload,
			headerSig: validSig,
			secret:    validSecret,
			wantErr:   nil,
		},
		{
			name:      "tampered payload",
			payload:   []byte(`{"action":"tampered"}`),
			headerSig: validSig,
			secret:    validSecret,
			wantErr:   webhookverifiertypes.ErrInvalidSignature,
		},
		{
			name:      "wrong secret",
			payload:   payload,
			headerSig: validSig,
			secret:    []byte("different-secret-key-1234567890"),
			wantErr:   webhookverifiertypes.ErrInvalidSignature,
		},
		{
			name:      "empty secret",
			payload:   payload,
			headerSig: validSig,
			secret:    []byte(""),
			wantErr:   webhookverifiertypes.ErrSecretNotConfigured,
		},
		{
			name:      "nil secret",
			payload:   payload,
			headerSig: validSig,
			secret:    nil,
			wantErr:   webhookverifiertypes.ErrSecretNotConfigured,
		},
		{
			name:      "empty signature header",
			payload:   payload,
			headerSig: "",
			secret:    validSecret,
			wantErr:   webhookverifiertypes.ErrMissingSignature,
		},
		{
			name:      "missing sha256 prefix",
			payload:   payload,
			headerSig: hex.EncodeToString(mac.Sum(nil)),
			secret:    validSecret,
			wantErr:   webhookverifiertypes.ErrInvalidSignature,
		},
		{
			name:      "sha1 prefix instead of sha256",
			payload:   payload,
			headerSig: "sha1=abcdef123456",
			secret:    validSecret,
			wantErr:   webhookverifiertypes.ErrInvalidSignature,
		},
		{
			name:      "invalid hex in signature",
			payload:   payload,
			headerSig: "sha256=zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			secret:    validSecret,
			wantErr:   webhookverifiertypes.ErrInvalidSignature,
		},
		{
			name:      "truncated hex signature",
			payload:   payload,
			headerSig: "sha256=abcdef",
			secret:    validSecret,
			wantErr:   webhookverifiertypes.ErrInvalidSignature,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			verifier := NewWebhookVerifier(tc.secret)
			err := verifier.VerifySignature(tc.payload, tc.headerSig)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("VerifySignature() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}

	t.Run("nil verifier returns ErrSecretNotConfigured", func(t *testing.T) {
		t.Parallel()
		var nilVerifier *WebhookVerifier
		err := nilVerifier.VerifySignature(payload, validSig)
		if !errors.Is(err, webhookverifiertypes.ErrSecretNotConfigured) {
			t.Errorf("nil verifier should return ErrSecretNotConfigured, got %v", err)
		}
	})
}

func FuzzWebhookVerifier_VerifySignature(f *testing.F) {
	f.Add([]byte(`{"action":"push","repository":{"id":12345}}`), []byte("a-secret-longer-than-16-bytes"))
	f.Add([]byte(""), []byte("short-secret"))
	f.Add([]byte("\x00\xff\x00\xff"), []byte("special-bytes"))
	f.Add([]byte("sample payload"), []byte(""))

	f.Fuzz(func(t *testing.T, payload []byte, secret []byte) {
		verifier := NewWebhookVerifier(secret)

		// Invariant 1: Empty Secret Boundary
		if len(secret) == 0 {
			err := verifier.VerifySignature(payload, "sha256=dummy")
			if !errors.Is(err, webhookverifiertypes.ErrSecretNotConfigured) {
				t.Fatalf("expected ErrSecretNotConfigured for empty secret, got %v", err)
			}

			return
		}

		// Generate a valid signature for this random payload & secret
		mac := hmac.New(sha256.New, secret)
		mac.Write(payload)
		validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

		// Invariant 2: Legitimate Signatures Must Always Pass
		if err := verifier.VerifySignature(payload, validSig); err != nil {
			t.Fatalf("valid signature failed verification: %v\npayload: %q\nsecret: %q", err, payload, secret)
		}

		// Invariant 3: Tampered Payloads Must Always Fail
		if len(payload) > 0 {
			tamperedPayload := make([]byte, len(payload))
			copy(tamperedPayload, payload)
			tamperedPayload[0] ^= 0xFF // Flip bits in the first byte

			if err := verifier.VerifySignature(tamperedPayload, validSig); err == nil {
				t.Fatalf("tampered payload unexpectedly passed verification!\npayload: %q", payload)
			}
		}

		// Invariant 4: Corrupted Signatures Must Always Fail
		corruptedHex := []byte(validSig[7:])
		if corruptedHex[0] == '0' {
			corruptedHex[0] = '1'
		} else {
			corruptedHex[0] = '0'
		}
		corruptedSig := "sha256=" + string(corruptedHex)
		if err := verifier.VerifySignature(payload, corruptedSig); err == nil {
			t.Fatalf("corrupted signature unexpectedly passed verification!")
		}
	})
}
