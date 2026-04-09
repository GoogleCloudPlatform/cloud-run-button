package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func composeRunUp(appDir, region string) error {
	cmd := exec.Command("gcloud", "beta", "run", "compose", "up", "--region", region)
	cmd.Dir = appDir
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gcloud run compose up failed: %v, output:\n%s", err, string(b))
	}
	return nil
}

func composeFileExists(dir string) (bool, error) {
	files := []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			return true, nil
		}
	}
	return false, nil
}
