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
	"encoding/json"
	"fmt"
	"os/exec"
)

// checkBillingEnabled checks if there's a billing account associated to the
// GCP project ID
func checkBillingEnabled(projectID string) (bool, error) {
	var o bytes.Buffer
	var e bytes.Buffer
	cmd := exec.Command("gcloud", "beta", "billing", "projects", "describe", "-q", "--format=json", projectID)
	cmd.Stdout = &o
	cmd.Stderr = &e
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("error determining if billing account is linked: %+v. output=\n%s", err, e.String())
	}
	v := struct {
		BillingEnabled bool `json:"billingEnabled"`
	}{}
	if err := json.NewDecoder(&o).Decode(&v); err != nil {
		return false, fmt.Errorf("error decoding gcloud --format=json output: %+v", err)
	}
	return v.BillingEnabled, nil
}
