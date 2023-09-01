package filefilter

import (
	"strings"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/workflows/steps/common/repo_downloader"
)

type Engine struct {
	filters *[]repo_downloader.FileFilter
}

func NewEngine(filters *[]repo_downloader.FileFilter) *Engine {
	return &Engine{filters: filters}
}

func (e *Engine) Applies(filename string) bool {
	// If there are no filters, it applies
	if e.filters == nil || len(*e.filters) == 0 {
		return true
	}

	for _, filter := range *e.filters {
		if filter.Prefix != nil &&
			filter.Suffix != nil &&
			strings.HasPrefix(filename, *filter.Prefix) &&
			strings.HasSuffix(filename, *filter.Suffix) {
			return true
		}
		if filter.Prefix != nil && strings.HasPrefix(filename, *filter.Prefix) && filter.Suffix == nil {
			return true
		}
		if filter.Suffix != nil && strings.HasSuffix(filename, *filter.Suffix) && filter.Prefix == nil {
			return true
		}
	}
	return false
}
