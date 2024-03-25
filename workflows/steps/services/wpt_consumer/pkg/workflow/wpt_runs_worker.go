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

func NewWptRunsWorker(runsGetter RunsGetter, runsProcessor RunsProcessor) *WptRunsWorker {
	return &WptRunsWorker{
		runsGetter:    runsGetter,
		runsProcessor: runsProcessor,
	}
}

type WptRunsWorker struct {
	runsGetter    RunsGetter
	runsProcessor RunsProcessor
}

type workflowArguments struct {
	from    time.Time
	browser string
	channel string
}

type RunsProcessor interface {
	ProcessRuns(context.Context, shared.TestRuns) error
}

// RunsGetter represents the behavior to get all the runs up until the given
// date.
type RunsGetter interface {
	GetRuns(
		ctx context.Context,
		from time.Time,
		runsPerPage int,
		browserName string,
		channelName string,
	) (shared.TestRuns, error)
}

type ConsumedRunTracker interface {
	StoreConsumedRun(
		ctx context.Context,
		runID string,
		runStartTime time.Time,
		binarySHA string,
		consumeDate time.Time) error
}

type Worker interface {
	processWorkflow(ctx context.Context, job workflowArguments) error
}

type RunsWorkerManager struct {
	worker Worker
}

func NewRunsWorkerManager(in Worker) *RunsWorkerManager {
	return &RunsWorkerManager{
		worker: in,
	}
}

func (w RunsWorkerManager) Start(
	ctx context.Context,
	id int,
	wg *sync.WaitGroup,
	jobs <-chan workflowArguments,
	errChan chan<- error) {
	slog.Info("starting worker", "worker id", id)
	defer wg.Done()
	for job := range jobs {
		err := w.worker.processWorkflow(ctx, job)
		if err != nil {
			errChan <- err
		}
	}
	close(errChan)
}

func (w WptRunsWorker) processWorkflow(
	ctx context.Context,
	job workflowArguments) error {
	runs, err := w.runsGetter.GetRuns(ctx, job.from, shared.MaxCountMaxValue, job.browser, job.channel)
	if err != nil {
		return err
	}

	err = w.runsProcessor.ProcessRuns(ctx, runs)
	if err != nil {
		return err
	}

	return nil
}
