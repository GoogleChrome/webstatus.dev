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
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const testSpannerProject = "local"
const testSpannerInstance = "local-instance"
const testSpannerDBName = "test-db"
const projectFlag = "--project " + testSpannerProject
const projectAndVerbosityFlag = projectFlag + " --verbosity=debug"

type testMigrationHandler struct {
	ddlFolderAbsPath string
	files            []string
}

func newTestMigrationHandler(ddlFolderAbsPath string) *testMigrationHandler {
	return &testMigrationHandler{
		ddlFolderAbsPath: ddlFolderAbsPath,
		files:            []string{"spanner.sql"},
	}
}

func (h testMigrationHandler) ApplyAll(t *testing.T) {
	for _, file := range h.files {
		cmdString := fmt.Sprintf(
			"gcloud spanner databases ddl update %s --ddl-file %s --instance %s %s",
			testSpannerDBName,
			filepath.Join(h.ddlFolderAbsPath, file),
			testSpannerInstance,
			projectAndVerbosityFlag,
		)
		t.Logf("debug database migration for %s cmd %s", file, cmdString)
		cmd := exec.Command("/bin/bash", "-c", cmdString)
		cmd.Env = os.Environ()
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("gcloud command failed: %v, output: %s", err, output)
		}
	}
}

// nolint: exhaustruct,lll // No need to use every option of 3rd party struct.
func getTestDatabase(t testing.TB) (*Client, *testMigrationHandler) {
	ctx := context.Background()
	spannerFolder, err := filepath.Abs(filepath.Join(".", "..", "..", ".dev", "spanner"))
	if err != nil {
		t.Error(err)
	}
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Dockerfile: filepath.Join("Dockerfile"),
			Context:    spannerFolder,
		},
		ExposedPorts: []string{"9010/tcp", "9020/tcp"},
		WaitingFor:   wait.ForLog("Cloud Spanner emulator running"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Error(err)
	}

	mappedPort, err := container.MappedPort(ctx, "9010")
	if err != nil {
		t.Error(err)
	}

	mappedPort2, err := container.MappedPort(ctx, "9020")
	if err != nil {
		t.Error(err)
	}

	spannerHost := fmt.Sprintf("localhost:%s", mappedPort.Port())
	// Set this for the sdk to automatically detect.
	os.Setenv("SPANNER_EMULATOR_HOST", spannerHost)
	spannerClient, err := NewSpannerClient(testSpannerProject, testSpannerInstance, testSpannerDBName)
	if err != nil {
		if unsetErr := os.Unsetenv("SPANNER_EMULATOR_HOST"); unsetErr != nil {
			t.Errorf("failed to unset env. %s", unsetErr.Error())
		}
		spannerClient.Close()
		if terminateErr := container.Terminate(ctx); terminateErr != nil {
			t.Errorf("failed to terminate datastore. %s", terminateErr.Error())
		}
		t.Fatalf("failed to create datastore client. %s", err.Error())
	}

	ddlFilePath, err := filepath.Abs(filepath.Join(".", "..", "..", "infra", "storage"))
	if err != nil {
		t.Error(err)
	}

	cmdString := fmt.Sprintf(`
		gcloud config configurations create emulator; \
		gcloud config configurations activate emulator; \
		gcloud config set disable_prompts true && \
		gcloud config set auth/disable_credentials true && \
		gcloud config set api_endpoint_overrides/spanner http://localhost:%s/ && \
		gcloud spanner instances create %s --config=emulator-config --description='Test Instance' --nodes=1 %s && \
		gcloud spanner databases create %s --instance %s %s
	`,
		// Mapped port for api_endpoint_override
		mappedPort2.Port(),
		// Spanner instance name for create
		testSpannerInstance,
		projectAndVerbosityFlag,
		// Spanner database name for create
		testSpannerDBName,
		testSpannerInstance,
		projectAndVerbosityFlag,
	)

	cmd := exec.Command("/bin/bash", "-c", cmdString)
	t.Logf("debug database startup cmd %s", cmdString)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("gcloud command failed: %v, output: %s", err, output)
	}

	t.Cleanup(func() {
		if unsetErr := os.Unsetenv("SPANNER_EMULATOR_HOST"); unsetErr != nil {
			t.Errorf("failed to unset env. %s", unsetErr.Error())
		}
		spannerClient.Close()
		if err := container.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate datastore. %s", err.Error())
		}
	})

	return spannerClient, newTestMigrationHandler(ddlFilePath)
}

// This also tests the success path of NewSpannerClient.
func TestGetTestDatabase(t *testing.T) {
	client, migrationHandler := getTestDatabase(t)
	if client == nil {
		t.Error("exepected a client")
	}
	// Attempt to apply all the migrations
	migrationHandler.ApplyAll(t)
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
