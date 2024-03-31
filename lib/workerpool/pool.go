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
	"log/slog"
	"sync"
)

// Generic pool type, can work with any job type.
type Pool[TJob any] struct{}

// A worker has a Work method handling a single job from the jobs channel and reporting any errors.
type Worker[TJob any] interface {
	Work(ctx context.Context, id int, wg *sync.WaitGroup, jobs <-chan TJob, errChan chan<- error)
}

func (p Pool[TJob]) Start(ctx context.Context, jobsChan <-chan TJob, numWorkers int, worker Worker[TJob]) []error {
	wg := sync.WaitGroup{}
	errChan := make(chan error)

	// Start the workers
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go worker.Work(ctx, i, &wg, jobsChan, errChan)
	}
	doneChan := make(chan struct{})
	// Wait for workers and handle errors
	go func() {
		wg.Wait()
		slog.Info("finished waiting")
		close(errChan) // Signal all errors collected
		doneChan <- struct{}{}
	}()

	var allErrors []error

	for {
		select {
		case err, ok := <-errChan:
			if !ok {
				// Channel closed, all errors gathered
				return allErrors

			}
			if err != nil {
				allErrors = append(allErrors, err)
			}
		case <-doneChan:
			// All workers finished
			slog.Info("Finished processing", "error count", len(allErrors))

			return allErrors
		}

	}
}
