// Copyright 2025 Google LLC
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

package gcppubsubadapters

import (
	"context"
	"sync"
)

// RunGroup runs multiple blocking functions concurrently.
// - It returns the first error encountered.
// - If one function fails, it cancels the context for the others.
// - It waits for all functions to exit before returning.
func RunGroup(ctx context.Context, fns ...func(ctx context.Context) error) error {
	// 1. Create a derived context so we can signal cancellation to all siblings
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	errChan := make(chan error, len(fns))

	for _, fn := range fns {
		wg.Add(1)
		// Capture fn in the loop scope
		go func(f func(context.Context) error) {
			defer wg.Done()

			// Pass the cancellable context to the function
			if err := f(ctx); err != nil {
				// Try to push the error; if channel is full, we already have an error
				select {
				case errChan <- err:
					// Signal other routines to stop
					cancel()
				default:
				}
			}
		}(fn)
	}

	wg.Wait()
	close(errChan)

	// Return the first error (if any)
	return <-errChan
}
