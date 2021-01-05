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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestValidRepoURL(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"http://should-not-be-http", false},
		{"https://github.com/user/bar", true},
		{"https://github.com/user/bar.git", true},
	}
	for _, tt := range tests {
		if got := validRepoURL(tt.in); got != tt.want {
			t.Fatalf("validRepoURL(%s) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestRepoDirName(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"foo-bar", "", true}, // cannot infer repo name after '/'
		{"/bar", "bar", false},
		{"https://github.com/foo/bar", "bar", false},
	}
	for _, tt := range tests {
		got, err := repoDirName(tt.in)
		if (err != nil) != tt.wantErr {
			t.Fatalf("repoDirName(%s) error = %v, wantErr %v (got=%s)", tt.in, err, tt.wantErr, got)
		} else if got != tt.want {
			t.Fatalf("repoDirName(%s) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestClone(t *testing.T) {
	tests := []struct {
		name    string
		gitRepo string
		wantErr bool
	}{
		{"404", "http://example.com/git/repo", true},
		{"https", "https://github.com/google/new-project", false},
		{"https+.git", "https://github.com/google/new-project.git", false},
	}
	testDir, err := ioutil.TempDir(os.TempDir(), "git-clone-test")
	if err != nil {
		t.Fatal(err)
	}
	for i, tt := range tests {
		t.Run(tt.name, func(ts *testing.T) {
			cloneDir := filepath.Join(testDir, fmt.Sprintf("test-%d", i))
			err := clone(tt.gitRepo, cloneDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("clone(%s) error = %v, wantErr %v", tt.gitRepo, err, tt.wantErr)
				return
			}
		})
	}
}

func TestGitCheckout(t *testing.T) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "checkout-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	run := func(tt *testing.T, cmd string, args ...string) {
		tt.Helper()
		c := exec.Command(cmd, args...)
		c.Dir = tmpDir
		if b, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%s %v failed: %+v\n%s", cmd, args, err, string(b))
		}
	}

	run(t, "git", "init", ".")
	run(t, "git", "commit", "--allow-empty", "--message", "initial commit")
	run(t, "git", "branch", "foo")

	if err := gitCheckout(tmpDir, "main"); err != nil {
		t.Fatal(err)
	}
	if err := gitCheckout(tmpDir, "foo"); err != nil {
		t.Fatal(err)
	}

}
