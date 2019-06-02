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

	"github.com/fatih/color"

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
	// customize survey visuals ideally these shouldn't be global
	// see https://github.com/AlecAivazis/survey/issues/192
	// TODO(ahmetb): if the issue above is fixed, make the settings per-question
	defer func(s string) {
		surveycore.QuestionIcon = s
	}(surveycore.QuestionIcon)
	surveycore.QuestionIcon = questionPrefix

	var ok bool
	projectLabel := color.New(color.Bold, color.FgHiCyan).Sprint(project)
	if err := survey.AskOne(&survey.Confirm{
		Default: true,
		Message: fmt.Sprintf("Would you like to use existing GCP project %v to deploy this app?", projectLabel),
	}, &ok, nil); err != nil {
		return false, fmt.Errorf("could not prompt for confirmation using project %s: %+v", project, err)
	}
	return ok, nil
}

func promptMultipleProjects(projects []string) (string, error) {
	// customize survey visuals ideally these shouldn't be global
	// see https://github.com/AlecAivazis/survey/issues/192
	// TODO(ahmetb): if the issue above is fixed, make the settings per-question
	defer func(s string) {
		surveycore.QuestionIcon = s
	}(surveycore.QuestionIcon)
	defer func(s string) {
		surveycore.ErrorIcon = s
	}(surveycore.ErrorIcon)
	defer func(s string) {
		surveycore.SelectFocusIcon = s
	}(surveycore.SelectFocusIcon)
	surveycore.QuestionIcon = questionPrefix
	surveycore.ErrorIcon = errorPrefix
	surveycore.SelectFocusIcon = questionSelectFocusIcon

	var p string
	if err := survey.AskOne(&survey.Select{
		Message: "Choose a project to deploy this application:",
		Options: projects,
	}, &p, survey.Required); err != nil {
		return p, fmt.Errorf("could not choose a project: %+v", err)
	}
	return p, nil
}
