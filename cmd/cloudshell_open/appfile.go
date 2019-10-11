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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"crypto/rand"

	"github.com/fatih/color"

	"github.com/AlecAivazis/survey/v2"
)

type env struct {
	Description string `json:"description"`
	Value       string `json:"value"`
	Required    *bool  `json:"required"`
	Generator   string `json:"generator"`
}

type options struct {
	AllowUnauthenticated *bool `json:"allow-unauthenticated"`
}

type scripts struct {
	PostDeploy string `json:"postdeploy"`
}

type appFile struct {
	Name        string         `json:"name"`
	Env         map[string]env `json:"env"`
	Scripts 	scripts        `json:"scripts"`
	Options     options        `json:"options"`

	// The following are unused variables that are still silently accepted
	// for compatibility with Heroku app.json files.
	IgnoredDescription string   `json:"description"`
	IgnoredKeywords    []string `json:"keywords"`
	IgnoredLogo        string   `json:"logo"`
	IgnoredRepository  string   `json:"repository"`
	IgnoredWebsite     string   `json:"website"`
}

const appJSON = `app.json`

// hasAppFile checks if the directory has an app.json file.
func hasAppFile(dir string) (bool, error) {
	path := filepath.Join(dir, appJSON)
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return fi.Mode().IsRegular(), nil
}

func parseAppFile(r io.Reader) (*appFile, error) {
	var v appFile
	d := json.NewDecoder(r)
	d.DisallowUnknownFields()
	if err := d.Decode(&v); err != nil {
		return nil, fmt.Errorf("failed to parse app.json: %+v", err)
	}

	// make "required" true by default
	for k, env := range v.Env {
		if env.Required == nil {
			v := true
			env.Required = &v
		}
		v.Env[k] = env
	}

	for k, env := range v.Env {
		if env.Generator == "secret" && env.Value != "" {
			return nil, fmt.Errorf("env var %q can't have both a value and use the secret generator", k)
		}
	}

	return &v, nil
}

// getAppFile returns the parsed app.json in the directory if it exists,
// otherwise returns a zero appFile.
func getAppFile(dir string) (appFile, error) {
	var v appFile
	ok, err := hasAppFile(dir)
	if err != nil {
		return v, err
	}
	if !ok {
		return v, nil
	}
	f, err := os.Open(filepath.Join(dir, appJSON))
	if err != nil {
		return v, fmt.Errorf("error opening app.json file: %v", err)
	}
	defer f.Close()
	af, err := parseAppFile(f)
	if err != nil {
		return v, fmt.Errorf("failed to parse app.json file: %v", err)
	}
	return *af, nil
}

func rand64String() (string, error) {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// takes the envs defined in app.json, and the existing envs and returns the new envs that need to be prompted for
func needEnvs(list map[string]env, existing map[string]struct{}) map[string]env {
	for k := range list {
		_, isPresent := existing[k]
		if isPresent {
			delete(list, k)
		}
	}

	return list
}

func promptOrGenerateEnvs(list map[string]env) ([]string, error) {
	var toGenerate []string
	var toPrompt = make(map[string]env)

	for k, e := range list {
		if e.Generator == "secret" {
			toGenerate = append(toGenerate, k)
		} else {
			toPrompt[k] = e
		}
	}

	generated, err := generateEnvs(toGenerate)
	if err != nil {
		return nil, err
	}

	prompted, err := promptEnv(toPrompt)
	if err != nil {
		return nil, err
	}

	return append(generated, prompted...), nil
}

func generateEnvs(keys []string) ([]string, error) {

	for i, key := range keys {
		resp, err := rand64String()
		if err != nil {
			return nil, fmt.Errorf("failed to generate secret for %s : %v", key, err)
		}
		keys[i] = key+"="+resp
	}

	return keys, nil
}

func promptEnv(list map[string]env) ([]string, error) {
	// TODO(ahmetb): remove these defers and make customizations at the
	// individual prompt-level once survey lib allows non-global settings.

	var out []string
	// TODO(ahmetb): we should ideally use an ordered map structure for Env
	// field and prompt the questions as they appear in the app.json file as
	// opposed to random order we do here.
	for k, e := range list {
		var resp string

		if err := survey.AskOne(&survey.Input{
			Message: fmt.Sprintf("Value of %s environment variable (%s)",
				color.CyanString(k),
				color.HiBlackString(e.Description)),
			Default: e.Value,
		}, &resp,
			survey.WithValidator(survey.Required),
			surveyIconOpts,
		); err != nil {
			return nil, fmt.Errorf("failed to get a response for environment variable %s", k)
		}
		out = append(out, k+"="+resp)
	}

	return out, nil
}
