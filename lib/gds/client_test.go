// Copyright 2023 Google LLC
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

package gds

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const testDatastoreProject = "local"

// nolint: exhaustruct // No need to use every option of 3rd party struct.
func getTestDatabase(ctx context.Context, t *testing.T) (*Client, func()) {
	datastoreFolder, err := filepath.Abs(filepath.Join(".", "..", "..", ".dev", "datastore"))
	if err != nil {
		t.Fatal(err)
	}
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Dockerfile: filepath.Join("Dockerfile"),
			Context:    datastoreFolder,
		},
		ExposedPorts: []string{"8085/tcp"},
		WaitingFor:   wait.ForHTTP("/").WithPort("8085/tcp"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal(err)
	}

	mappedPort, err := container.MappedPort(ctx, "8085")
	if err != nil {
		t.Fatal(err)
	}

	db := ""
	dbPtr := &db
	os.Setenv("DATASTORE_EMULATOR_HOST", fmt.Sprintf("localhost:%s", mappedPort.Port()))
	dsClient, err := NewDatastoreClient(testDatastoreProject, dbPtr)
	if err != nil {
		if unsetErr := os.Unsetenv("DATASTORE_EMULATOR_HOST"); unsetErr != nil {
			t.Errorf("failed to unset env. %s", unsetErr.Error())
		}
		if closeErr := dsClient.Close(); closeErr != nil {
			t.Errorf("failed to close datastore client. %s", closeErr.Error())
		}
		if terminateErr := container.Terminate(ctx); terminateErr != nil {
			t.Errorf("failed to terminate datastore. %s", terminateErr.Error())
		}
		t.Fatalf("failed to create datastore client. %s", err.Error())
	}

	return dsClient, func() {
		if unsetErr := os.Unsetenv("DATASTORE_EMULATOR_HOST"); unsetErr != nil {
			t.Errorf("failed to unset env. %s", unsetErr.Error())
		}
		if err := dsClient.Close(); err != nil {
			t.Errorf("failed to close datastore client. %s", err.Error())
		}
		if err := container.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate datastore. %s", err.Error())
		}
	}
}

const sampleKey = "SampleData"

type TestSample struct {
	Name      string    `datastore:"name"`
	Value     *int      `datastore:"value"`
	CreatedAt time.Time `datastore:"created_at"`
}

type nameFilter struct {
	name string
}

func (f nameFilter) FilterQuery(query *datastore.Query) *datastore.Query {
	return query.FilterField("name", "=", f.name)
}

type sortSampleFilter struct {
}

func (f sortSampleFilter) FilterQuery(query *datastore.Query) *datastore.Query {
	return query.Order("-created_at")
}

type limitSampleFilter struct {
	size int
}

func (f limitSampleFilter) FilterQuery(query *datastore.Query) *datastore.Query {
	return query.Limit(f.size)
}

// testSampleMerge implements Mergeable for TestSample.
type testSampleMerge struct{}

func (m testSampleMerge) Merge(existing *TestSample, new *TestSample) *TestSample {
	return &TestSample{
		Value: cmp.Or[*int](new.Value, existing.Value),
		// The below fields cannot be overridden during a merge.
		Name:      existing.Name,
		CreatedAt: existing.CreatedAt,
	}
}

func intPtr(in int) *int {
	return &in
}

// nolint: gochecknoglobals
var testSamples = []TestSample{
	{
		Name:      "a",
		Value:     intPtr(0),
		CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
	},
	{
		Name:      "b",
		Value:     intPtr(1),
		CreatedAt: time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC),
	},
	{
		Name:      "c",
		Value:     intPtr(2),
		CreatedAt: time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC),
	},
	{
		Name:      "d",
		Value:     intPtr(3),
		CreatedAt: time.Date(2002, time.January, 1, 0, 0, 0, 0, time.UTC),
	},
}

