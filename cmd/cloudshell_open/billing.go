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

	"google.golang.org/api/cloudbilling/v1"
)

// checkBillingEnabled checks if there's a billing account associated to the
// GCP project ID.
func checkBillingEnabled(projectID string) (bool, error) {
	client, err := cloudbilling.NewService(context.TODO())
	if err != nil {
		return false, fmt.Errorf("failed to initialize cloud billing client: %w", err)
	}
	bo, err := client.Projects.GetBillingInfo("projects/" + projectID).Context(context.TODO()).Do()
	if err != nil {
		return false, fmt.Errorf("failed to query project billing info: %w", err)
	}
	return bo.BillingEnabled, nil
}

func billingAccounts() ([]cloudbilling.BillingAccount, error) {
	var out []cloudbilling.BillingAccount

	client, err := cloudbilling.NewService(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloud billing client: %w", err)
	}
	billingAccounts, err := client.BillingAccounts.List().Context(context.TODO()).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to query billing accounts: %w", err)
	}

	for _, p := range billingAccounts.BillingAccounts {
		if p.Open {
			out = append(out, *p)
		}
	}

	return out, nil
}
