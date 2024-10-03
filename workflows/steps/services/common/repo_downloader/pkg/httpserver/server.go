// Copyright 2023 Google LLC
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

package httpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/workflows/steps/common/repo_downloader"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/common/repo_downloader/pkg/filefilter"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/common/repo_downloader/pkg/gh"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/common/repo_downloader/pkg/targz"
)

type Storer interface {
	Store(ctx context.Context, data io.Reader, filename string) error
	GetLocation() string
}

type Downloader interface {
	Download(ctx context.Context, repoOwner, repoName string, ref *string) (io.ReadCloser, string, error)
}

type Server struct {
	downloader Downloader
	storer     Storer
}

// nolint:ireturn // Expected ireturn for openapi generation.
func (s *Server) PostV1GithubComOwnerName(ctx context.Context,
	request repo_downloader.PostV1GithubComOwnerNameRequestObject) (
	repo_downloader.PostV1GithubComOwnerNameResponseObject, error) {
	var archive io.ReadCloser
	var branch string
	var err error

	// Step 1. Download the archive
	switch request.Body.Archive.Type {
	case repo_downloader.TAR:
		archive, branch, err = s.downloader.Download(ctx, request.Owner, request.Name, nil)
	default:
		err = fmt.Errorf("unsupported archive type. %s", request.Body.Archive.Type)
	}
	if err != nil {
		// TODO: separate the different errors
		slog.Error("unable to download archive.", "error", err.Error())

		return repo_downloader.PostV1GithubComOwnerName400JSONResponse{
			Code:    400,
			Message: "unable to download archive",
		}, nil
	}
	defer archive.Close()

	repoPathPrefix := fmt.Sprintf("%s/%s/%s", request.Owner, request.Name, branch)
	// Iterator is responsible for closing the resp.Body
	archiveReader, err := targz.NewTarGzArchiveIterator(archive, request.Body.Archive.TarStripComponents)
	if err != nil {
		slog.Error("unable to extract archive. %s", "error", err.Error())

		return repo_downloader.PostV1GithubComOwnerName400JSONResponse{
			Code:    400,
			Message: "unable to extract archive",
		}, nil
	}
	defer archiveReader.Close()

	// Step 2. Only extract files that the user want
	// Build file filter engine
	fileFilterEngine := filefilter.NewEngine(request.Body.FileFilters)
	filteredFileNames := []string{}
	for {
		file, err := archiveReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if !fileFilterEngine.Applies(file.GetName()) {
			continue
		}
		filteredFileNames = append(filteredFileNames, file.GetName())
		// Step 3. Store the files
		err = s.storer.Store(
			ctx,
			file.GetData(),
			fmt.Sprintf("%s/%s", repoPathPrefix, file.GetName()))
		if err != nil {
			slog.Error("unable to store file. %s", "error", err.Error())

			return repo_downloader.PostV1GithubComOwnerName500JSONResponse{
				Code:    500,
				Message: "unable to store file",
			}, nil
		}
	}

	return repo_downloader.PostV1GithubComOwnerName200JSONResponse{
		Destination: repo_downloader.UploadDestinationReport{
			Gcs: &repo_downloader.GCSUploadReport{
				Filenames:  &filteredFileNames,
				RepoPrefix: repoPathPrefix,
				Bucket:     s.storer.GetLocation(),
			},
		},
	}, nil
}

func NewHTTPServer(port string, downloader *gh.Downloader, storer Storer) (*http.Server, error) {
	_, err := repo_downloader.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("error loading swagger spec. %w", err)
	}

	// Create an instance of our handler which satisfies the generated interface
	srv := &Server{
		downloader: downloader,
		storer:     storer,
	}

	srvStrictHandler := repo_downloader.NewStrictHandler(srv, nil)

	// Use standard library router
	r := http.NewServeMux()

	// Use our validation middleware to check all requests against the
	// repo_downloader schema.
	// r.Use(middleware.OapiRequestValidatorWithOptions(swagger, &middleware.Options{
	// 	SilenceServersWarning: true,
	// }))

	// We now register our repo downloader above as the handler for the interface
	repo_downloader.HandlerFromMux(srvStrictHandler, r)

	// nolint:exhaustruct // No need to populate 3rd party struct
	return &http.Server{
		Handler:           r,
		Addr:              net.JoinHostPort("0.0.0.0", port),
		ReadHeaderTimeout: 30 * time.Second,
	}, nil
}
