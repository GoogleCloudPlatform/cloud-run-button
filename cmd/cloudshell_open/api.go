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
	"os/exec"
	"strings"
)

func enableAPIs(project string, apis []string) error {
	enabled, err := enabledAPIs(project)
	if err != nil {
		return err
	}

	var needAPIs []string
	for _, api := range apis {
		need := true
		for _, v := range enabled {
			if v == api {
				need = false
				break
			}
		}
		if need {
			needAPIs = append(needAPIs, api)
		}
	}
	if len(needAPIs) == 0 {
		return nil
	}

	cmd := exec.Command("gcloud", append([]string{"services", "enable", "--project", project, "-q"}, needAPIs...)...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable apis: %s", string(b))
	}
	return nil
}

func enabledAPIs(project string) ([]string, error) {
	cmd := exec.Command("gcloud", "services", "list", "--project", project, "--format", "value(config.name)")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled services on project %q. output: %s", project, string(b))
	}
	return strings.Split(strings.TrimSpace(string(b)), "\n"), nil
}
