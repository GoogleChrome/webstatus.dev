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
	"testing"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

type mockProcessRunCall struct {
	run shared.TestRun
	err error
}
type mockProcessRunConfig struct {
	expectedCalls []mockProcessRunCall
}

func NewMockRunProcessor(t *testing.T, cfg mockProcessRunConfig) *MockRunProcessor {
	return &MockRunProcessor{
		processRunCount:   0,
		mockProcessRunCfg: cfg,
		t:                 t,
	}
}

type MockRunProcessor struct {
	processRunCount   int
	mockProcessRunCfg mockProcessRunConfig
	t                 *testing.T
}

func (m *MockRunProcessor) ProcessRun(_ context.Context, run shared.TestRun) error {
	if m.processRunCount >= len(m.mockProcessRunCfg.expectedCalls) {
		m.t.Fatal("test not configured with expected number of calls")
	}
	idx := m.processRunCount
	if !reflect.DeepEqual(m.mockProcessRunCfg.expectedCalls[idx].run, run) {
		m.t.Error("unexpected run")
	}
	m.processRunCount++

	return m.mockProcessRunCfg.expectedCalls[idx].err
}

var errTestProcessRun = errors.New("process run error")

func TestProcessRuns(t *testing.T) {
	testCases := []struct {
		name             string
		mockProcesRunCfg mockProcessRunConfig
		inputRuns        shared.TestRuns
		expectedError    error
	}{
		{
			name: "success",
			mockProcesRunCfg: mockProcessRunConfig{
				expectedCalls: []mockProcessRunCall{
					{
						// nolint: exhaustruct // WONTFIX. struct from external package.
						run: shared.TestRun{
							ID:         0,
							ResultsURL: "http://example.com/0",
						},
						err: nil,
					},
				},
			},
			// nolint: exhaustruct // WONTFIX. struct from external package.
			inputRuns: []shared.TestRun{
				{
					ID:         0,
					ResultsURL: "http://example.com/0",
				},
			},
			expectedError: nil,
		},
		{
			name: "error",
			mockProcesRunCfg: mockProcessRunConfig{
				expectedCalls: []mockProcessRunCall{
					{
						// nolint: exhaustruct // WONTFIX. struct from external package.
						run: shared.TestRun{
							ID:         0,
							ResultsURL: "http://example.com/0",
						},
						err: errTestProcessRun,
					},
				},
			},
			// nolint: exhaustruct // WONTFIX. struct from external package.
			inputRuns: []shared.TestRun{
				{
					ID:         0,
					ResultsURL: "http://example.com/0",
				},
			},
			expectedError: errTestProcessRun,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewMockRunProcessor(t, tc.mockProcesRunCfg)
			p := NewWPTRunsProcessor(m)
			err := p.ProcessRuns(context.Background(), tc.inputRuns)
			if !errors.Is(err, tc.expectedError) {
				t.Error("unexpected error")
			}
			if m.processRunCount != len(tc.mockProcesRunCfg.expectedCalls) {
				t.Errorf("missing calls. got %d expected %d",
					m.processRunCount, len(tc.mockProcesRunCfg.expectedCalls))
			}
		})
	}
}
