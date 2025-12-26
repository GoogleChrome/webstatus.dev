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
	"errors"
	"testing"
	"time"
)

func TestRunGroup(t *testing.T) {
	tests := []struct {
		name        string
		fns         []func(context.Context) error
		expectedErr error
	}{
		{
			name: "All functions succeed",
			fns: []func(context.Context) error{
				func(_ context.Context) error { return nil },
				func(_ context.Context) error { return nil },
			},
			expectedErr: nil,
		},
		{
			name: "One function fails immediately",
			fns: []func(context.Context) error{
				func(_ context.Context) error { return errors.New("fail") },
				func(ctx context.Context) error {
					// Simulate work
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(100 * time.Millisecond):
						return nil
					}
				},
			},
			expectedErr: errors.New("fail"),
		},
		{
			name: "Multiple failures return first error",
			fns: []func(context.Context) error{
				func(_ context.Context) error { return errors.New("error 1") },
				func(_ context.Context) error {
					time.Sleep(10 * time.Millisecond) // Ensure this happens slightly later

					return errors.New("error 2")
				},
			},
			expectedErr: errors.New("error 1"),
		},
		{
			name: "Cancellation propagates",
			fns: []func(context.Context) error{
				func(_ context.Context) error {
					return errors.New("trigger cancel")
				},
				func(ctx context.Context) error {
					select {
					case <-ctx.Done():
						// Correct behavior: context was cancelled
						return nil
					case <-time.After(1 * time.Second):
						return errors.New("timeout: context was not cancelled")
					}
				},
			},
			expectedErr: errors.New("trigger cancel"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := RunGroup(context.Background(), tc.fns...)

			if tc.expectedErr == nil {
				if err != nil {
					t.Errorf("RunGroup() unexpected error: %v", err)
				}
			} else {
				if err == nil || err.Error() != tc.expectedErr.Error() {
					t.Errorf("RunGroup() error = %v, want %v", err, tc.expectedErr)
				}
			}
		})
	}
}
