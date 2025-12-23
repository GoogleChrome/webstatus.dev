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

package generic

import "encoding/json"

// OptionallySet allows distinguishing between "Missing Field" (Schema Cold Start)
// and "Zero Value" (Valid Data).
type OptionallySet[T any] struct {
	Value T
	IsSet bool
}

func (o OptionallySet[T]) IsZero() bool {
	return !o.IsSet
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (o *OptionallySet[T]) UnmarshalJSON(data []byte) error {
	o.IsSet = true

	return json.Unmarshal(data, &o.Value)
}

// MarshalJSON implements the json.Marshaler interface.
func (o OptionallySet[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.Value)
}

// UnsetOpt is a helper that creates an OptionallySet with IsSet=false.
func UnsetOpt[T any]() OptionallySet[T] {
	var zeroValue T

	return OptionallySet[T]{
		Value: zeroValue,
		IsSet: false,
	}
}

// SetOpt is a helper that creates an OptionallySet with IsSet=true.
func SetOpt[T any](value T) OptionallySet[T] {
	return OptionallySet[T]{
		Value: value,
		IsSet: true,
	}
}
