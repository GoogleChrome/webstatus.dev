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

package data

import (
	"os"
	"path"
	"testing"
)

func TestParse(t *testing.T) {
	file, err := os.Open(path.Join("testdata", "data.json"))
	if err != nil {
		t.Fatalf("unable to read file err %s", err.Error())
	}
	result, err := Parser{}.Parse(file)
	if err != nil {
		t.Errorf("unable to parse file err %s", err.Error())
	}
	if len(result) == 0 {
		t.Error("unexpected empty map")
	}
}
