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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const testSpannerProject = "local"
const testSpannerInstance = "local-instance"
const testSpannerDBName = "test-db"

// nolint: exhaustruct,lll // No need to use every option of 3rd party struct.
func getTestDatabase(t testing.TB) *Client {
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

	ddlFilePath, err := filepath.Abs(filepath.Join(".", "..", "..", "infra", "storage", "spanner.sql"))
	if err != nil {
		t.Error(err)
	}

	verbosityFlag := "--verbosity=debug"

	cmd1 := "gcloud config configurations create emulator"
	cmd2 := "gcloud config configurations activate emulator"
	cmd3 := "gcloud config set disable_prompts true"
	cmd4 := "gcloud config set auth/disable_credentials true"
	cmd5 := "gcloud config set api_endpoint_overrides/spanner http://localhost:" + mappedPort2.Port() + "/"
	cmd6 := "gcloud spanner instances create " + testSpannerInstance + " --config=emulator-config --description=\"Test Instance\" --nodes=1 --project " + testSpannerProject + " " + verbosityFlag
	cmd7 := "gcloud spanner databases create " + testSpannerDBName + " --project " + testSpannerProject + " --instance " + testSpannerInstance + " " + verbosityFlag
	cmd8 := "gcloud spanner databases ddl update " + testSpannerDBName + " --ddl-file " + ddlFilePath + " --project " + testSpannerProject + " --instance " + testSpannerInstance + " " + verbosityFlag

	cmdString := fmt.Sprintf("%s; %s; %s && %s && %s && %s && %s && %s", cmd1, cmd2, cmd3, cmd4, cmd5, cmd6, cmd7, cmd8)
	cmd := exec.Command("/bin/bash", "-c", cmdString)
	t.Logf("debug cmd %s", cmdString)
	cmd.Env = []string{fmt.Sprintf("SPANNER_EMULATOR_HOST=%s", spannerHost)}

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

	return spannerClient
}
