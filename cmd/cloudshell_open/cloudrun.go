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
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/AlecAivazis/survey/v2"
	runapi "google.golang.org/api/run/v1alpha1"
)

const (
	defaultRunRegion = "us-central1"
	defaultRunMemory = "512Mi"
)

type service struct {
	Spec serviceSpec `json:"spec"`
	Status serviceStatus `json:"status"`
}

type serviceSpec struct {
	Template serviceTemplate `json:"template"`
}

type serviceTemplate struct {
	Spec templateSpec `json:"spec"`
}

type templateSpec struct {
	Containers []container `json:"containers"`
}

type container struct {
	EnvVars []envVar `json:"env"`
}

type envVar struct {
	Name string `json:"name"`
	Value string `json:"value"`
}

type serviceStatus struct {
	Address address `json:"address"`
}

type address struct {
	Url string `json:"url"`
}

func optionsToFlags(options options) []string {
	authSetting := "--allow-unauthenticated"
	if options.AllowUnauthenticated != nil && *options.AllowUnauthenticated == false {
		authSetting = "--no-allow-unauthenticated"
	}

	return []string{authSetting}
}

func deploy(project, name, image, region string, envs []string, options options) (string, error) {
	optionsFlags := optionsToFlags(options)

	params := []string{
		"beta", "run", "deploy", "-q",
		name,
		"--project", project,
		"--platform", "managed",
		"--image", image,
		"--region", region,
		"--memory", defaultRunMemory,
		"--update-env-vars", strings.Join(envs, ","),
	}

	cmd := exec.Command("gcloud", append(params, optionsFlags...)...)

	if b, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to deploy to Cloud Run: %+v. output:\n%s", err, string(b))
	}
	return serviceURL(project, name, region)
}

func projectRunLocations(ctx context.Context, project string) ([]string, error) {
	runSvc, err := runapi.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Run API client: %+v", err)
	}

	var locations []string
	if err := runapi.NewProjectsLocationsService(runSvc).
		List("projects/"+project).Pages(ctx, func(resp *runapi.ListLocationsResponse) error {
		for _, v := range resp.Locations {
			locations = append(locations, v.LocationId)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("request to query Cloud Run locations failed: %+v", err)
	}
	sort.Strings(locations)
	return locations, nil
}

func promptDeploymentRegion(ctx context.Context, project string) (string, error) {
	locations, err := projectRunLocations(ctx, project)
	if err != nil {
		return "", fmt.Errorf("cannot retrieve Cloud Run locations: %+v", err)
	}

	var choice string
	if err := survey.AskOne(&survey.Select{
		Message: "Choose a region to deploy this application:",
		Options: locations,
		Default: defaultRunRegion,
	}, &choice,
		surveyIconOpts,
		survey.WithValidator(survey.Required),
	); err != nil {
		return choice, fmt.Errorf("could not choose a region: %+v", err)
	}
	return choice, nil
}


func describe(project, name, region string) (service, error) {
	var service service
	var o bytes.Buffer
	var e bytes.Buffer

	cmd := exec.Command("gcloud", "run", "services", "describe", name,
		"--project", project,
		"--platform", "managed",
		"--region", region,
		"--format=json")

	cmd.Stdout = &o
	cmd.Stderr = &e
	if err := cmd.Run(); err != nil {
		return service, fmt.Errorf("error describing service: %+v. output=\n%s", err, e.String())
	}

	if err := json.NewDecoder(&o).Decode(&service); err != nil {
		return service, fmt.Errorf("error decoding gcloud --format=json output: %+v", err)
	}

	return service, nil
}

func serviceURL(project, name, region string) (string, error) {
	service, err := describe(project, name, region)

	if err != nil {
		return "", err
	}

	return service.Status.Address.Url, nil
}

func envVars(project, name, region string) (map[string]struct{}, error) {
	service, err := describe(project, name, region)

	if err != nil {
		return nil, err
	}

	existing := map[string]struct{}{}

	for _, container := range service.Spec.Template.Spec.Containers {
		for _, envVar := range container.EnvVars {
			existing[envVar.Name] = struct{}{}
		}
	}

	return existing, nil
}

// tryFixServiceName attempts replace the service name with a better one to
// prevent deployment failures due to Cloud Run service naming constraints such
// as:
//
//   * names with a leading non-letter (e.g. digit or '-') are prefixed
//   * names over 63 characters are truncated
//   * names ending with a '-' have the suffix trimmed
func tryFixServiceName(name string) string {
	if name == "" {
		return name
	}

	name = strings.ToLower(name)

	reg := regexp.MustCompile("[^a-z0-9-]+")

	name = reg.ReplaceAllString(name, "-")

	if name[0] == '-' {
		name = fmt.Sprintf("svc%s", name)
	}

	if !unicode.IsLetter([]rune(name)[0]) {
		name = fmt.Sprintf("svc-%s", name)
	}

	if len(name) > 63 {
		name = name[:63]
	}

	for name[len(name)-1] == '-' {
		name = name[:len(name)-1]
	}

	return name
}
