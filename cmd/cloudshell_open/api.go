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

	"google.golang.org/api/serviceusage/v1"
)

func enableAPIs(project string, apis []string) error {
	client, err := serviceusage.NewService(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to create resource manager client: %w", err)
	}

	// TODO(ahmetb) specify this explicitly, otherwise for some reason this becomes serviceusage.mtls.googleapis.com (and 404s)
	// while querying the Operation. investigate later with the client library teams.
	client.BasePath = "https://serviceusage.googleapis.com/"

	enabled, err := enabledAPIs(client, project)
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

	op, err := client.Services.BatchEnable("projects/"+project, &serviceusage.BatchEnableServicesRequest{
		ServiceIds: needAPIs,
	}).Do()
	if err != nil {
		return fmt.Errorf("failed to issue enable APIs request: %w", err)
	}

	opID := op.Name
	for !op.Done {
		op, err = client.Operations.Get(opID).Context(context.TODO()).Do()
		if err != nil {
			return fmt.Errorf("failed to query operation status (%s): %w", opID, err)
		}
		if op.Error != nil {
			return fmt.Errorf("enabling APIs failed (operation=%s, code=%d): %s", op.Name, op.Error.Code, op.Error.Message)
		}
	}
	return nil
}

func enabledAPIs(client *serviceusage.Service, project string) ([]string, error) {
	var out []string
	if err := client.Services.List("projects/"+project).PageSize(200).Pages(context.TODO(),
		func(resp *serviceusage.ListServicesResponse) error {
			for _, p := range resp.Services {
				if p.State == "ENABLED" {
					out = append(out, p.Config.Name)
				}
			}
			return nil
		}); err != nil {
		return nil, fmt.Errorf("failed to list APIs on the project: %w", err)
	}
	return out, nil
}
