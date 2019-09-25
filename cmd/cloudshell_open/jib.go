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
	"os"
	"os/exec"
	"path/filepath"
)

func jibMavenBuild(dir string, image string) error {
	cmd := exec.Command("mvn", "--file="+filepath.Join(dir, "pom.xml"),
		"--fail-fast", "--batch-mode", "--no-transfer-progress", "-Dmaven.test.skip=true",
		"package", "jib:build", "-Dimage="+image, "-Djib.to.auth.credHelper=gcloud")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Jib Maven build failed: %v, output:\n%s", err, string(b))
	}
	return nil
}

func jibMavenConfigured(dir string) (bool, error) {
	if _, err := os.Stat(filepath.Join(dir, "pom.xml")); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check for pom.xml in the repo: %v", err)
	}

	cmd := exec.Command("mvn", "--file="+filepath.Join(dir, "pom.xml"),
		"--fail-fast", "--batch-mode", "--no-transfer-progress",
		"jib:_skaffold-fail-if-jib-out-of-date", "-Djib.requiredVersion=1.4.0")
	if _, err := cmd.CombinedOutput(); err != nil {
		return false, nil
	}

	return true, nil
}
