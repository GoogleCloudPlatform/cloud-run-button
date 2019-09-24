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

import (
	"fmt"
	"reflect"
	"testing"
)

func Test_tryFixServiceName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no change", "foo", "foo"},
		{"no change - empty", "", ""},
		{"no leading letter - digit", "0abcdef", "svc-0abcdef"},
		{"no leading letter - sign", "-abcdef", "svc-abcdef"},
		{"trailing dash", "abc-def-", "abc-def"},
		{"trailing dashes", "abc-def---", "abc-def"},
		{"only dashes, trimmed and prefixed", "---", "svc"},
		{"upper to lower", "Foo", "foo"},
		{"truncate to 63 chars",
			"A123456789012345678901234567890123456789012345678901234567890123456789",
			"a12345678901234567890123456789012345678901234567890123456789012"},
		{"already at max length - no truncate",
			"A12345678901234567890123456789012345678901234567890123456789012",
			"a12345678901234567890123456789012345678901234567890123456789012",
		},
		{"leading digit + trunc",
			"0123456789012345678901234567890123456789012345678901234567890123456789",
			"svc-01234567890123456789012345678901234567890123456789012345678"},
		{"invalid chars", "+hello, world", "svc-hello-world"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tryFixServiceName(tt.in); got != tt.want {
				t.Errorf("tryFixServiceName(%s) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// this integration test depends on gcloud being auth'd as someone that has access to the crb-test project
// todo(jamesward) only run the integration test if the environment can do so
func Test_describe(t *testing.T) {
	tests := []struct {
		project string
		region string
		service string
		wantErr bool
	}{
		{"crb-test", "us-central1", "asdf1234zxcv5678", true},
		{"crb-test", "us-central1", "cloud-run-hello", false},
	}
	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			_, err := describe(tt.project, tt.service, tt.region)

			tparams := fmt.Sprintf("describe(%s, %s, %s)", tt.project, tt.region, tt.service)

			if err == nil && tt.wantErr {
				t.Error(tparams + " was expected to error but did not")
			}

			if err != nil && !tt.wantErr {
				t.Error(tparams + " produced an error: %s", err)
			}
		})
	}
}

// this integration test depends on gcloud being auth'd as someone that has access to the crb-test project
// todo(jamesward) only run the integration test if the environment can do so
func Test_envVars(t *testing.T) {
	tests := []struct {
		project string
		region string
		service string
		want map[string]struct{}
	}{
		{"crb-test", "us-central1", "cloud-run-hello", map[string]struct{}{"FOO": {}, "BAR": {}}},
	}
	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			if got, err := envVars(tt.project, tt.service, tt.region); !reflect.DeepEqual(got, tt.want) || err != nil {
				t.Errorf("envVars(%s, %s, %s) = %v, want %v", tt.project, tt.region, tt.service, got, tt.want)
			}
		})
	}
}
