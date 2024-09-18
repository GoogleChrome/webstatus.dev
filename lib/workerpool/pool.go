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

// JobProcessor defines the contract for processing a single job within a workflow.
type JobProcessor[TJob any] interface {
	Process(
		ctx context.Context,
		job TJob) error
}

func (p Pool[TJob]) StartWorker(
	ctx context.Context,
	processor JobProcessor[TJob],
	id int,
	wg *sync.WaitGroup,
	jobs <-chan TJob,
	errChan chan<- error) {
	slog.InfoContext(ctx, "starting worker", "worker id", id)
	defer wg.Done()

	// Processes jobs received on the 'jobs' channel
	for job := range jobs {
		err := processor.Process(ctx, job)
		if err != nil {
			errChan <- err
		}
	}
	// Do not close the shared error channel here.
	// It will prevent others from returning their errors.
}

func (p Pool[TJob]) Start(ctx context.Context, numWorkers int, processor JobProcessor[TJob], jobs []TJob) []error {
	wg := sync.WaitGroup{}
	errChan := make(chan error)
	jobsChan := make(chan TJob)

	// Start the workers
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go p.StartWorker(ctx, processor, i, &wg, jobsChan, errChan)
	}
	// Send the job
	go func() {
		for _, job := range jobs {
			slog.InfoContext(ctx, "sending job to workers in pool", "job", job)
			jobsChan <- job
		}
		// Close the job channel now that we are done.
		close(jobsChan)
	}()

	doneChan := make(chan struct{})
	// Wait for workers and handle errors
	go func() {
		wg.Wait()
		slog.InfoContext(ctx, "finished waiting")
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
			slog.InfoContext(ctx, "Finished processing", "error count", len(allErrors))

			return allErrors
		}

	}
}
