package main

import (
	"fmt"
	"io/ioutil"
	"os"
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
		{"git@invalid characters", false},
		{"git@github.com/user/bar?invalid=chars", false},
		{"git@github.com/user/bar.git", true},
		{"https://github.com/user/bar", true},
		{"https://github.com/user/bar.git", true},
		{"git://github.com/user/bar", true},
		{"git://github.com/user/bar.git", true},
		{" git://github.com/user/bar.git", false},
		{"git://github.com/user/bar.git ", false},
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
		{"git://github.com/user/foo/", "", true},  // base name empty
		{"git://github.com/user/foo//", "", true}, // base name empty
		{"https://github.com/foo/bar", "bar", false},
		{"git://github.com/user/bar.git", "bar", false},
		{"git://github.com/user/.bar.git", "", true}, // dir starts with dot
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
		{"git@", "git@github.com:google/new-project.git", false},
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
