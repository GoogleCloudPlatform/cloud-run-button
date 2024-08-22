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

	artifactregistrypb "cloud.google.com/go/artifactregistry/apiv1/artifactregistrypb"
	"google.golang.org/api/artifactregistry/v1"
)

// Create a "Cloud Run Source Deploy" repository in Artifact Registry (if it doesn't already exist)
func createArtifactRegistry(project string, region string, repoName string) error {
	client, err := artifactregistry.NewClient(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to create artifact registry client: %w", err)
	}

	//TODO(glasnt): check registry already exists before trying to create it.

	req := &artifactregistrypb.CreateRepositoryRequest{
		parent:       fmt.Sprintf("projects/%s/locations/%s", project, region),
		RepositoryId: repoName,
		Repository:   "docker",
	}

	op, err := client.CreateRepository(context.TODO(), req)
	if err != nil {
		// TODO: Handle error.
	}
	return nil
}
