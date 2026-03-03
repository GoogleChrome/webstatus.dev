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

package httputils

import (
	"errors"
	"net/url"
	"strings"
)

var (
	// ErrInvalidSlackWebhookURL is returned when the Slack webhook URL is invalid.
	ErrInvalidSlackWebhookURL = errors.New(
		"invalid Slack webhook URL. Must be a valid https://hooks.slack.com/services/ URL")
)

// ValidateSlackWebhookURL validates the URL matches the expected Slack webhook URL format as defined by
// https://docs.slack.dev/messaging/sending-messages-using-incoming-webhooks/
// Ex. "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"
func ValidateSlackWebhookURL(webhookURL string) error {
	u, err := url.Parse(webhookURL)
	if err != nil {
		return err
	}

	if u.Scheme != "https" ||
		u.Host != "hooks.slack.com" ||
		!strings.HasPrefix(u.Path, "/services/") {
		return ErrInvalidSlackWebhookURL
	}

	return nil
}