func insertEntities(
	ctx context.Context,
	t *testing.T,
	c entityClient[TestSample]) {
	for i := range testSamples {
		err := c.upsert(ctx, sampleKey, &testSamples[i], testSampleMerge{}, nameFilter{name: testSamples[i].Name})
		if err != nil {
			t.Fatalf("failed to insert entities. %s", err.Error())
		}
	}
}

func TestEntityClientOperations(t *testing.T) {
	ctx := context.Background()
	client, cleanup := getTestDatabase(ctx, t)
	defer cleanup()
	c := entityClient[TestSample]{client}
	// Step 1. Make sure the entity is not there yet.
	// Step 1a. Do Get
	entity, err := c.get(ctx, sampleKey, nameFilter{name: "a"})
	if entity != nil {
		t.Error("expected no entity")
	}
	if !errors.Is(err, ErrEntityNotFound) {
		t.Error("expected ErrEntityNotFound")
	}
	// Step 1b. Do List
	pageEmpty, nextPageToken, err := c.list(ctx, sampleKey, nil)
	if err != nil {
		t.Errorf("list query failed. %s", err.Error())
	}
	if nextPageToken == nil {
		t.Error("expected next page token")
	}
	if pageEmpty != nil {
		t.Error("expected empty page")
	}
	// Step 2. Insert the entities
	insertEntities(ctx, t, c)
	// Step 3. Get the entity
	entity, err = c.get(ctx, sampleKey, nameFilter{name: "a"})
	if err != nil {
		t.Errorf("expected error %s", err.Error())
	}
	if entity == nil {
		t.Error("expected entity")
		t.FailNow()
	}
	if !reflect.DeepEqual(*entity, TestSample{
		Name:      "a",
		Value:     intPtr(0),
		CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
	}) {
		t.Errorf("values not equal. received %+v", *entity)
	}
	// Step 4. Upsert the entity
	entity.Value = intPtr(200)
	// CreatedAt should not update due to the Mergeable policy
	entity.CreatedAt = time.Date(3000, time.March, 1, 0, 0, 0, 0, time.UTC)
	err = c.upsert(ctx, sampleKey, entity, testSampleMerge{}, nameFilter{name: "a"})
	if err != nil {
		t.Errorf("upsert failed %s", err.Error())
	}
	// Step 5. Get the updated entity
	entity, err = c.get(ctx, sampleKey, nameFilter{name: "a"})
	if err != nil {
		t.Errorf("expected error %s", err.Error())
	}
	if entity == nil {
		t.Error("expected entity")
		t.FailNow()
	}
	if !reflect.DeepEqual(*entity, TestSample{
		Name:      "a",
		Value:     intPtr(200),
		CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
	}) {
		t.Errorf("values not equal. received %+v", *entity)
	}
	// Step 6. List the entities
	// Step 6a. Get first page
	filters := []Filterable{sortSampleFilter{}, limitSampleFilter{size: 2}}
	pageOne, nextPageToken, err := c.list(ctx, sampleKey, nil, filters...)
	if err != nil {
		t.Errorf("page one query failed. %s", err.Error())
	}
	if nextPageToken == nil {
		t.Error("expected next page token")
	}
	expectedPageOne := []*TestSample{
		{
			Name:      "d",
			Value:     intPtr(3),
			CreatedAt: time.Date(2002, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Name:      "c",
			Value:     intPtr(2),
			CreatedAt: time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	if !reflect.DeepEqual(pageOne, expectedPageOne) {
		t.Error("values not equal")
	}
	// Step 6b. Get second page
	pageTwo, nextPageToken, err := c.list(ctx, sampleKey, nextPageToken, filters...)
	if err != nil {
		t.Errorf("page two query failed. %s", err.Error())
	}
	if nextPageToken == nil {
		t.Error("expected next page token")
	}
	expectedPageTwo := []*TestSample{
		{
			Name:      "a",
			Value:     intPtr(200),
			CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Name:      "b",
			Value:     intPtr(1),
			CreatedAt: time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	if !reflect.DeepEqual(pageTwo, expectedPageTwo) {
		t.Error("values not equal")
	}
}
