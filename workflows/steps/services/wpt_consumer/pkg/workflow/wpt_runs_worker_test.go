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
	"errors"
	"testing"
	"time"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

var (
	errGetRuns     = errors.New("mock RunsGetter error")
	errProcessRuns = errors.New("mock RunsProcessor error")
)

type mockGetRunsConfig struct {
	shouldFail bool
	runs       shared.TestRuns
}

type MockRunsGetter struct {
	mockGetRunsCfg mockGetRunsConfig
}

func (m *MockRunsGetter) GetRuns(
	_ context.Context,
	_ time.Time,
	_ int,
	_ string,
	_ string) (shared.TestRuns, error) {
	if m.mockGetRunsCfg.shouldFail {
		return nil, errGetRuns
	}

	return m.mockGetRunsCfg.runs, nil
}

type mockProcessRunsConfig struct {
	shouldFail bool
}

type MockRunsProcessor struct {
	mockProcessRunsCfg mockProcessRunsConfig
}

func (m *MockRunsProcessor) ProcessRuns(_ context.Context, _ shared.TestRuns) error {
	if m.mockProcessRunsCfg.shouldFail {
		return errProcessRuns
	}

	return nil
}

type processWorkflowTest struct {
	name               string
	job                JobArguments
	mockGetRunsCfg     mockGetRunsConfig
	mockProcessRunsCfg mockProcessRunsConfig
	expectedErr        error
}

func TestProcess(t *testing.T) {
	testCases := []processWorkflowTest{
		{
			name: "Successful Workflow",
			job: NewJobArguments(
				time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				"Chrome",
				"stable",
				25,
			),
			mockGetRunsCfg: mockGetRunsConfig{
				// nolint: exhaustruct // WONTFIX: external struct
				runs: []shared.TestRun{
					{
						ID: 0,
					},
				},
				shouldFail: false,
			},
			mockProcessRunsCfg: mockProcessRunsConfig{
				shouldFail: false,
			},
			expectedErr: nil,
		},
		{
			name: "Failed to get runs",
			job: NewJobArguments(
				time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				"Chrome",
				"stable",
				25,
			),
			mockGetRunsCfg: mockGetRunsConfig{
				// nolint: exhaustruct // WONTFIX: external struct
				runs: []shared.TestRun{
					{
						ID: 0,
					},
				},
				shouldFail: true,
			},
			mockProcessRunsCfg: mockProcessRunsConfig{
				shouldFail: false,
			},
			expectedErr: errGetRuns,
		},
		{
			name: "Failed to process runs",
			job: NewJobArguments(
				time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				"Chrome",
				"stable",
				25,
			),
			mockGetRunsCfg: mockGetRunsConfig{
				// nolint: exhaustruct // WONTFIX: external struct
				runs: []shared.TestRun{
					{
						ID: 0,
					},
				},
				shouldFail: false,
			},
			mockProcessRunsCfg: mockProcessRunsConfig{
				shouldFail: true,
			},
			expectedErr: errProcessRuns,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runsGetter := MockRunsGetter{
				mockGetRunsCfg: tc.mockGetRunsCfg,
			}
			runsProcessor := MockRunsProcessor{
				mockProcessRunsCfg: tc.mockProcessRunsCfg,
			}
			processor := NewWPTJobProcessor(&runsGetter, &runsProcessor)

			err := processor.Process(context.Background(), tc.job)
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("Expected error: %v, Got: %v", tc.expectedErr, err)
			}
		})
	}
}
