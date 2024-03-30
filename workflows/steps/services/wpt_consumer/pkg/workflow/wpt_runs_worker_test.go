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

type MockJobProcessor struct {
	processJobs            []JobArguments
	mockProcessWorkflowCfg mockProcessWorkflowConfig
}

type mockProcessWorkflowConfig struct {
	shouldFail bool
}

func (m *MockJobProcessor) Process(_ context.Context, job JobArguments) error {
	if m.mockProcessWorkflowCfg.shouldFail {
		return errMockWriterFail
	}
	m.processJobs = append(m.processJobs, job)

	return nil
}

var errMockWriterFail = errors.New("mock writer test erro")

type startWorkerTest struct {
	name                   string
	jobs                   []JobArguments
	expectedErrs           []error // Errors expected on the error channel
	mockProcessWorkflowCfg mockProcessWorkflowConfig
	expectJobs             []JobArguments // To check if jobs were passed correctly
}

// nolint: gocognit // TODO. Refactor test to make it clearer
func TestWork(t *testing.T) {
	testCases := []startWorkerTest{
		{
			name: "Successful Jobs",
			jobs: []JobArguments{
				NewJobArguments(
					time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
					"Chrome",
					"stable",
					25,
				),
				NewJobArguments(
					time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC),
					"Firefox",
					"experimental",
					25,
				),
			},
			mockProcessWorkflowCfg: mockProcessWorkflowConfig{
				shouldFail: false,
			},
			expectJobs: []JobArguments{
				{
					from:     time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
					browser:  "Chrome",
					channel:  "stable",
					pageSize: 25,
				},
				{
					from:     time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC),
					browser:  "Firefox",
					channel:  "experimental",
					pageSize: 25,
				},
			},
			expectedErrs: nil,
		},
		{
			name: "Worker Failure",
			jobs: []JobArguments{
				NewJobArguments(time.Now(), "Chrome", "stable", 25),
			},
			mockProcessWorkflowCfg: mockProcessWorkflowConfig{
				shouldFail: true,
			},
			expectedErrs: []error{errMockWriterFail},
			expectJobs:   []JobArguments{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testJobs := make(chan JobArguments)
			testErrChan := make(chan error)
			testWg := &sync.WaitGroup{}

			ctx, cancelFunc := context.WithCancel(context.Background()) // For potential cancellation tests
			defer cancelFunc()                                          // Ensure cleanup

			testWg.Add(1)

			jobProcessor := &MockJobProcessor{
				processJobs:            []JobArguments{},
				mockProcessWorkflowCfg: tc.mockProcessWorkflowCfg,
			}
			w := WptRunsWorker{
				jobProcessor: jobProcessor,
			}

			go w.Work(ctx, 1, testWg, testJobs, testErrChan)

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

			if !reflect.DeepEqual(jobProcessor.processJobs, tc.expectJobs) {
				t.Errorf("Expected jobs: %v, received: %v", tc.expectJobs, jobProcessor.processJobs)
			}
			if !slices.Equal(receivedErrors, tc.expectedErrs) {
				t.Errorf("unexpected errors. expected %v, received %v", tc.expectedErrs, receivedErrors)
			}
		})
	}
}

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
			job: JobArguments{
				from:     time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				browser:  "Chrome",
				channel:  "stable",
				pageSize: 25,
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
			job: JobArguments{
				from:     time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				browser:  "Chrome",
				channel:  "stable",
				pageSize: 25,
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
			job: JobArguments{
				from:     time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				browser:  "Chrome",
				channel:  "stable",
				pageSize: 25,
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
			worker := NewWptRunsWorker(&runsGetter, &runsProcessor)

			err := worker.jobProcessor.Process(context.Background(), tc.job)
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("Expected error: %v, Got: %v", tc.expectedErr, err)
			}
		})
	}
}
