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

package event

import "errors"

// ErrInvalidEnvelope indicates the message structure is malformed.
var ErrInvalidEnvelope = errors.New("invalid envelope")

// ErrNoHandler indicates no registered handler matched the message.
var ErrNoHandler = errors.New("no handler registered")

// ErrSchemaValidation indicates the payload did not match the expected struct.
var ErrSchemaValidation = errors.New("schema validation failed")

// ErrUnprocessableEntity indicates the entity could not be processed.
var ErrUnprocessableEntity = errors.New("unprocessable entity")

// ErrTransientFailure indicates a transient failure that may succeed if retried.
var ErrTransientFailure = errors.New("transient failure, please retry")
