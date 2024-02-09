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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

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
