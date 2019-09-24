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
)

func runScript(dir, phase string, project string, service string, region string, command string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Env = []string{
		fmt.Sprintf("PROJECT_ID=%s", project),
		fmt.Sprintf("SERVICE=%s", service),
		fmt.Sprintf("REGION=%s", region)}
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%s script failed: %v", phase, err)
	}
	return nil
}
