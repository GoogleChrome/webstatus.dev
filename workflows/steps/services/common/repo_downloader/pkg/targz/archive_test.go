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

package targz

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestIterator(t *testing.T) {
	testCases := []struct {
		name            string
		filename        string
		expectedFiles   []string
		stripComponents int
	}{
		{
			name:            "basic - one file - 0 strip components",
			filename:        "case01_basic.tar.gz",
			expectedFiles:   []string{"case01_basic/test.txt"},
			stripComponents: 0,
		},
		{
			name:            "basic - one file - 1 strip components",
			filename:        "case01_basic.tar.gz",
			expectedFiles:   []string{"test.txt"},
			stripComponents: 1,
		},
		{
			name:            "nested directories - two files - 0 strip components",
			filename:        "case02_nested.tar.gz",
			expectedFiles:   []string{"case02_nested/bar/foo/test2.txt", "case02_nested/test1.txt"},
			stripComponents: 0,
		},
		{
			name:            "nested directories - two files - 1 strip components",
			filename:        "case02_nested.tar.gz",
			expectedFiles:   []string{"bar/foo/test2.txt", "test1.txt"},
			stripComponents: 1,
		},
		{
			name:            "empty - zero files - 0 strip components",
			filename:        "case03_empty.tar.gz",
			expectedFiles:   []string{},
			stripComponents: 0,
		},
		{
			name:            "empty - zero files - 1 strip components",
			filename:        "case03_empty.tar.gz",
			expectedFiles:   []string{},
			stripComponents: 1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			compressedFilename := filepath.Join("testdata", "compressed", tc.filename)
			compressedFile, err := os.OpenFile(compressedFilename, os.O_RDONLY, 0600)
			stripComponents := tc.stripComponents
			if err != nil {
				t.Fatalf("unable to open file: %s\n", err.Error())
			}
			defer compressedFile.Close()
			iterator, err := NewTarGzArchiveIterator(compressedFile, &stripComponents)
			if err != nil {
				t.Errorf("unable to create iterator: %s\n", err.Error())

				return
			}
			filesInArchive := []string{}
			for {
				file, err := iterator.Next()
				if err != nil {
					break
				}
				filesInArchive = append(filesInArchive, file.name)
			}
			if !slices.Equal(tc.expectedFiles, filesInArchive) {
				t.Errorf("expected files: %v\nactual files in archive: %v\n", tc.expectedFiles, filesInArchive)
			}
		})
	}
}
