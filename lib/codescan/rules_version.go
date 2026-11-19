// Copyright 2026 Google LLC
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

package codescan

// RulesVersion identifies the rules that turn a repository into code subscriptions: directive
// parsing, file selection, and scan limits. The scanner records it on every scan log row.
//
// vcs-sync does not rescan a repository whose latest scan is still valid, and a scan recorded with
// an older RulesVersion is not valid. Bump this whenever a change could produce different results
// for a repository that has not changed. Every repository is then rescanned once.
const RulesVersion int64 = 1
