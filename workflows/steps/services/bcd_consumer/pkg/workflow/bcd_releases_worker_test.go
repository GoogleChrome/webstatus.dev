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
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/bcdconsumertypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/mdn__browser_compat_data"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/bcd_consumer/pkg/data"
)

var (
	errTestProcess = errors.New("test process error")
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
		return errTestProcess
	}
	m.processJobs = append(m.processJobs, job)

	return nil
}

type workerTest struct {
	name                   string
	jobs                   []JobArguments
	expectedErrs           []error // Errors expected on the error channel
	mockProcessWorkflowCfg mockProcessWorkflowConfig
	expectJobs             []JobArguments // To check if jobs were passed correctly
}

// nolint: gocognit // TODO. Refactor test to make it clearer
func TestWork(t *testing.T) {
	testCases := []workerTest{
		{
			name: "Successful Jobs",
			jobs: []JobArguments{
				NewJobArguments([]string{"fooBrowser", "barBrowser"}),
			},
			mockProcessWorkflowCfg: mockProcessWorkflowConfig{
				shouldFail: false,
			},
			expectJobs: []JobArguments{
				{
					browsers: []string{"fooBrowser", "barBrowser"},
				},
			},
			expectedErrs: nil,
		},
		{
			name: "Worker Failure",
			jobs: []JobArguments{
				NewJobArguments([]string{"fooBrowser", "barBrowser"}),
			},
			mockProcessWorkflowCfg: mockProcessWorkflowConfig{
				shouldFail: true,
			},
			expectJobs:   []JobArguments{},
			expectedErrs: []error{errTestProcess},
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
			w := BCDReleasesWorker{
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

type mockDownloadFileFromReleaseConfig struct {
	repoOwner   string
	repoName    string
	filePattern string
	fakeFile    io.ReadCloser
	err         error
}

type MockDataGetter struct {
	t                              *testing.T
	mockDownloadFileFromReleaseCfg *mockDownloadFileFromReleaseConfig
}

func (m *MockDataGetter) DownloadFileFromRelease(
	_ context.Context,
	owner, repo string,
	_ *http.Client,
	filePattern string) (io.ReadCloser, error) {
	if m.mockDownloadFileFromReleaseCfg.repoOwner != owner ||
		m.mockDownloadFileFromReleaseCfg.repoName != repo ||
		m.mockDownloadFileFromReleaseCfg.filePattern != filePattern {
		m.t.Error("unexpected args to DownloadFileFromRelease")
	}

	return m.mockDownloadFileFromReleaseCfg.fakeFile, m.mockDownloadFileFromReleaseCfg.err
}

type mockParseConfig struct {
	expectedFileContents string
	ret                  *data.BCDData
	err                  error
}

type MockDataParser struct {
	t            *testing.T
	mockParseCfg *mockParseConfig
}

func (m *MockDataParser) Parse(in io.ReadCloser) (*data.BCDData, error) {
	defer in.Close()
	fileContents, err := io.ReadAll(in)
	if err != nil {
		m.t.Errorf("unable to read file")
	}
	if m.mockParseCfg.expectedFileContents != string(fileContents) {
		m.t.Error("unexpected file contents")
	}

	return m.mockParseCfg.ret, m.mockParseCfg.err
}

type mockFilterDataConfig struct {
	expectedData    *data.BCDData
	expectedFilters []string
	retReleases     []bcdconsumertypes.BrowserRelease
	err             error
}

type MockDataFilter struct {
	t                 *testing.T
	mockFilterDataCfg *mockFilterDataConfig
}

func (m *MockDataFilter) FilterData(in *data.BCDData, filters []string) ([]bcdconsumertypes.BrowserRelease, error) {
	if !reflect.DeepEqual(in, m.mockFilterDataCfg.expectedData) ||
		!slices.Equal(filters, m.mockFilterDataCfg.expectedFilters) {
		m.t.Error("unexpected args to FilterData")
	}

	return m.mockFilterDataCfg.retReleases, m.mockFilterDataCfg.err
}

type mockInsertBrowserReleasesConfig struct {
	expectedReleases []bcdconsumertypes.BrowserRelease
	err              error
}

type MockDataStorer struct {
	mockInsertBrowserReleasesCfg *mockInsertBrowserReleasesConfig
	t                            *testing.T
}

func (m *MockDataStorer) InsertBrowserReleases(_ context.Context, releases []bcdconsumertypes.BrowserRelease) error {
	if !reflect.DeepEqual(m.mockInsertBrowserReleasesCfg.expectedReleases, releases) {
		m.t.Error("unexpected args to InsertBrowserReleases")
	}

	return m.mockInsertBrowserReleasesCfg.err
}

type processWorkflowTest struct {
	name                           string
	job                            JobArguments
	mockDownloadFileFromReleaseCfg *mockDownloadFileFromReleaseConfig
	mockParseCfg                   *mockParseConfig
	mockFilterDataCfg              *mockFilterDataConfig
	mockInsertBrowserReleasesCfg   *mockInsertBrowserReleasesConfig
	expectedErr                    error
}

const repoOwner = "owner"
const repoName = "repo"
const filePattern = "data.json"

var (
	errTestGetter = errors.New("test getter error")
	errTestParse  = errors.New("test parse error")
	errTestFilter = errors.New("test filter error")
	errTestInsert = errors.New("test insert error")
)

func valuePtr[T any](in T) *T { return &in }

func getSampleBCDData() *data.BCDData {
	return &data.BCDData{
		// nolint: exhaustruct // WONTFIX external struct
		BrowserData: mdn__browser_compat_data.BrowserData{
			Browsers: map[string]mdn__browser_compat_data.BrowserStatement{
				"fooBrowser": {
					Releases: map[string]mdn__browser_compat_data.ReleaseStatement{
						"0": {
							ReleaseDate: valuePtr("2000-01-01"),
						},
					},
				},
				"barBrowser": {
					Releases: map[string]mdn__browser_compat_data.ReleaseStatement{
						"0": {
							ReleaseDate: valuePtr("2000-01-02"),
						},
					},
				},
			},
		},
	}
}

func getSampleReleases() []bcdconsumertypes.BrowserRelease {
	return []bcdconsumertypes.BrowserRelease{
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "0",
			ReleaseDate:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			BrowserName:    "barBrowser",
			BrowserVersion: "0",
			ReleaseDate:    time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
	}
}

func TestProcess(t *testing.T) {
	testCases := []processWorkflowTest{
		{
			name: "successful process",
			job: JobArguments{
				browsers: []string{"fooBrowser", "barBrowser"},
			},
			mockDownloadFileFromReleaseCfg: &mockDownloadFileFromReleaseConfig{
				repoOwner:   repoOwner,
				repoName:    repoName,
				filePattern: filePattern,
				fakeFile:    io.NopCloser(strings.NewReader("success")),
				err:         nil,
			},
			mockParseCfg: &mockParseConfig{
				expectedFileContents: "success",
				ret:                  getSampleBCDData(),
				err:                  nil,
			},
			mockFilterDataCfg: &mockFilterDataConfig{
				expectedData:    getSampleBCDData(),
				expectedFilters: []string{"fooBrowser", "barBrowser"},
				retReleases:     getSampleReleases(),
				err:             nil,
			},
			mockInsertBrowserReleasesCfg: &mockInsertBrowserReleasesConfig{
				expectedReleases: getSampleReleases(),
				err:              nil,
			},
			expectedErr: nil,
		},
		{
			name: "failed to get data",
			job: JobArguments{
				browsers: []string{"fooBrowser", "barBrowser"},
			},
			mockDownloadFileFromReleaseCfg: &mockDownloadFileFromReleaseConfig{
				repoOwner:   repoOwner,
				repoName:    repoName,
				filePattern: filePattern,
				fakeFile:    io.NopCloser(strings.NewReader("success")),
				err:         errTestGetter,
			},
			mockParseCfg:                 nil,
			mockFilterDataCfg:            nil,
			mockInsertBrowserReleasesCfg: nil,
			expectedErr:                  errTestGetter,
		},
		{
			name: "failed to parse data",
			job: JobArguments{
				browsers: []string{"fooBrowser", "barBrowser"},
			},
			mockDownloadFileFromReleaseCfg: &mockDownloadFileFromReleaseConfig{
				repoOwner:   repoOwner,
				repoName:    repoName,
				filePattern: filePattern,
				fakeFile:    io.NopCloser(strings.NewReader("success")),
				err:         nil,
			},
			mockParseCfg: &mockParseConfig{
				expectedFileContents: "success",
				ret:                  getSampleBCDData(),
				err:                  errTestParse,
			},
			mockFilterDataCfg:            nil,
			mockInsertBrowserReleasesCfg: nil,
			expectedErr:                  errTestParse,
		},
		{
			name: "failed to filter data",
			job: JobArguments{
				browsers: []string{"fooBrowser", "barBrowser"},
			},
			mockDownloadFileFromReleaseCfg: &mockDownloadFileFromReleaseConfig{
				repoOwner:   repoOwner,
				repoName:    repoName,
				filePattern: filePattern,
				fakeFile:    io.NopCloser(strings.NewReader("success")),
				err:         nil,
			},
			mockParseCfg: &mockParseConfig{
				expectedFileContents: "success",
				ret:                  getSampleBCDData(),
				err:                  nil,
			},
			mockFilterDataCfg: &mockFilterDataConfig{
				expectedData:    getSampleBCDData(),
				expectedFilters: []string{"fooBrowser", "barBrowser"},
				retReleases:     getSampleReleases(),
				err:             errTestFilter,
			},
			mockInsertBrowserReleasesCfg: nil,
			expectedErr:                  errTestFilter,
		},
		{
			name: "failed to store data",
			job: JobArguments{
				browsers: []string{"fooBrowser", "barBrowser"},
			},
			mockDownloadFileFromReleaseCfg: &mockDownloadFileFromReleaseConfig{
				repoOwner:   repoOwner,
				repoName:    repoName,
				filePattern: filePattern,
				fakeFile:    io.NopCloser(strings.NewReader("success")),
				err:         nil,
			},
			mockParseCfg: &mockParseConfig{
				expectedFileContents: "success",
				ret:                  getSampleBCDData(),
				err:                  nil,
			},
			mockFilterDataCfg: &mockFilterDataConfig{
				expectedData:    getSampleBCDData(),
				expectedFilters: []string{"fooBrowser", "barBrowser"},
				retReleases:     getSampleReleases(),
				err:             nil,
			},
			mockInsertBrowserReleasesCfg: &mockInsertBrowserReleasesConfig{
				expectedReleases: getSampleReleases(),
				err:              errTestInsert,
			},
			expectedErr: errTestInsert,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			worker := NewBCDReleasesWorker(
				&MockDataGetter{
					t:                              t,
					mockDownloadFileFromReleaseCfg: tc.mockDownloadFileFromReleaseCfg,
				},
				&MockDataParser{
					t:            t,
					mockParseCfg: tc.mockParseCfg,
				},
				&MockDataFilter{
					t:                 t,
					mockFilterDataCfg: tc.mockFilterDataCfg,
				},
				&MockDataStorer{
					t:                            t,
					mockInsertBrowserReleasesCfg: tc.mockInsertBrowserReleasesCfg,
				},
				repoOwner,
				repoName,
				filePattern,
			)

			err := worker.jobProcessor.Process(context.Background(), tc.job)
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("Expected error: %v, Got: %v", tc.expectedErr, err)
			}
		})
	}
}
