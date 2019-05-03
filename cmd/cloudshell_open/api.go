package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func enableAPIs(project string, apis []string) error {
	enabled, err := enabledAPIs(project)
	if err != nil {
		return err
	}

	var needAPIs []string
	for _, api := range apis {
		need := true
		for _, v := range enabled {
			if v == api {
				need = false
				break
			}
		}
		if need {
			needAPIs = append(needAPIs, api)
		}
	}
	if len(needAPIs) == 0 {
		return nil
	}

	cmd := exec.Command("gcloud", append([]string{"services", "enable", "--project", project, "-q"}, needAPIs...)...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable apis: %s", string(b))
	}
	return nil
}

func enabledAPIs(project string) ([]string, error) {
	cmd := exec.Command("gcloud", "services", "list", "--project", project, "--format", "value(config.name)")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled services on project %q. output: %s", project, string(b))
	}
	return strings.Split(strings.TrimSpace(string(b)), "\n"), nil
}
