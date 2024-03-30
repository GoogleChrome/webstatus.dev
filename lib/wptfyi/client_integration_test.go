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

package wptfyi

import (
	"context"
	"testing"
	"time"
)

func TestGetRunsIntegration(t *testing.T) {
	client := NewHTTPClient("wpt.fyi")
	pageSize := 5
	runs, err := client.GetRuns(context.TODO(), time.Now().AddDate(0, 0, -365).UTC(), pageSize, "chrome", "stable")
	if err != nil {
		t.Errorf("unexpected error getting runs: %s\n", err.Error())
	}
	if len(runs) != pageSize {
		t.Errorf("unexpected page size. %d", len(runs))
	}
}
