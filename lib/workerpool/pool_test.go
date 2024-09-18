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
	"errors"
	"fmt"
	"testing"
)

type mockJobProcessor[TJob any] struct {
	processFunc func(ctx context.Context, job TJob) error
}

func (m *mockJobProcessor[TJob]) Process(ctx context.Context, job TJob) error {
	if m.processFunc != nil {
		return m.processFunc(ctx, job)
	}

	return nil
}

type startPoolTest struct {
	name        string
	numWorkers  int
	jobs        []int
	processFunc func(ctx context.Context, job int) error
	want        []error
}

func TestWorkflowStart(t *testing.T) {
	testCases := []startPoolTest{
		{
			name:       "no workers",
			numWorkers: 0,
			jobs:       []int{1, 2, 3},
			processFunc: func(_ context.Context, _ int) error {
				return nil
			},
			want: nil, // Expect no errors since no workers are started
		},
		{
			name:       "success",
			numWorkers: 2,
			jobs:       []int{1, 2, 3},
			processFunc: func(_ context.Context, _ int) error {
				return nil
			},
			want: nil,
		},
		{
			name:       "single error",
			numWorkers: 2,
			jobs:       []int{1, 2, 3},
			processFunc: func(_ context.Context, job int) error {
				if job == 2 {
					return errors.New("error processing job 2")
				}

				return nil
			},
			want: []error{errors.New("error processing job 2")},
		},
		{
			name:       "multiple errors",
			numWorkers: 3,
			jobs:       []int{1, 2, 3, 4, 5},
			processFunc: func(_ context.Context, job int) error {
				if job == 2 || job == 4 {
					return fmt.Errorf("error processing job %d", job)
				}

				return nil
			},
			want: []error{
				errors.New("error processing job 2"),
				errors.New("error processing job 4"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := Pool[int]{}
			processor := &mockJobProcessor[int]{processFunc: tc.processFunc}
			got := p.Start(context.Background(), tc.numWorkers, processor, tc.jobs)

			// Compare errors (order doesn't matter)
			if len(got) != len(tc.want) {
				t.Errorf("Start() returned %d errors, want %d errors", len(got), len(tc.want))
			} else {
				for _, wantErr := range tc.want {
					found := false
					for _, gotErr := range got {
						if wantErr.Error() == gotErr.Error() {
							found = true

							break
						}
					}
					if !found {
						t.Errorf("Start() returned unexpected error: %v", wantErr)
					}
				}
			}
		})
	}
}
