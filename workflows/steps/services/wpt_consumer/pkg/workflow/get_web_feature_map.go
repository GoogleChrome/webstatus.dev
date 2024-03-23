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

	"github.com/web-platform-tests/wpt.fyi/shared"
)

func NewGitHubWebFeaturesDataGetter(client *shared.GitHubWebFeaturesClient) *GitHubWebFeaturesDataGetter {
	return &GitHubWebFeaturesDataGetter{client: client}
}

type GitHubWebFeaturesDataGetter struct {
	client *shared.GitHubWebFeaturesClient
}

func (g GitHubWebFeaturesDataGetter) GetWebFeaturesData(ctx context.Context) (shared.WebFeaturesData, error) {
	// TODO. cache the result
	return g.client.Get(ctx)
}
