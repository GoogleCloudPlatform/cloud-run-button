package main

import (
	"fmt"
	"os/exec"
)

func build(dir, image string) error {
	cmd := exec.Command("docker", "build", "--quiet", "--tag", image, dir)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker build failed: %v, output:\n%s", err, string(b))
	}
	return nil
}

func push(image string) error {
	cmd := exec.Command("docker", "push", image)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker push failed: %v, output:\n%s", err, string(b))
	}
	return nil
}
