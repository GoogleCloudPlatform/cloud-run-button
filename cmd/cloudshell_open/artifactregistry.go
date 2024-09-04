// Copyright 2024 Google LLC
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

	artifactregistry "cloud.google.com/go/artifactregistry/apiv1"
	artifactregistrypb "cloud.google.com/go/artifactregistry/apiv1/artifactregistrypb"
)

// Create a "Cloud Run Source Deploy" repository in Artifact Registry (if it doesn't already exist)
func createArtifactRegistry(project string, region string, repoName string) error {

	repoPrefix := fmt.Sprintf("projects/%s/locations/%s", project, region)
	repoFull := fmt.Sprintf("%s/repositories/%s", repoPrefix, repoName)

	ctx := context.Background()

	//Check for existing repo
	client, err := artifactregistry.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create artifact registry client: %w", err)
	}

	req := &artifactregistrypb.GetRepositoryRequest{
		Name: repoFull,
	}
	existingRepo, err := client.GetRepository(ctx, req)

	if err != nil {
		return fmt.Errorf("failed to retrieve existing artifact registry client: %w", err)
	}

	// Create repo if it doesn't already exist.
	if existingRepo != nil {
		req := &artifactregistrypb.CreateRepositoryRequest{
			Parent:       repoPrefix,
			RepositoryId: repoName,
			Repository: &artifactregistrypb.Repository{
				Name:   repoFull,
				Format: artifactregistrypb.Repository_DOCKER,
			},
		}

		_, err := client.CreateRepository(context.TODO(), req)
		if err != nil {
			return fmt.Errorf("failed to create artifact registry: %w", err)
		}
	}

	return nil

}
