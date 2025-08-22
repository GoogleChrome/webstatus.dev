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
	"io"
	"net/http"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/bcdconsumertypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gh"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/bcd_consumer/pkg/data"
)

func NewJobArguments(browsers []string) JobArguments {
	return JobArguments{
		browsers: browsers,
	}
}

type JobArguments struct {
	browsers []string // List of browsers that we will search for and store their respective release information.
}

// JobProcessor defines the contract for processing a single job within the BCD Releases workflow.
type JobProcessor interface {
	Process(
		ctx context.Context,
		job JobArguments) error
}

type DataGetter interface {
	DownloadFileFromRelease(
		ctx context.Context,
		owner, repo string,
		httpClient *http.Client,
		filePattern string) (*gh.ReleaseFile, error)
}

// DataParser describes the behavior to read raw bytes into the expected BCDData struct.
type DataParser interface {
	Parse(in io.ReadCloser) (*data.BCDData, error)
}

// DataFilter describes the behavior to take full BCDData and only filter for the applicable browser releases.
type DataFilter interface {
	FilterData(*data.BCDData, []string) ([]bcdconsumertypes.BrowserRelease, error)
}

// DataStorer describes the behavior to store the release information.
type DataStorer interface {
	InsertBrowserReleases(ctx context.Context, releases []bcdconsumertypes.BrowserRelease) error
}

func NewBCDJobProcessor(dataGetter DataGetter,
	dataParser DataParser,
	dataFilter DataFilter,
	dataStorer DataStorer,
	repoOwner string,
	repoName string,
	releaseAssetFilename string) BCDJobProcessor {
	return BCDJobProcessor{
		dataGetter:           dataGetter,
		dataParser:           dataParser,
		dataFilter:           dataFilter,
		dataStorer:           dataStorer,
		repoOwner:            repoOwner,
		repoName:             repoName,
		releaseAssetFilename: releaseAssetFilename,
	}
}

type BCDJobProcessor struct {
	dataGetter           DataGetter // Dependency for fetching data from the BCD repo.
	dataParser           DataParser // Dependency for parsing BCD Data.
	dataFilter           DataFilter // Dependency for filtering BCD Data into the anticipated data structure
	dataStorer           DataStorer // Dependency for storing the BCD Data.
	repoOwner            string
	repoName             string
	releaseAssetFilename string
}

func (p BCDJobProcessor) Process(
	ctx context.Context,
	job JobArguments) error {
	// Step 1. Download the file.
	file, err := p.dataGetter.DownloadFileFromRelease(
		ctx,
		p.repoOwner,
		p.repoName,
		http.DefaultClient,
		p.releaseAssetFilename)
	if err != nil {
		return err
	}

	// Step 2. Parse the file.
	data, err := p.dataParser.Parse(file.Contents)
	if err != nil {
		return err
	}

	// Step 3. Filter the data.
	filteredData, err := p.dataFilter.FilterData(data, job.browsers)
	if err != nil {
		return err
	}

	// Step 4. Insert the browser release data.
	err = p.dataStorer.InsertBrowserReleases(ctx, filteredData)
	if err != nil {
		return err
	}

	return nil
}
