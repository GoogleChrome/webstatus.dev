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
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

func TestUpsertDailyChromiumHistogramCapstone(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	sampleEnums := getSampleChromiumHistogramEnums()
	insertTestChromiumHistogramEnums(ctx, spannerClient, t, sampleEnums)

	in := DailyChromiumHistogramEnumCapstone{
		HistogramName: metricdatatypes.WebDXFeatureEnum,
		Day: civil.Date{
			Year:  2000,
			Month: time.January,
			Day:   3,
		},
	}
	// Test absence
	found, err := spannerClient.HasDailyChromiumHistogramCapstone(ctx, in)
	if err != nil {
		t.Errorf("unable to get capstone. error %s", err)
	}

	if *found {
		t.Error("expected false")
	}

	// Insert capstone
	err = spannerClient.UpsertDailyChromiumHistogramCapstone(ctx, in)
	if err != nil {
		t.Errorf("unable to upsert capstone. error %s", err)
	}

	// Test presence
	found, err = spannerClient.HasDailyChromiumHistogramCapstone(ctx, in)
	if err != nil {
		t.Errorf("unable to get capstone. error %s", err)
	}

	if !*found {
		t.Error("expected true")
	}
}
