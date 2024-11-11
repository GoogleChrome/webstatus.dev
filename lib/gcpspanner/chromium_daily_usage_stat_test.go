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
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestListChromiumDailyUsageStatsForFeatureID(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	// TODO(DanielRyanSmith): Change tests when fake data is no longer used.
	stats, token, err := spannerClient.ListChromiumDailyUsageStatsForFeatureID(
		ctx,
		"feature1",
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		10,
		nil,
	)

	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token != nil {
		t.Error("expected null token")
	}
	expectedStats := []ChromiumDailyUsageStatWithTime{
		{
			Date:  time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			Usage: nil,
		},
	}

	if !reflect.DeepEqual(expectedStats, stats) {
		t.Errorf("unequal metrics. expected (%+v) received (%+v) ", expectedStats, stats)
	}
}
