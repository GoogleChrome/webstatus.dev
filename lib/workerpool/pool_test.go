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

package workerpool

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"sync"
	"testing"
)

type mockStartConfig struct {
	workersToFail map[int]bool
}

type MockWorker struct {
	startedWorkers []int
	calls          int
	mockStartCfg   mockStartConfig
	jobs           []string
	mu             *sync.Mutex
}

func (m *MockWorker) addCallAndID(id int) {
	m.mu.Lock()
	m.calls++
	m.startedWorkers = append(m.startedWorkers, id)
	defer m.mu.Unlock()
}

func (m *MockWorker) addJob(job string) {
	m.mu.Lock()
	m.jobs = append(m.jobs, job)
	defer m.mu.Unlock()
}

func (m *MockWorker) Work(
	_ context.Context,
	id int,
	wg *sync.WaitGroup,
	jobsChan <-chan string,
	errChan chan<- error) {
	defer wg.Done()
	m.addCallAndID(id)
	slog.Info("start running", "id", id)
	if m.mockStartCfg.workersToFail[id] { // Check if we should fail this worker
		errChan <- fmt.Errorf("Mock WorkerStarter error from worker %d", id)
	} else {
		for job := range jobsChan {
			m.addJob(job)
		}
	}

}

type startPoolTest struct {
	name               string
	numWorkers         int
	mockStartCfg       mockStartConfig
	expectedErrors     []error
	expectedStartedIDs []int // Worker IDs we expect to be started
}

func TestWorkflowStart(t *testing.T) {
	testCases := []startPoolTest{
		{
			name:       "Successful Start",
			numWorkers: 2,
			mockStartCfg: mockStartConfig{
				workersToFail: nil,
			},
			expectedStartedIDs: []int{0, 1},
			expectedErrors:     nil,
		},
		{
			name:       "Some Worker Errors",
			numWorkers: 4,
			mockStartCfg: mockStartConfig{
				workersToFail: map[int]bool{
					2: true,
				},
			},
			expectedErrors: []error{ // Expect errors reported
				fmt.Errorf("Mock WorkerStarter error from worker 2"),
			},
			expectedStartedIDs: []int{0, 1, 2, 3},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			worker := MockWorker{
				mu:             &sync.Mutex{},
				startedWorkers: []int{},
				calls:          0,
				jobs:           []string{},
				mockStartCfg:   tt.mockStartCfg,
			}
			jobs := []string{"a", "b", "c", "d", "e", "f"}
			pool := Pool[string]{}

			jobChan := make(chan string)
			go func() {
				for _, job := range jobs {
					jobChan <- job
				}
				close(jobChan)
			}()
			errs := pool.Start(
				context.Background(),
				jobChan,
				tt.numWorkers,
				&worker,
			)

			// Error Assertions
			if len(tt.expectedErrors) == 0 && errs != nil {
				t.Errorf("Unexpected error: %v", errs)
			} else if !reflect.DeepEqual(tt.expectedErrors, errs) {
				t.Errorf("Expected errors: %v, Got: %v", tt.expectedErrors, errs)
			}

			// Assertions on MockWorker (calls, started IDs, etc.)
			if worker.calls != tt.numWorkers {
				t.Errorf("Expected %d calls to WorkerStarter.Start, got %d", tt.numWorkers, worker.calls)
			}
			slices.Sort(worker.startedWorkers)
			if !reflect.DeepEqual(worker.startedWorkers, tt.expectedStartedIDs) {
				t.Errorf("Expected started worker IDs: %v, got: %v", tt.expectedStartedIDs, worker.startedWorkers)
			}

			slices.Sort(worker.jobs)
			if !reflect.DeepEqual(worker.jobs, jobs) {
				t.Errorf("Expected jobs: %v, got: %v", jobs, worker.jobs)
			}
		})
	}
}
