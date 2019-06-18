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

func dockerBuild(dir, image string) error {
	cmd := exec.Command("docker", "build", "--quiet", "--tag", image, dir)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker build failed: %v, output:\n%s", err, string(b))
	}
	return nil
}

func dockerPush(image string) error {
	cmd := exec.Command("docker", "push", image)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker push failed: %v, output:\n%s", err, string(b))
	}
	return nil
}

func dockerFileExists(dir string) (bool, error) {
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check for Dockerfile in the repo: %v", err)
	}

	return true, nil
}
