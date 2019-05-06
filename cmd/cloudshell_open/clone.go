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
	"regexp"
	"strings"
)

var (
	repoPattern = regexp.MustCompile(`^(git@|git://|https://)[a-zA-Z0-9/._:-]*$`)
)

func validRepoURL(repo string) bool { return repoPattern.MatchString(repo) }

func handleRepo(repo string) (string, error) {
	if !validRepoURL(repo) {
		return "", fmt.Errorf("invalid git repo url: %s", repo)
	}
	dir, err := repoDirName(repo)
	if err != nil {
		return "", err
	}
	return dir, clone(repo, dir)
}

func repoDirName(repo string) (string, error) {
	repo = strings.TrimSuffix(repo, ".git")
	i := strings.LastIndex(repo, "/")
	if i == -1 {
		return "", fmt.Errorf("cannot infer directory name from repo %s", repo)
	}
	dir := repo[i+1:]
	if dir == "" {
		return "", fmt.Errorf("cannot parse directory name from repo %s", repo)
	}
	if strings.HasPrefix(dir, ".") {
		return "", fmt.Errorf("attempt to clone into hidden directory: %s", dir)
	}
	return dir, nil
}

func clone(gitRepo, dir string) error {
	cmd := exec.Command("git", "clone", "--", gitRepo, dir)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %+v, output:\n%s", err, string(b))
	}
	return nil
}
