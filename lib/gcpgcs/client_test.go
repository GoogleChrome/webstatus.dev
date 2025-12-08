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

package gcpgcs

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/blobtypes"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// nolint:gochecknoglobals // WONTFIX. Used for testing.
var (
	gcsContainer testcontainers.Container
	gcsClient    *Client
	gcsHost      string
)

const testBucket = "local"

func TestMain(m *testing.M) {
	err := createGCSContainer()
	if err != nil {
		fmt.Printf("failed to create container. error: %s", err.Error())
		os.Exit(1)
	}
	code := m.Run()
	err = terminateGCSContainer()
	if err != nil {
		fmt.Printf("Warning: failed to terminate container. error: %s", err.Error())
		os.Exit(1)
	}
	os.Exit(code)
}

// nolint:exhaustruct // WONTFIX: external struct
func createGCSContainer() error {
	ctx := context.Background()
	repoRoot, err := filepath.Abs(filepath.Join(".", "..", ".."))
	if err != nil {
		return err
	}

	goarch := runtime.GOARCH
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Dockerfile: filepath.Join(".dev", "gcs", "Dockerfile"),
			Context:    repoRoot,
			BuildArgs:  map[string]*string{"TARGETARCH": &goarch},
			KeepImage:  true,
		},
		ExposedPorts: []string{"4443/tcp"},
		WaitingFor:   wait.ForListeningPort("4443/tcp"),
		Name:         "webstatus-dev-test-gcs",
		// Needed because of https://github.com/fsouza/fake-gcs-server/issues/982
		Cmd: []string{"-scheme", "http", "-public-host", "localhost", "-backend", "memory"},
	}
	gcsContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return err
	}

	mappedPort, err := gcsContainer.MappedPort(ctx, "4443")
	if err != nil {
		return err
	}

	gcsHost = fmt.Sprintf("localhost:%s", mappedPort.Port())
	err = newTestGCSClient()
	if err != nil {
		return err
	}

	return nil
}

func newTestGCSClient() error {
	var err error
	// Set this for the sdk to automatically detect.
	os.Setenv("STORAGE_EMULATOR_HOST", gcsHost)
	gcsClient, err = NewClient(context.Background(), testBucket)
	if err != nil {
		if unsetErr := os.Unsetenv("STORAGE_EMULATOR_HOST"); unsetErr != nil {
			return fmt.Errorf("failed to unset env. %s", unsetErr.Error())
		}
		gcsClient.Close()
		if terminateErr := gcsContainer.Terminate(context.Background()); terminateErr != nil {
			return fmt.Errorf("failed to terminate container. %s", terminateErr.Error())
		}

		return fmt.Errorf("failed to create client. %s", err.Error())
	}

	bucket := gcsClient.client.Bucket(testBucket)
	err = bucket.Create(context.Background(), "testproject", nil)
	if err != nil {
		return fmt.Errorf("failed to create test bucket. %s", err.Error())
	}

	return nil
}

func terminateGCSContainer() error {
	if unsetErr := os.Unsetenv("STORAGE_EMULATOR_HOST"); unsetErr != nil {
		return fmt.Errorf("failed to unset env. %s", unsetErr.Error())
	}
	gcsClient.Close()
	if err := gcsContainer.Terminate(context.Background()); err != nil {
		return fmt.Errorf("failed to terminate datastore. %s", err.Error())
	}

	return nil
}

func TestWriteReadBlob(t *testing.T) {
	ctx := context.Background()
	path := "test-blob.txt"
	data := []byte("Hello, GCS!")
	contentType := "text/plain"
	metadata := map[string]string{"Foo": "BAR"}

	t.Cleanup(func() {
		// Clean up the blob after the test.
		err := gcsClient.client.Bucket(testBucket).Object(path).Delete(ctx)
		if err != nil {
			t.Logf("Failed to delete test blob %s: %v", path, err)
		}
	})

	// Test WriteBlob
	err := gcsClient.WriteBlob(ctx, path, data,
		blobtypes.WithContentType(contentType),
		blobtypes.WithMetadata(metadata),
	)
	if err != nil {
		t.Fatalf("WriteBlob failed: %v", err)
	}

	// Test ReadBlob
	blob, err := gcsClient.ReadBlob(ctx, path)
	if err != nil {
		t.Fatalf("ReadBlob failed: %v", err)
	}

	if string(blob.Data) != string(data) {
		t.Errorf("Read data mismatch: got %q, want %q", string(blob.Data), string(data))
	}
	if blob.ContentType != contentType {
		t.Errorf("Read ContentType mismatch: got %q, want %q", blob.ContentType, contentType)
	}
	if blob.Generation == 0 {
		t.Errorf("Read Generation should not be 0")
	}
	if !maps.Equal(metadata, blob.Metadata) {
		t.Errorf("Read Metadata mismatch: got %v, want %v", blob.Metadata, metadata)
	}
}

func TestReadBlob_NotFound(t *testing.T) {
	ctx := context.Background()
	path := "non-existent-blob.txt"

	_, err := gcsClient.ReadBlob(ctx, path)
	if !errors.Is(err, blobtypes.ErrBlobNotFound) {
		t.Errorf("Expected ErrBlobNotFound, got %v", err)
	}
}

func TestWriteBlob_PreconditionFailed(t *testing.T) {
	ctx := context.Background()
	path := "precondition-test-blob.txt"
	data := []byte("some data")

	t.Cleanup(func() {
		// Clean up the blob after the test.
		err := gcsClient.client.Bucket(testBucket).Object(path).Delete(ctx)
		if err != nil {
			t.Logf("Failed to delete test blob %s: %v", path, err)
		}
	})

	// First write should succeed with DoesNotExist condition
	generation := int64(0)
	err := gcsClient.WriteBlob(ctx, path, data, blobtypes.WithExpectedGeneration(generation))
	if err != nil {
		t.Fatalf("First WriteBlob failed: %v", err)
	}

	// Second write with the same condition should fail
	err = gcsClient.WriteBlob(ctx, path, []byte("new data"), blobtypes.WithExpectedGeneration(generation))
	if !errors.Is(err, blobtypes.ErrPreconditionFailed) {
		t.Errorf("Expected ErrPreconditionFailed on second write, got %v", err)
	}

	// Read the blob to get its current generation
	blob, err := gcsClient.ReadBlob(ctx, path)
	if err != nil {
		t.Fatalf("ReadBlob failed: %v", err)
	}
	// Check the contents to ensure it wasn't overwritten
	if string(blob.Data) != string(data) {
		t.Errorf("Blob data was modified unexpectedly: got %q, want %q", string(blob.Data), string(data))
	}

	// Now try writing with the correct current generation
	err = gcsClient.WriteBlob(ctx, path, []byte("updated data"), blobtypes.WithExpectedGeneration(blob.Generation))
	if err != nil {
		t.Fatalf("WriteBlob with correct generation failed: %v", err)
	}

	// Verify the update
	updatedBlob, err := gcsClient.ReadBlob(ctx, path)
	if err != nil {
		t.Fatalf("ReadBlob after update failed: %v", err)
	}
	if string(updatedBlob.Data) != "updated data" {
		t.Errorf("Blob data mismatch after update: got %q, want %q", string(updatedBlob.Data), "updated data")
	}
}
