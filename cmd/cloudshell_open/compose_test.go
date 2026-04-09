package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestComposeFileExists(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected bool
	}{
		{
			name:     "no files",
			files:    []string{},
			expected: false,
		},
		{
			name:     "compose.yaml exists",
			files:    []string{"compose.yaml"},
			expected: true,
		},
		{
			name:     "compose.yml exists",
			files:    []string{"compose.yml"},
			expected: true,
		},
		{
			name:     "docker-compose.yaml exists",
			files:    []string{"docker-compose.yaml"},
			expected: true,
		},
		{
			name:     "docker-compose.yml exists",
			files:    []string{"docker-compose.yml"},
			expected: true,
		},
		{
			name:     "irrelevant files",
			files:    []string{"Dockerfile", "app.json", "README.md"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir(os.TempDir(), "compose-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			for _, f := range tt.files {
				if err := ioutil.WriteFile(filepath.Join(tmpDir, f), []byte("version: '3'"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			got, err := composeFileExists(tmpDir)
			if err != nil {
				t.Errorf("composeFileExists() error = %v", err)
				return
			}
			if got != tt.expected {
				t.Errorf("composeFileExists() got = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestComposeFileExists_IsDirectory(t *testing.T) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "compose-test-dir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// If compose.yaml is a directory instead of a file
	if err := os.Mkdir(filepath.Join(tmpDir, "compose.yaml"), 0755); err != nil {
		t.Fatal(err)
	}

	got, err := composeFileExists(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	// os.Stat returns no error for a directory, so currently composeFileExists would return true.
	// This matches the behavior of dockerFileExists in docker.go which also just uses os.Stat.
	if !got {
		t.Errorf("composeFileExists() expected to return true even if it's a directory (matching existing code patterns)")
	}
}
