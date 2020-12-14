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
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/iancoleman/orderedmap"

	"github.com/AlecAivazis/survey/v2"
)

type environment struct {
	Variables []env
}

type env struct {
	Name        string `json:"-"`
	Description string `json:"description"`
	Value       string `json:"value"`
	Required    bool   `json:"required"`
	Generator   string `json:"generator"`
}

type options struct {
	AllowUnauthenticated *bool  `json:"allow-unauthenticated"`
	Memory               string `json:"memory"`
	CPU                  string `json:"cpu"`
	Port                 int    `json:"port"`
}

type hook struct {
	Commands []string `json:"commands"`
}

type buildpacks struct {
	Builder string `json:"builder"`
}

type build struct {
	Skip       *bool      `json:"skip"`
	Buildpacks buildpacks `json:"buildpacks"`
}

type hooks struct {
	PreCreate  hook `json:"precreate"`
	PostCreate hook `json:"postcreate"`
	PreBuild   hook `json:"prebuild"`
	PostBuild  hook `json:"postbuild"`
}

type appFile struct {
	Name    string      `json:"name"`
	Env     environment `json:"env"`
	Options options     `json:"options"`
	Build   build       `json:"build"`
	Hooks   hooks       `json:"hooks"`

	// The following are unused variables that are still silently accepted
	// for compatibility with Heroku app.json files.
	IgnoredDescription string      `json:"description"`
	IgnoredKeywords    []string    `json:"keywords"`
	IgnoredLogo        string      `json:"logo"`
	IgnoredRepository  string      `json:"repository"`
	IgnoredWebsite     string      `json:"website"`
	IgnoredStack       string      `json:"stack"`
	IgnoredFormation   interface{} `json:"formation"`
}

const appJSON = `app.json`

// Custom, order-preserving unmarshalling of the environment entries
func (e *environment) UnmarshalJSON(b []byte) error {
	envVars := orderedmap.New()

	if err := json.Unmarshal(b, &envVars); err != nil {
		return fmt.Errorf("failed to parse the 'env' of app.json: %+v", err)
	}

	keys := envVars.Keys()
	for _, envVarName := range keys {
		envVarValue, _ := envVars.Get(envVarName)

		envVarMap, ok := envVarValue.(orderedmap.OrderedMap)
		if !ok {
			return fmt.Errorf("failed to parse the 'env' of app.json: expected a JSON object for the key '%s', found a string; value: '%s'", envVarName, envVarValue)
		}

		envVar := env{
			Name:     envVarName,
			Required: true,
		}

		for _, envVarFieldName := range envVarMap.Keys() {
			envVarValue, _ := envVarMap.Get(envVarFieldName)

			switch envVarFieldName {
			case "description":
				envVar.Description, ok = envVarValue.(string)
				if !ok {
					return fmt.Errorf("failed to parse the '%s' env of app.json: expected JSON string; found: '%T'", envVarName, envVarValue)
				}
			case "value":
				envVar.Value, ok = envVarValue.(string)
				if !ok {
					return fmt.Errorf("failed to parse the '%s' env of app.json: expected JSON string; found: '%T'", envVarName, envVarValue)
				}
			case "required":
				envVar.Required, ok = envVarValue.(bool)
				if !ok {
					return fmt.Errorf("failed to parse the '%s' env of app.json: expected JSON boolean; found: '%T'", envVarName, envVarValue)
				}
			case "generator":
				envVar.Generator, ok = envVarValue.(string)
				if !ok {
					return fmt.Errorf("failed to parse the '%s' env of app.json: expected JSON string; found: '%T'", envVarName, envVarValue)
				}
			default:
				return fmt.Errorf("Unexpected key '%s'", envVarFieldName)
			}
		}

		if envVar.Generator == "secret" && envVar.Value != "" {
			return fmt.Errorf("env var %q can't have both a value and use the secret generator", envVarName)
		}

		e.Variables = append(e.Variables, envVar)
	}

	return nil
}

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
func needEnvs(list []env, existing map[string]struct{}) []env {
	var neededEnvs []env

	for _, e := range list {
		_, isPresent := existing[e.Name]
		if !isPresent {
			neededEnvs = append(neededEnvs, e)
		}
	}

	return neededEnvs
}

func promptOrGenerateEnvs(list []env) ([]string, error) {
	var toGenerate []string
	var toPrompt []env

	for _, e := range list {
		k := e.Name
		if e.Generator == "secret" {
			toGenerate = append(toGenerate, k)
		} else {
			toPrompt = append(toPrompt, e)
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
		keys[i] = key + "=" + resp
	}

	return keys, nil
}

func promptEnv(list []env) ([]string, error) {
	// TODO(ahmetb): remove these defers and make customizations at the
	// individual prompt-level once survey lib allows non-global settings.

	var out []string
	// TODO(ahmetb): we should ideally use an ordered map structure for Env
	// field and prompt the questions as they appear in the app.json file as
	// opposed to random order we do here.
	for _, e := range list {
		k := e.Name
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
