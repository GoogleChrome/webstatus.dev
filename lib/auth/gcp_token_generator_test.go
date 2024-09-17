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

package auth

import (
	"context"
	"testing"
)

// An optional integration test to show that the library can get a token for a particular GCP resource.
func TestGCPTokenGenerator_Generate(t *testing.T) {
	// According to the docs, this will not work locally. Has to be run on GCP infrastructure.
	// https://cloud.google.com/docs/authentication/get-id-token#metadata-server
	t.Skip("need to be logged in with GCP to run this test. Skip by default")
	generator := GCPTokenGenerator{}
	token, err := generator.Generate(context.Background(), "https://uma-export.appspot.com/webstatus/")
	if err != nil {
		t.Errorf("unable to get token, err %s", err)
	}
	if token == nil {
		t.Error("expected non nil token")
	}
}
