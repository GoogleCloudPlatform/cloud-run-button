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
	"unicode"
)

const (
	defaultRunMemory = "512Mi"
)

func deploy(project, name, image, region string, envs []string) (string, error) {
	cmd := exec.Command("gcloud", "beta", "run", "deploy", "-q",
		name,
		"--project", project,
		"--platform", "managed",
		"--image", image,
		"--region", region,
		"--memory", defaultRunMemory,
		"--allow-unauthenticated",
		"--set-env-vars", strings.Join(envs, ","))
	if b, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to deploy to Cloud Run: %+v. output:\n%s", err, string(b))
	}
	return serviceURL(project, name, region)
}

func serviceURL(project, name, region string) (string, error) {
	cmd := exec.Command("gcloud", "beta", "run", "services", "describe", name,
		"--project", project,
		"--platform", "managed",
		"--region", region,
		"--format", "value(status.domain)")

	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("deployment to Cloud Run failed: %+v. output:\n%s", err, string(b))
	}
	return strings.TrimSpace(string(b)), nil
}

// tryFixServiceName attempts replace the service name with a better one to
// prevent deployment failures due to Cloud Run service naming constraints such
// as:
//
//   * names with a leading non-letter (e.g. digit or '-') are prefixed
//   * names over 63 characters are truncated
//   * names ending with a '-' have the suffix trimmed
func tryFixServiceName(name string) string {
	if name == "" {
		return name
	}
	if !unicode.IsLetter([]rune(name)[0]) {
		name = fmt.Sprintf("svc-%s", name)
	}
	if len(name) > 63 {
		name = name[:63]
	}
	for name[len(name)-1] == '-' {
		name = name[:len(name)-1]
	}
	return name
}
