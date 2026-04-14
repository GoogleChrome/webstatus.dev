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

package httpserver

type fieldValidationErrors struct {
	fieldErrorMap map[string]string
}

func (f *fieldValidationErrors) addFieldError(field string, err error) {
	if f.fieldErrorMap == nil {
		f.fieldErrorMap = make(map[string]string)
	}
	f.fieldErrorMap[field] = err.Error()
}

func (f fieldValidationErrors) hasErrors() bool {
	return len(f.fieldErrorMap) > 0
}
