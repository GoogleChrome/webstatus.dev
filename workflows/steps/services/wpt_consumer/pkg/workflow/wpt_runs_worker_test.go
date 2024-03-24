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
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

type MockWorker struct {
	processJobs            []workflowArguments
	mockProcessWorkflowCfg mockProcessWorkflowConfig
}

type mockProcessWorkflowConfig struct {
	shouldFail bool
}

func (m *MockWorker) processWorkflow(_ context.Context, job workflowArguments) error {
	if m.mockProcessWorkflowCfg.shouldFail {
		return errMockWriterFail
	}
	m.processJobs = append(m.processJobs, job)

	return nil
}

var errMockWriterFail = errors.New("mock writer test erro")

type startWorkerTest struct {
	name                   string
	jobs                   []workflowArguments
	expectedErrs           []error // Errors expected on the error channel
	mockProcessWorkflowCfg mockProcessWorkflowConfig
	expectJobs             []workflowArguments // To check if jobs were passed correctly
}

// nolint: gocognit // TODO. Refactor test to make it clearer
func TestStart(t *testing.T) {
	testCases := []startWorkerTest{
		{
			name: "Successful Jobs",
			jobs: []workflowArguments{
				{
					from:    time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
					browser: "Chrome",
					channel: "stable",
				},
				{
					from:    time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC),
					browser: "Firefox",
					channel: "beta",
				},
			},
			mockProcessWorkflowCfg: mockProcessWorkflowConfig{
				shouldFail: false,
			},
			expectJobs: []workflowArguments{
				{
					from:    time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
					browser: "Chrome",
					channel: "stable",
				},
				{
					from:    time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC),
					browser: "Firefox",
					channel: "beta",
				},
			},
			expectedErrs: nil,
		},
		{
			name: "Worker Failure",
			jobs: []workflowArguments{
				{from: time.Now(), browser: "Chrome", channel: "stable"},
			},
			mockProcessWorkflowCfg: mockProcessWorkflowConfig{
				shouldFail: true,
			},
			expectedErrs: []error{errMockWriterFail},
			expectJobs:   []workflowArguments{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testJobs := make(chan workflowArguments)
			testErrChan := make(chan error)
			testWg := &sync.WaitGroup{}

			manager := RunsWorkerManager{}

			ctx, cancelFunc := context.WithCancel(context.Background()) // For potential cancellation tests
			defer cancelFunc()                                          // Ensure cleanup

			testWg.Add(1)
			worker := MockWorker{
				processJobs:            []workflowArguments{},
				mockProcessWorkflowCfg: tc.mockProcessWorkflowCfg,
			}
			go manager.Start(ctx, 1, testWg, testJobs, testErrChan, &worker)

			// Send jobs
			for _, job := range tc.jobs {
				testJobs <- job
			}
			close(testJobs)

			done := make(chan struct{}) // Signal completion
			go func() {
				testWg.Wait()
				done <- struct{}{}
				close(done)
			}()

			// Assertions
			receivedErrors := []error{} // Collect errors
			isDone := false
			for {
				select {
				case err := <-testErrChan:
					if err != nil {
						receivedErrors = append(receivedErrors, err)
					}
				case <-ctx.Done():
					if ctx.Err() == context.DeadlineExceeded {
						t.Error("Timeout waiting for errors")
					} else {
						t.Errorf("Unexpected error: %v", ctx.Err())
					}
				case <-done:
					isDone = true
				default:
				}
				if isDone {
					break
				}
			}

			if !reflect.DeepEqual(worker.processJobs, tc.expectJobs) {
				t.Errorf("Expected jobs: %v, received: %v", tc.expectJobs, worker.processJobs)
			}
			if !slices.Equal(receivedErrors, tc.expectedErrs) {
				t.Errorf("unexpected errors. expected %v, received %v", tc.expectedErrs, receivedErrors)
			}
		})
	}
}

var (
	errGetRuns    = errors.New("Mock RunsGetter error")
	errProcesRuns = errors.New("Mock RunsProcessor error")
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
		return errProcesRuns
	}

	return nil
}

type processWorkflowTest struct {
	name               string
	job                workflowArguments
	mockGetRunsCfg     mockGetRunsConfig
	mockProcessRunsCfg mockProcessRunsConfig
	expectedErr        error
}

func TestProcessWorkflow(t *testing.T) {
	testCases := []processWorkflowTest{
		{
			name: "Successful Workflow",
			job: workflowArguments{
				from:    time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				browser: "Chrome",
				channel: "stable",
			},
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
			job: workflowArguments{
				from:    time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				browser: "Chrome",
				channel: "stable",
			},
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
			job: workflowArguments{
				from:    time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				browser: "Chrome",
				channel: "stable",
			},
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
			expectedErr: errProcesRuns,
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
			worker := NewWptRunsWorker(&runsGetter, &runsProcessor)

			err := worker.processWorkflow(context.Background(), tc.job)
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("Expected error: %v, Got: %v", tc.expectedErr, err)
			}
		})
	}
}
