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
	"strings"

	"github.com/google/go-github/v79/github"
)

// MinWebhookSecretLength is the minimum secret length required to prevent empty/trivial secret bypass.
const MinWebhookSecretLength = 16

// VerifyWebhookSignature verifies a GitHub HMAC SHA-256 webhook signature header (e.g. "sha256=...")
// using the official go-github SDK (github.ValidatePayloadFromBody).
func VerifyWebhookSignature(payload []byte, headerSig string, secret []byte) bool {
	if len(secret) < MinWebhookSecretLength {
		return false
	}
	cleanSig := strings.TrimSpace(headerSig)
	if cleanSig == "" {
		return false
	}
	if strings.HasPrefix(cleanSig, "sha256=") {
		cleanSig = "sha256=" + strings.TrimSpace(strings.TrimPrefix(cleanSig, "sha256="))
	}

	_, err := github.ValidatePayloadFromBody("application/json", bytes.NewReader(payload), cleanSig, secret)

	return err == nil
}
