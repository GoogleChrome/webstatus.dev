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
	"errors"
	"testing"
)

const testDataKind = "TestData"

// --- Mocks: Version 1 ---

type MetaV1 struct {
	Timestamp string `json:"ts"`
}
type DataV1 struct {
	Name string `json:"name"`
}

// SnapshotV1 represents the full content of a V1 blob (Data + Metadata).
type SnapshotV1 struct {
	Meta MetaV1 `json:"metadata"`
	Data DataV1 `json:"data"`
}

func (s SnapshotV1) Kind() string    { return testDataKind }
func (s SnapshotV1) Version() string { return "v1" }

// Upgrade implements Migratable[SnapshotV2].
func (s SnapshotV1) Upgrade() (SnapshotV2, error) {
	return SnapshotV2{
		Meta: MetaV2{Epoch: 100},
		Data: DataV2{FullName: s.Data.Name},
	}, nil
}

// --- Mocks: Version 2 ---

type MetaV2 struct {
	Epoch int `json:"epoch"`
}
type DataV2 struct {
	FullName string `json:"full_name"`
}

type SnapshotV2 struct {
	Meta MetaV2 `json:"metadata"`
	Data DataV2 `json:"data"`
}

func (s SnapshotV2) Kind() string    { return testDataKind }
func (s SnapshotV2) Version() string { return "v2" }

// Upgrade implements Migratable[SnapshotV3].
func (s SnapshotV2) Upgrade() (SnapshotV3, error) {
	return SnapshotV3{
		Meta: s.Meta,
		Data: DataV3{First: s.Data.FullName},
	}, nil
}

// Mocks: Version 3 (Target).

type DataV3 struct {
	First string `json:"first"`
}

type SnapshotV3 struct {
	Meta MetaV2 `json:"metadata"`
	Data DataV3 `json:"data"`
}

func (s SnapshotV3) Kind() string    { return testDataKind }
func (s SnapshotV3) Version() string { return "v3" }

// Mocks: Cycle.
const cycleKind = "Cycle"

type CycleA struct{}

func (c CycleA) Kind() string             { return cycleKind }
func (c CycleA) Version() string          { return "A" }
func (c CycleA) Upgrade() (CycleB, error) { return CycleB{}, nil }

type CycleB struct{}

func (c CycleB) Kind() string             { return cycleKind }
func (c CycleB) Version() string          { return "B" }
func (c CycleB) Upgrade() (CycleA, error) { return CycleA{}, nil }

// CycleC is a dummy target to force graph traversal.
type CycleC struct{}

func (c CycleC) Kind() string    { return cycleKind }
func (c CycleC) Version() string { return "C" }

func TestMigrationChain(t *testing.T) {
	m := NewMigrator()

	// 1. Registration
	Register[SnapshotV1, SnapshotV2](m)
	Register[SnapshotV2, SnapshotV3](m)

	// 2. Validate Path
	if err := ValidatePath[SnapshotV1, SnapshotV3](m); err != nil {
		t.Fatalf("ValidatePath failed: %v", err)
	}

	// 3. Create Source Blob (V1)
	v1State := SnapshotV1{
		Meta: MetaV1{Timestamp: "2023-01-01"},
		Data: DataV1{Name: "Alice"},
	}

	blobBytes, err := NewBlob(v1State)
	if err != nil {
		t.Fatalf("NewBlob failed: %v", err)
	}

	// 4. Apply Migration (V1 -> V3)
	migratedBytes, err := Apply[SnapshotV3](m, blobBytes)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// 5. Verify Output
	var raw map[string]interface{}
	if err := json.Unmarshal(migratedBytes, &raw); err != nil {
		t.Fatalf("Unmarshal result failed: %v", err)
	}

	if raw["apiVersion"] != "v3" {
		t.Errorf("Version header incorrect. Got %v", raw["apiVersion"])
	}

	var res SnapshotV3
	if err := json.Unmarshal(migratedBytes, &res); err != nil {
		t.Fatalf("Unmarshal struct failed: %v", err)
	}

	if res.Data.First != "Alice" {
		t.Errorf("Data migration incorrect. Got %q, want 'Alice'", res.Data.First)
	}
	if res.Meta.Epoch != 100 {
		t.Errorf("Metadata migration incorrect. Got %d", res.Meta.Epoch)
	}
}

func TestBrokenChain(t *testing.T) {
	m := NewMigrator()
	Register[SnapshotV1, SnapshotV2](m)

	err := ValidatePath[SnapshotV1, SnapshotV3](m)
	if !errors.Is(err, ErrMigrationPathNotFound) {
		t.Errorf("Expected ErrMigrationPathNotFound, got %v", err)
	}
}

func TestCycleDetection(t *testing.T) {
	m := NewMigrator()
	Register[CycleA, CycleB](m)
	Register[CycleB, CycleA](m)

	// Attempt to go from A to C. C is not reachable, but A->B->A is a loop.
	// The validator should traverse A -> B -> A and detect the visited node A.
	err := ValidatePath[CycleA, CycleC](m)
	if !errors.Is(err, ErrMigrationCycle) {
		t.Errorf("Expected ErrMigrationCycle, got %v", err)
	}
}

func TestApply_NoOp(t *testing.T) {
	m := NewMigrator()

	v3State := SnapshotV3{Data: DataV3{First: "Direct"}, Meta: MetaV2{Epoch: 100}}
	blobBytes, err := NewBlob(v3State)
	if err != nil {
		t.Fatalf("NewBlob failed: %v", err)
	}

	resBytes, err := Apply[SnapshotV3](m, blobBytes)
	if err != nil {
		t.Fatalf("Apply no-op failed: %v", err)
	}

	var res SnapshotV3
	err = json.Unmarshal(resBytes, &res)
	if err != nil {
		t.Fatalf("Unmarshal result failed: %v", err)
	}

	if res.Data.First != "Direct" {
		t.Errorf("Data corrupted in no-op")
	}
}

func TestRegister_Conflict(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic on duplicate registration")
		}
	}()

	m := NewMigrator()
	Register[SnapshotV1, SnapshotV2](m)
	Register[SnapshotV1, SnapshotV2](m) // Duplicate registration
}
