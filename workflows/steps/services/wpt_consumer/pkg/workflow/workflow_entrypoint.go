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
	"log/slog"
	"sync"
	"time"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

type Entrypoint struct {
	workerStarter WorkerStarter
}

type WorkerStarter interface {
	Start(ctx context.Context, id int, wg *sync.WaitGroup, jobs <-chan workflowArguments, errChan chan<- error)
}

func (w Entrypoint) Start(ctx context.Context, numWorkers int, stopAt time.Time) error {
	browsers := shared.GetDefaultBrowserNames()
	channels := []string{shared.StableLabel, shared.ExperimentalLabel}
	wg := sync.WaitGroup{}
	numberOfJobs := len(browsers) * len(channels)
	jobsChan := make(chan workflowArguments, numberOfJobs)
	errChan := make(chan error, numberOfJobs)

	// Start the workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go w.workerStarter.Start(ctx, i, &wg, jobsChan, errChan)
	}
	wg.Add(len(browsers) * len(channels))
	for _, browser := range browsers {
		for _, channel := range channels {
			jobsChan <- workflowArguments{
				stopAt:  stopAt,
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
		close(errChan)
	}()

	var allErrors []error
	errWg := sync.WaitGroup{}
	errWg.Add(1)
	for {
		select {
		case err, ok := <-errChan:
			if !ok {
				// Handle collected errors
				if len(allErrors) > 0 {
					return errors.Join(allErrors...)
				}

				break
			}
			allErrors = append(allErrors, err)
		case <-doneChan:
			// Channel closed, proceed
			slog.Info("Finished processing", "error count", len(allErrors))

			return nil
		}

	}
}
