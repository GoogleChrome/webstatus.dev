package gcpspanner

import (
	"fmt"

	"cloud.google.com/go/spanner"
)

const chromiumHistogramEnumsTable = "ChromiumHistogramEnums"

type chromiumHistogramMapper struct{}

func (m chromiumHistogramMapper) Table() string {
	return chromiumHistogramEnumsTable
}

func (m chromiumHistogramMapper) SelectOne(key spannerChromiumHistogramKey) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID, HistogramName, BucketID, Label
	FROM %s
	WHERE HistogramName = @histogramName AND BucketID = @bucketID
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"histogramName": key.HistorgramName,
		"bucketID":      key.BucketID,
	}
	stmt.Params = parameters

	return stmt
}

type ChromiumHistogramEnum struct {
}

type spannerChromiumHistogram struct{}

type spannerChromiumHistogramKey struct {
	HistorgramName string
	BucketID       int64
}
