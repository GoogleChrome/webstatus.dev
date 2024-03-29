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

package workflow

import (
	"context"
	"testing"
)

func TestHTTPResultsGetter(t *testing.T) {
	g := NewHTTPResultsGetter()
	// nolint:lll
	sampleURL := "https://storage.googleapis.com/wptd/9593290ad1bd621f74c697c7cc347348af2de32a/chrome-117.0.5938.62-linux-20.04-ddee0c57b6-summary_v2.json.gz"
	_, err := g.DownloadResults(context.Background(), sampleURL)
	if err != nil {
		t.Errorf("unexpected error during download. %s", err.Error())
	}
}
