package workflow

import (
	"context"
	"time"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

// Workflow contains all the steps for the workflow to consume wpt data.
type Workflow struct{}

// RunsGetter represents the behavior to get all the runs up until the given
// date.
type RunsGetter interface {
	GetRuns(
		ctx context.Context,
		stopAt time.Time,
		runsPerPage int,
	) (shared.TestRuns, error)
}
