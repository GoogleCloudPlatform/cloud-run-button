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
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
)

var (
	surveyIconOpts = survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question = survey.Icon{Text: questionPrefix}
		icons.Error = survey.Icon{Text: errorPrefix}
		icons.SelectFocus = survey.Icon{Text: questionSelectFocusIcon}
		icons.HelpInput = survey.Icon{Text: "Arrows to navigate"}
	})
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
	if len(projects) == 0 {
		return "", errors.New("cannot prompt with an empty list of projects")
	} else if len(projects) == 1 {
		ok, err := confirmProject(projects[0])
		if err != nil {
			return "", err
		} else if !ok {
			return "", fmt.Errorf("not allowed to use project %s", projects[0])
		}
		return projects[0], nil
	}
	return promptMultipleProjects(projects)
}

func confirmProject(project string) (bool, error) {
	var ok bool
	projectLabel := color.New(color.Bold, color.FgHiCyan).Sprint(project)
	if err := survey.AskOne(&survey.Confirm{
		Default: true,
		Message: fmt.Sprintf("Would you like to use existing GCP project %v to deploy this app?", projectLabel),
	}, &ok, surveyIconOpts); err != nil {
		return false, fmt.Errorf("could not prompt for confirmation using project %s: %+v", project, err)
	}
	return ok, nil
}

func promptMultipleProjects(projects []string) (string, error) {
	var p string
	if err := survey.AskOne(&survey.Select{
		Message: "Choose a project to deploy this application:",
		Options: projects,
	}, &p,
		surveyIconOpts,
		survey.WithValidator(survey.Required),
	); err != nil {
		return p, fmt.Errorf("could not choose a project: %+v", err)
	}
	return p, nil
}
