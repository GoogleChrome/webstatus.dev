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
	"log/slog"
	"sync"
	"time"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

type Entrypoint struct {
	Starter    WorkerStarter
	NumWorkers int
}

type WorkerStarter interface {
	Start(ctx context.Context, id int, wg *sync.WaitGroup, jobs <-chan workflowArguments, errChan chan<- error)
}

func (w Entrypoint) Start(ctx context.Context, from time.Time) []error {
	browsers := shared.GetDefaultBrowserNames()
	channels := []string{shared.StableLabel, shared.ExperimentalLabel}
	wg := sync.WaitGroup{}
	numberOfJobs := len(browsers) * len(channels)
	jobsChan := make(chan workflowArguments, numberOfJobs)
	errChan := make(chan error, numberOfJobs)

	// Start the workers
	wg.Add(w.NumWorkers)
	for i := 0; i < w.NumWorkers; i++ {
		go w.Starter.Start(ctx, i, &wg, jobsChan, errChan)
	}
	for _, browser := range browsers {
		for _, channel := range channels {
			jobsChan <- workflowArguments{
				from:    from,
				browser: browser,
				channel: channel,
			}
		}
	}
	close(jobsChan)
	doneChan := make(chan struct{})
	// Wait for workers and handle errors
	go func() {
		wg.Wait()
		slog.Info("finished waiting")
		doneChan <- struct{}{}
	}()

	var allErrors []error

	for {
		select {
		case err, ok := <-errChan:
			if !ok {
				// Handle collected errors
				return allErrors

			}
			allErrors = append(allErrors, err)
		case <-doneChan:
			// Channel closed, proceed
			slog.Info("Finished processing", "error count", len(allErrors))

			return allErrors
		}

	}
}
