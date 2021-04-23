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

package instrumentless

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Coupon struct {
	URL string `json:"url"`
}

func GetCoupon(event string, bearerToken string) (*Coupon, error) {
	url := fmt.Sprintf("https://api.gcpcredits.com/%s", event)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New(res.Status)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	coupon := Coupon{}
	err = json.NewDecoder(res.Body).Decode(&coupon)
	if err != nil {
		return nil, fmt.Errorf("failed to read and parse the response: %v", err)
	}

	return &coupon, nil
}
