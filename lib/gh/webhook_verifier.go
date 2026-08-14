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
	"bytes"

	"github.com/google/go-github/v79/github"
)

// WebhookVerifier validates incoming GitHub webhook signatures using the official go-github SDK.
type WebhookVerifier struct {
	secret []byte
}

// NewWebhookVerifier creates a new WebhookVerifier with the provided secret.
func NewWebhookVerifier(secret []byte) *WebhookVerifier {
	return &WebhookVerifier{
		secret: secret,
	}
}

// VerifySignature verifies a GitHub HMAC SHA-256 webhook signature header (e.g. "sha256=...").
func (v *WebhookVerifier) VerifySignature(payload []byte, headerSig string) bool {
	if v == nil || len(v.secret) == 0 {
		return false
	}

	_, err := github.ValidatePayloadFromBody("application/json", bytes.NewReader(payload), headerSig, v.secret)

	return err == nil
}
