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
)

func packBuild(dir, image string) error {
	cmd := exec.Command("pack", "build", "--quiet", "--builder", "heroku/buildpacks", "--path", dir, image)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pack build failed: %v, output:\n%s", err, string(b))
	}
	return nil
}
