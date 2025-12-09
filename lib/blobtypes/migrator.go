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

package blobtypes

import (
	"encoding/json"
	"fmt"
)

// Payload represents a complete, versioned data structure intended for storage as a blob.
//
// The Kind() and Version() methods are crucial for the migration system.
// - Kind() returns a stable, machine-readable name for the data type (e.g., "SavedSearchSnapshot").
// - Version() returns the specific schema version of the struct (e.g., "v1", "v2").
//
// Together, they allow the Migrator to identify the exact schema of a raw blob
// and apply the correct upgrade logic.
//
// Example Implementation:
//
//	const myKind = "MyDataType"
//
//	type MyDataV1 struct {
//		Name string `json:"name"`
//	}
//
//	func (d MyDataV1) Kind() string    { return myKind }
//	func (d MyDataV1) Version() string { return "v1" }
type Payload interface {
	Kind() string
	Version() string
}

// Migratable is a generic interface that enforces type-safe upgrades.
// A type T implementing Migratable guarantees it can convert itself (Metadata + Data)
// to the Next version.
type Migratable[Next Payload] interface {
	Payload
	Upgrade() (Next, error)
}

// NewBlob creates the final JSON bytes for storage.
// It takes the high-level wrapper (Payload) and injects the standard envelope headers.
func NewBlob(p Payload) ([]byte, error) {
	// 1. Marshal the Payload itself (e.g. { "metadata":..., "data":... })
	b, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// 2. Inject headers
	return injectHeaders(b, p.Kind(), p.Version())
}

// injectHeaders takes a marshaled payload, unmarshals it into a map,
// injects the standard apiVersion and kind headers, and remarshals it.
func injectHeaders(payloadBytes []byte, kind, version string) ([]byte, error) {
	// Unmarshal into a map to inject headers
	// This creates a flat JSON object: { "apiVersion": "v1", "kind": "...", "metadata": {...}, "data": {...} }
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payloadBytes, &raw); err != nil {
		// This can happen if the payload is not a JSON object (e.g. a raw string)
		return nil, fmt.Errorf("payload is not a JSON object: %w", err)
	}

	// Inject Identity Headers
	verBytes, err := json.Marshal(version)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal version header: %w", err)
	}
	kindBytes, err := json.Marshal(kind)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal kind header: %w", err)
	}

	raw["apiVersion"] = verBytes
	raw["kind"] = kindBytes

	return json.Marshal(raw)
}

// internalHandler wraps the typed logic into a generic byte-processor.
type internalHandler func(oldBlob []byte) (newBlob []byte, nextKind, nextVersion string, err error)

// Migrator manages the chain of schema upgrades.
type Migrator struct {
	handlers  map[string]internalHandler
	pathGraph map[string]string
}

func NewMigrator() *Migrator {
	return &Migrator{
		handlers:  make(map[string]internalHandler),
		pathGraph: make(map[string]string),
	}
}

// Register adds a type-safe migration step to the chain.
//
// Usage:
//
//	blobtypes.Register[SnapshotV1, SnapshotV2](m)
func Register[Current Migratable[Next], Next Payload](m *Migrator) {
	var zeroCurr Current
	var zeroNext Next

	fromKey := makeKey(zeroCurr.Kind(), zeroCurr.Version())
	toKey := makeKey(zeroNext.Kind(), zeroNext.Version())

	if fromKey == toKey {
		panic(fmt.Errorf("%w: cannot migrate %q to itself", ErrInvalidHandlerRegistration, fromKey))
	}

	// Check for duplicates
	if _, exists := m.handlers[fromKey]; exists {
		panic(fmt.Errorf("%w: handler already registered for %q", ErrInvalidHandlerRegistration, fromKey))
	}

	handler := func(blob []byte) ([]byte, string, string, error) {
		// 1. Unmarshal into Current Struct
		var input Current
		if err := json.Unmarshal(blob, &input); err != nil {
			return nil, "", "", fmt.Errorf("failed to unmarshal source %s: %w", fromKey, err)
		}

		// 2. Upgrade (Atomic Data + Meta)
		output, err := input.Upgrade()
		if err != nil {
			return nil, "", "", fmt.Errorf("upgrade %s->%s failed: %w", fromKey, toKey, err)
		}

		// 3. Sanity Check
		outKey := makeKey(output.Kind(), output.Version())
		if outKey != toKey {
			return nil, "", "", fmt.Errorf("migration logic error: expected output %q but got %q", toKey, outKey)
		}

		// 4. Marshal result (Partial JSON)
		outBytes, err := json.Marshal(output)
		if err != nil {
			return nil, "", "", fmt.Errorf("failed to marshal target %s: %w", toKey, err)
		}

		return outBytes, output.Kind(), output.Version(), nil
	}

	m.handlers[fromKey] = handler
	m.pathGraph[fromKey] = toKey
}

