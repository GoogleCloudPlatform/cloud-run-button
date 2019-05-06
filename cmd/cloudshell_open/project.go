package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"gopkg.in/AlecAivazis/survey.v1"
	surveycore "gopkg.in/AlecAivazis/survey.v1/core"
)

func listProjects() ([]string, error) {
	cmd := exec.Command("gcloud", "projects", "list", "--format", "value(projectId)")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %+v, output:\n%s", err, string(b))
	}
	b = bytes.TrimSpace(b)
	p := strings.Split(string(b), "\n")
	sort.Strings(p)
	return p, err
}

func promptProject(projects []string) (string, error) {
	var p string

	// customize survey visuals ideally these shouldn't be global
	// see https://github.com/AlecAivazis/survey/issues/192
	surveycore.QuestionIcon = questionPrefix
	surveycore.ErrorIcon = errorPrefix
	surveycore.SelectFocusIcon = questionSelectFocusIcon

	if err := survey.AskOne(&survey.Select{
		Message: "Choose a GCP project to deploy:",
		Options: projects,
	}, &p, nil); err != nil {
		return p, fmt.Errorf("could not choose a project: %+v", err)
	}
	return p, nil
}
