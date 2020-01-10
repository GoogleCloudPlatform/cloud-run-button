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

	"github.com/fatih/color"
)

var cmdColor = color.New(color.FgHiBlue)

func runScript(dir, command string, envs []string) error {
	fmt.Println(infoPrefix + "Running command:")
	cmdColor.Printf("\t%s\n", command)

	cmd := exec.Command("/bin/bash", "-c", "set -euo pipefail; set -x; " + command)
	cmd.Env = envs
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
