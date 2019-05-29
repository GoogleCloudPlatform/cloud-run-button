// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import "testing"

func Test_tryFixServiceName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no change", "foo", "foo"},
		{"no change - empty", "", ""},
		{"no leading letter - digit", "0abcdef", "svc-0abcdef"},
		{"no leading letter - sign", "-abcdef", "svc--abcdef"},
		{"trailing dash", "abc-def-", "abc-def"},
		{"trailing dashes", "abc-def---", "abc-def"},
		{"only dashes, trimmed and prefixed", "---", "svc"},
		{"truncate to 63 chars",
			"A123456789012345678901234567890123456789012345678901234567890123456789",
			"A12345678901234567890123456789012345678901234567890123456789012"},
		{"already at max length - no truncate",
			"A12345678901234567890123456789012345678901234567890123456789012",
			"A12345678901234567890123456789012345678901234567890123456789012",
		},
		{"leading digit + trunc",
			"0123456789012345678901234567890123456789012345678901234567890123456789",
			"svc-01234567890123456789012345678901234567890123456789012345678"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tryFixServiceName(tt.in); got != tt.want {
				t.Errorf("tryFixServiceName(%s) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
