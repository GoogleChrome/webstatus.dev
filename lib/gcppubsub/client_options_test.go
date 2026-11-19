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

package gcppubsub

import (
	"testing"

	"cloud.google.com/go/pubsub/v2"
)

func TestApplyReceiveSettings(t *testing.T) {
	t.Parallel()

	t.Run("no options keeps library defaults", func(t *testing.T) {
		t.Parallel()

		//nolint:exhaustruct // The pubsub client is not needed to apply settings.
		c := &Client{}
		settings := pubsub.DefaultReceiveSettings
		c.applyReceiveSettings(&settings)

		if settings.MaxOutstandingMessages != pubsub.DefaultReceiveSettings.MaxOutstandingMessages {
			t.Errorf("MaxOutstandingMessages = %d, want default %d", settings.MaxOutstandingMessages,
				pubsub.DefaultReceiveSettings.MaxOutstandingMessages)
		}
	})

	t.Run("option overrides the concurrency limit", func(t *testing.T) {
		t.Parallel()

		//nolint:exhaustruct // The pubsub client is not needed to apply settings.
		c := &Client{}
		WithMaxOutstandingMessages(20)(c)

		settings := pubsub.DefaultReceiveSettings
		c.applyReceiveSettings(&settings)

		if settings.MaxOutstandingMessages != 20 {
			t.Errorf("MaxOutstandingMessages = %d, want 20", settings.MaxOutstandingMessages)
		}
	})
}
