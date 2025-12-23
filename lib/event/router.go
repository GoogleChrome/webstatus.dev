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
	"fmt"
)

// TypedHandler is a function that processes a specific, strongly-typed struct.
type TypedHandler[T Event] func(ctx context.Context, eventID string, event T) error

// Router acts as a multiplexer for incoming Pub/Sub messages.
// It automatically handles parsing the envelope, unmarshalling the payload,
// and dispatching to the correct type-safe handler.
type Router struct {
	routes []route
}

// NewRouter creates a new message router.
func NewRouter() *Router {
	return &Router{
		routes: make([]route, 0),
	}
}

// route hides the generic type T behind a common interface for dispatching.
type route interface {
	Matches(kind, version string) bool
	Dispatch(ctx context.Context, eventID string, data json.RawMessage) error
}

// Register adds a new handler for a specific Kind and APIVersion.
// T: The struct type you want to receive (e.g. *IngestionJobV1).
// The router will automatically json.Unmarshal the message into T before calling handler.
// This function PANICS if a handler for the same Kind and APIVersion is already registered.
func Register[T Event](r *Router, handler TypedHandler[T]) {
	var zero T
	kind := zero.Kind()
	version := zero.APIVersion()

	// Check for conflicts to prevent silent overwrites
	for _, existing := range r.routes {
		if existing.Matches(kind, version) {
			panic(fmt.Sprintf("router: duplicate handler registered for kind=%q version=%q", kind, version))
		}
	}

	r.routes = append(r.routes, &typedRoute[T]{
		kind:    kind,
		version: version,
		handler: handler,
	})
}

// HandleMessage is the single entry point.
func (r *Router) HandleMessage(ctx context.Context, eventID string, data []byte) error {
	// 1. Efficient Peek: We only parse the metadata fields.
	// The rest of the JSON is captured as RawMessage without full decoding.
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("%w: %w: %w", ErrUnprocessableEntity, ErrInvalidEnvelope, err)
	}

	// 2. Find matching route
	for _, route := range r.routes {
		if route.Matches(env.Kind, env.APIVersion) {
			// 3. Dispatch using only the inner 'data' payload

			return route.Dispatch(ctx, eventID, env.Data)
		}
	}

	return fmt.Errorf("%w: %w: kind=%q version=%q", ErrUnprocessableEntity, ErrNoHandler, env.Kind, env.APIVersion)
}

// envelope is used for the initial peek.
type envelope struct {
	APIVersion string          `json:"apiVersion"`
	Kind       string          `json:"kind"`
	Data       json.RawMessage `json:"data"`
}

type typedRoute[T Event] struct {
	kind    string
	version string
	handler TypedHandler[T]
}

func (tr *typedRoute[T]) Matches(kind, version string) bool {
	return tr.kind == kind && tr.version == version
}

func (tr *typedRoute[T]) Dispatch(ctx context.Context, eventID string, data json.RawMessage) error {
	// 1. Final Parse: Convert bytes directly to T
	var payload T
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("%w: %w: parsing %T failed: %w", ErrUnprocessableEntity, ErrSchemaValidation, payload, err)
	}

	// 2. Execute Handler
	return tr.handler(ctx, eventID, payload)
}
