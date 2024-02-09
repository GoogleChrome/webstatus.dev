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

// WPTRunsProcessor contains all the steps for the workflow to process wpt data
// of multiple WPT runs.
type WPTRunsProcessor struct {
	runProcessor RunProcessor
}

type RunProcessor interface {
	ProcessRun(context.Context, shared.TestRun) error
}

func (r WPTRunsProcessor) Start(ctx context.Context, runs shared.TestRuns) error {
	for _, run := range runs {
		err := r.runProcessor.ProcessRun(ctx, run)
		if err != nil {
			return err
		}
	}

	return nil
}
