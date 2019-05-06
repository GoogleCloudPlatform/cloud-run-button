package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func deploy(project, name, image, region string) (string, error) {
	cmd := exec.Command("gcloud", "beta", "run", "deploy", "-q",
		name,
		"--project", project,
		"--image", image,
		"--region", region,
		"--allow-unauthenticated")
	if b, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to deploy to Cloud Run: %+v. output:\n%s", err, string(b))
	}
	return serviceURL(project, name, region)
}

func serviceURL(project, name, region string) (string, error) {
	cmd := exec.Command("gcloud", "beta", "run", "services", "describe", name,
		"--project", project,
		"--region", region,
		"--format", "value(status.domain)")

	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("deployment to Cloud Run failed: %+v. output:\n%s", err, string(b))
	}
	return strings.TrimSpace(string(b)), nil
}
