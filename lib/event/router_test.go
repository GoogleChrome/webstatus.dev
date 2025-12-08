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
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// --- Mock Events for Testing ---

type TestEventV1 struct {
	ID string `json:"id"`
}

// Used for both versions.
const testEventKind = "TestEvent"

func (TestEventV1) Kind() string       { return testEventKind }
func (TestEventV1) APIVersion() string { return "v1" }

type TestEventV2 struct {
	ID       string `json:"id"`
	Priority int    `json:"priority"`
}

func (TestEventV2) Kind() string       { return testEventKind }
func (TestEventV2) APIVersion() string { return "v2" }

func TestRegister(t *testing.T) {
	t.Run("panic on duplicate registration", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("The code did not panic")
			}
		}()

		r := NewRouter()
		handler := func(_ context.Context, _ TestEventV1) error { return nil }

		// Register first time - should succeed
		Register(r, handler)

		// Register second time - should panic
		Register(r, handler)
	})
}

func TestNew(t *testing.T) {
	event := TestEventV1{ID: "test-id"}

	// Generate payload
	data, err := New(event)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Verify JSON structure manually
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("New generated invalid JSON: %v", err)
	}

	if raw["kind"] != testEventKind {
		t.Errorf("expected kind 'TestEvent', got %v", raw["kind"])
	}
	if raw["apiVersion"] != "v1" {
		t.Errorf("expected apiVersion 'v1', got %v", raw["apiVersion"])
	}

	payloadData, ok := raw["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'data' field to be an object")
	}
	if payloadData["id"] != "test-id" {
		t.Errorf("expected payload id 'test-id', got %v", payloadData["id"])
	}
}

func TestHandleMessageSuccess(t *testing.T) {
	r := NewRouter()

	var handledV1, handledV2 bool

	testHandler1 := func(_ context.Context, e TestEventV1) error {
		handledV1 = true
		if e.ID != "v1-id" {
			t.Errorf("v1 handler received wrong ID: %s", e.ID)
		}

		return nil
	}
	testHandler2 := func(_ context.Context, e TestEventV2) error {
		handledV2 = true
		if e.ID != "v2-id" || e.Priority != 5 {
			t.Errorf("v2 handler received wrong data: %+v", e)
		}

		return nil
	}

	// Register handlers
	Register(r, testHandler1)
	Register(r, testHandler2)

	// Create and route V1 event
	v1Payload, err := New(TestEventV1{ID: "v1-id"})
	if err != nil {
		t.Fatalf("failed to create v1 event: %v", err)
	}
	if err := r.HandleMessage(t.Context(), v1Payload); err != nil {
		t.Errorf("failed to handle v1: %v", err)
	}
	if !handledV1 {
		t.Error("v1 handler was not called")
	}

	// Create and route V2 event
	v2Payload, _ := New(TestEventV2{ID: "v2-id", Priority: 5})
	if err := r.HandleMessage(t.Context(), v2Payload); err != nil {
		t.Errorf("failed to handle v2: %v", err)
	}
	if !handledV2 {
		t.Error("v2 handler was not called")
	}
}

func TestHandleMessageErrors(t *testing.T) {
	r := NewRouter()
	Register(r, func(_ context.Context, _ TestEventV1) error { return nil })

	tests := []struct {
		name          string
		input         []byte
		errorContains string
	}{
		{
			name:          "invalid json",
			input:         []byte(`{ "kind": "TestEvent", "apiVe`),
			errorContains: "invalid envelope",
		},
		{
			name:          "unknown handler",
			input:         mustNew(TestEventV2{ID: "id", Priority: 0}),
			errorContains: "no handler registered",
		},
		{
			name: "schema mismatch (payload data invalid)",
			// Correct envelope, but 'data' is an array instead of expected object
			input:         []byte(`{"kind": "TestEvent", "apiVersion": "v1", "data": []}`),
			errorContains: "parsing event.TestEventV1 failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := r.HandleMessage(t.Context(), tc.input)

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			// All errors from router should be Permanent errors (ErrUnprocessableEntity)
			if !errors.Is(err, ErrUnprocessableEntity) {
				t.Errorf("expected error to wrap ErrUnprocessableEntity, got: %v", err)
			}

			if tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
				t.Errorf("expected error containing %q, got %q", tc.errorContains, err.Error())
			}
		})
	}
}

// Helper to ignore errors in test table setup.
func mustNew[T Event](e T) []byte {
	b, err := New(e)
	if err != nil {
		panic(err)
	}

	return b
}
