// Copyright 2024 Google LLC
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

package gcpspanner

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const testSpannerProject = "local"
const testSpannerInstance = "local"
const testSpannerDBName = "local"

// nolint:gochecknoglobals // WONTFIX. Used for testing.
var (
	spannerContainer testcontainers.Container
	spannerClient    *Client
	spannerHost      string
)

func TestMain(m *testing.M) {
	err := createDatabaseContainer()
	if err != nil {
		fmt.Printf("failed to create container. error: %s", err.Error())
		os.Exit(1)
	}
	code := m.Run()

	err = terminateDatabaseContainer()
	if err != nil {
		fmt.Printf("Warning: failed to terminate container. error: %s", err.Error())
		os.Exit(1)
	}

	os.Exit(code)
}

// nolint:exhaustruct // WONTFIX: external struct
func createDatabaseContainer() error {
	ctx := context.Background()
	repoRoot, err := filepath.Abs(filepath.Join(".", "..", ".."))
	if err != nil {
		return err
	}

	goarch := runtime.GOARCH
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Dockerfile: filepath.Join(".dev", "spanner", "Dockerfile"),
			Context:    repoRoot,
			BuildArgs:  map[string]*string{"TARGETARCH": &goarch},
			KeepImage:  true,
		},
		ExposedPorts: []string{"9010/tcp"},
		WaitingFor:   wait.ForLog("Spanner setup for webstatus.dev finished"),
		Name:         "webstatus-dev-test-spanner",
	}
	spannerContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return err
	}

	mappedPort, err := spannerContainer.MappedPort(ctx, "9010")
	if err != nil {
		return err
	}

	spannerHost = fmt.Sprintf("localhost:%s", mappedPort.Port())
	err = newTestSpannerClient()
	if err != nil {
		return err
	}

	return nil
}

func newTestSpannerClient() error {
	var err error
	// Set this for the sdk to automatically detect.
	os.Setenv("SPANNER_EMULATOR_HOST", spannerHost)
	spannerClient, err = NewSpannerClient(testSpannerProject, testSpannerInstance, testSpannerDBName)
	if err != nil {
		if unsetErr := os.Unsetenv("SPANNER_EMULATOR_HOST"); unsetErr != nil {
			return fmt.Errorf("failed to unset env. %s", unsetErr.Error())
		}
		spannerClient.Close()
		if terminateErr := spannerContainer.Terminate(context.Background()); terminateErr != nil {
			return fmt.Errorf("failed to terminate datastore. %s", terminateErr.Error())
		}

		return fmt.Errorf("failed to create datastore client. %s", err.Error())
	}

	return nil
}

func restartDatabaseContainer(t *testing.T) {
	spannerClient.Close()
	spannerClient = nil
	// Wait 30 seconds for the command to finish
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, _, err := spannerContainer.Exec(ctx, []string{
		"bash",
		"-c",
		"wrench reset --directory ./schemas/ && wrench migrate up --directory ./schemas/",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = newTestSpannerClient()
	if err != nil {
		t.Fatal(err)
	}
}

func terminateDatabaseContainer() error {
	if unsetErr := os.Unsetenv("SPANNER_EMULATOR_HOST"); unsetErr != nil {
		return fmt.Errorf("failed to unset env. %s", unsetErr.Error())
	}
	spannerClient.Close()
	if err := spannerContainer.Terminate(context.Background()); err != nil {
		return fmt.Errorf("failed to terminate datastore. %s", err.Error())
	}

	return nil
}

// This also tests the success path of NewSpannerClient.
func TestGetTestDatabase(t *testing.T) {
	restartDatabaseContainer(t)
	if spannerClient == nil {
		t.Error("exepected a client")
	}
}

func TestNewSpannerClient_Bad(t *testing.T) {
	testCases := []struct {
		testName      string
		projectID     string
		instanceID    string
		name          string
		expectedError error
	}{
		{
			testName:      "missing project ID",
			projectID:     "",
			instanceID:    "foo",
			name:          "foo",
			expectedError: ErrBadClientConfig,
		},
		{
			testName:      "missing instance ID",
			projectID:     "foo",
			instanceID:    "",
			name:          "foo",
			expectedError: ErrBadClientConfig,
		},
		{
			testName:      "missing name",
			projectID:     "foo",
			instanceID:    "foo",
			name:          "",
			expectedError: ErrBadClientConfig,
		},
		{
			testName:      "unable to establish client by using a bad character",
			projectID:     "foo",
			instanceID:    "foo",
			name:          "foo/",
			expectedError: ErrFailedToEstablishClient,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			client, err := NewSpannerClient(tc.projectID, tc.instanceID, tc.name)
			if client != nil {
				t.Error("expectetd nil client")
			}
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("expected error %s. received %s", tc.expectedError, err)
			}
		})
	}
}

func TestDecodeCursor_Bad(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedCursor *any
		expectedError  error
	}{
		{
			name:           "invalid base64",
			input:          "not-base64",
			expectedCursor: nil,
			expectedError:  ErrInvalidCursorFormat,
		},
		{
			name:           "invalid json",
			input:          base64.RawURLEncoding.EncodeToString([]byte("invalid-json")),
			expectedCursor: nil,
			expectedError:  ErrInvalidCursorFormat,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cursor, err := decodeCursor[any](tc.input)
			if !reflect.DeepEqual(tc.expectedCursor, cursor) {
				t.Error("unexpected cursor")
			}
			if !errors.Is(err, tc.expectedError) {
				t.Error("unexpected error")
			}
		})
	}
}

func TestDecodeWPTRunCursor(t *testing.T) {
	in := "eyJsYXN0X3RpbWVfc3RhcnQiOiIyMDAwLTAxLTAxVDAwOjAwOjAwWiIsImxhc3RfcnVuX2lkIjoxMDB9"
	cursor, err := decodeWPTRunCursor(in)
	if !errors.Is(err, nil) {
		t.Errorf("expected no error. received %s", err.Error())
	}
	expectedCursor := WPTRunCursor{
		LastTimeStart: time.Date(2000, time.January, 1, 0, 0, 00, 0, time.UTC),
		LastRunID:     100,
	}
	if !reflect.DeepEqual(expectedCursor, *cursor) {
		t.Errorf("unequal cursors expected %+v. received %+v", expectedCursor, *cursor)
	}
}

func TestEncodeWPTRunCursor(t *testing.T) {
	value := encodeWPTRunCursor(
		time.Date(2000, time.January, 1, 0, 0, 00, 0, time.UTC),
		100,
	)
	expected := "eyJsYXN0X3RpbWVfc3RhcnQiOiIyMDAwLTAxLTAxVDAwOjAwOjAwWiIsImxhc3RfcnVuX2lkIjoxMDB9"
	if expected != value {
		t.Errorf("unexpected wpt run cursor. received %s, expected %s", value, expected)
	}
}
