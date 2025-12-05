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

package event

import (
	"encoding/json"
	"fmt"
)

// Event is the interface that all routable messages must implement.
// It allows the router to infer the Kind and APIVersion from the type itself.
type Event interface {
	Kind() string
	APIVersion() string
}

// NewEvent creates a publishable JSON payload for the given event.
// It wraps the event data in the standard envelope with the correct Kind and APIVersion.
func New[T Event](payload T) ([]byte, error) {
	env := envelope{
		Kind:       payload.Kind(),
		APIVersion: payload.APIVersion(),
		Data:       nil,
	}

	// We marshal the payload into the RawMessage field of the envelope
	dataBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("event: failed to marshal payload: %w", err)
	}
	env.Data = dataBytes

	// Finally, marshal the entire envelope
	return json.Marshal(env)
}
