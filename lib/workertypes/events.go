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

package workertypes

// The structs in this file implement the event.Event interface from github.com/GoogleChrome/webstatus.dev/lib/event
// New fields must be added in a non-breaking way or in a new struct.

// NotificationEventCreatedV1 lets consumers know that a particular notification has been created.
type NotificationEventCreatedV1 struct {
	ID string `json:"id"`
}

func (e NotificationEventCreatedV1) Kind() string { return "NotificationEventCreated" }

func (e NotificationEventCreatedV1) APIVersion() string { return "v1" }