// envelopeHeader allows peeking at headers without decoding the body.
type envelopeHeader struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
}

// Apply transforms the FULL blob bytes to match the Target type's schema.
// Usage: blobtypes.Apply[SnapshotV3](m, fullBlobBytes).
func Apply[Target Payload](m *Migrator, fullBlob []byte) ([]byte, error) {
	// 1. Peek at Envelope
	var header envelopeHeader
	if err := json.Unmarshal(fullBlob, &header); err != nil {
		return nil, fmt.Errorf("invalid blob envelope: %w", err)
	}

	var zero Target
	targetKind := zero.Kind()
	targetVer := zero.Version()

	// 2. Run Migration Chain
	finalBytes, finalKind, finalVer, err := m.applyInternal(
		fullBlob, header.Kind, header.APIVersion, targetKind, targetVer)
	if err != nil {
		return nil, err
	}

	// 3. Re-inject Headers for the final blob
	return injectHeaders(finalBytes, finalKind, finalVer)
}

func (m *Migrator) applyInternal(blob []byte, currentKind, currentVersion, targetKind, targetVersion string) (
	[]byte, string, string, error) {
	currKey := makeKey(currentKind, currentVersion)
	targetKey := makeKey(targetKind, targetVersion)

	const maxDepth = 100
	for i := 0; i < maxDepth; i++ {
		if currKey == targetKey {
			return blob, currentKind, currentVersion, nil
		}

		handler, exists := m.handlers[currKey]
		if !exists {
			return nil, "", "", fmt.Errorf("%w: no handler for %q (aiming for %q)",
				ErrMigrationPathNotFound, currKey, targetKey)
		}

		var err error
		var nextKind, nextVersion string

		blob, nextKind, nextVersion, err = handler(blob)
		if err != nil {
			return nil, "", "", err
		}

		currentKind = nextKind
		currentVersion = nextVersion
		currKey = makeKey(nextKind, nextVersion)
	}

	return nil, "", "", fmt.Errorf("%w: halted at %q", ErrMaxMigrationDepth, currKey)
}

// ValidatePath checks if a valid migration chain exists between Start and End types.
func ValidatePath[Start Payload, End Payload](m *Migrator) error {
	var start Start
	var end End

	return m.validatePathInternal(start.Kind(), start.Version(), end.Kind(), end.Version())
}

func (m *Migrator) validatePathInternal(startKind, startVersion, endKind, endVersion string) error {
	currKey := makeKey(startKind, startVersion)
	targetKey := makeKey(endKind, endVersion)
	visited := make(map[string]bool)

	for currKey != targetKey {
		if visited[currKey] {
			return fmt.Errorf("%w: at %q", ErrMigrationCycle, currKey)
		}
		visited[currKey] = true

		nextKey, exists := m.pathGraph[currKey]
		if !exists {
			return fmt.Errorf("%w: broken chain at %q (target: %q)", ErrMigrationPathNotFound, currKey, targetKey)
		}
		currKey = nextKey
	}

	return nil
}

func makeKey(kind, version string) string {
	return fmt.Sprintf("%s:%s", kind, version)
}
