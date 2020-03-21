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
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/AlecAivazis/survey/v2"
	"google.golang.org/api/option"
	runapi "google.golang.org/api/run/v1"
)

const (
	defaultRunRegion = "us-central1"
	defaultRunMemory = "512Mi"
)

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

func getService(project, name, region string) (*runapi.Service, error) {
	client, err := runClient(region)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Run API client: %w", err)
	}
	return client.Namespaces.Services.Get(fmt.Sprintf("namespaces/%s/services/%s", project, name)).Do()
}

func runClient(region string) (*runapi.APIService, error) {
	regionalEndpoint := fmt.Sprintf("https://%s-run.googleapis.com/", region)
	return runapi.NewService(context.TODO(), option.WithEndpoint(regionalEndpoint))
}

func serviceURL(project, name, region string) (string, error) {
	service, err := getService(project, name, region)
	if err != nil {
		return "", fmt.Errorf("failed to get Service: %w", err)
	}
	return service.Status.Address.Url, nil
}

func envVars(project, name, region string) (map[string]struct{}, error) {
	service, err := getService(project, name, region)

	if err != nil {
		return nil, err
	}

	existing := make(map[string]struct{})

	for _, container := range service.Spec.Template.Spec.Containers {
		for _, envVar := range container.Env {
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
