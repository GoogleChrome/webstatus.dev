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

package httpserver

import (
	"context"
	"net/http"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// ListMissingOneImplementationFeatures implements backend.StrictServerInterface.
// nolint: ireturn // Signature generated from openapi
func (s *Server) ListMissingOneImplementationFeatures(
	_ context.Context,
	_ backend.ListMissingOneImplementationFeaturesRequestObject) (
	backend.ListMissingOneImplementationFeaturesResponseObject, error) {

	return backend.ListMissingOneImplementationFeatures400JSONResponse{
		Code:    http.StatusBadRequest,
		Message: "TODO",
	}, nil
}
