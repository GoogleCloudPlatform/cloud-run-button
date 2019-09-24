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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var (
	// used as convenience to take their reference in tests
	tru  = true
	fals = false
)

func TestHasAppFile(t *testing.T) {
	// non existing dir
	ok, err := hasAppFile(filepath.Join(os.TempDir(), "non-existing-dir"))
	if err != nil {
		t.Fatalf("got err for non-existing dir: %v", err)
	} else if ok {
		t.Fatal("returned true for non-existing dir")
	}

	tmpDir, err := ioutil.TempDir(os.TempDir(), "app.json-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// no file
	ok, err = hasAppFile(tmpDir)
	if err != nil {
		t.Fatalf("failed check for dir %s", tmpDir)
	} else if ok {
		t.Fatalf("returned false when app.json doesn't exist in dir %s", tmpDir)
	}

	// if app.json is a dir, must return false
	appJSON := filepath.Join(tmpDir, "app.json")
	if err := os.Mkdir(appJSON, 0755); err != nil {
		t.Fatal(err)
	}
	ok, err = hasAppFile(tmpDir)
	if err != nil {
		t.Fatal(err)
	} else if ok {
		t.Fatalf("reported true when app.json is a dir")
	}
	if err := os.RemoveAll(appJSON); err != nil {
		t.Fatal(err)
	}

	// write file
	if err := ioutil.WriteFile(appJSON, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	ok, err = hasAppFile(tmpDir)
	if err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Fatalf("reported false when app.json exists")
	}
}

func Test_parseAppFile(t *testing.T) {

	tests := []struct {
		name    string
		args    string
		want    *appFile
		wantErr bool
	}{
		{"empty json is EOF", ``, nil, true},
		{"empty object ok", `{}`, &appFile{RequireAuth: &fals}, false},
		{"bad object at root", `1`, nil, true},
		{"unknown field", `{"foo":"bar"}`, nil, true},
		{"require auth true", `{"require-auth": true}`, &appFile{RequireAuth: &tru}, false},
		{"wrong env type", `{"env": "foo"}`, nil, true},
		{"wrong env value type", `{"env": {"foo":"bar"}}`, nil, true},
		{"env not array", `{"env": []}`, nil, true},
		{"empty env list is ok", `{"env": {}}`, &appFile{RequireAuth: &fals, Env: map[string]env{}}, false},
		{"non-string key type in env", `{"env": {
			1: {}
		}}`, nil, true},
		{"unknown field in env", `{"env": {
			"KEY": {"unknown":"value"}
		}}`, nil, true},
		{"required is true by default", `{
			"env": {"KEY":{}}}`, &appFile{RequireAuth: &fals, Env: map[string]env{
			"KEY": env{Required: &tru}}}, false},
		{"required can be set to false", `{
			"env": {"KEY":{"required":false}}}`, &appFile{RequireAuth: &fals,
			Env: map[string]env{"KEY": env{Required: &fals}}}, false},
		{"required has to be bool", `{
			"env": {"KEY":{"required": "false"}}}`, nil, true},
		{"value has to be string", `{
			"env": {"KEY":{"value": 100}}}`, nil, true},
		{"parses ok", `{
			"name": "foo",
			"env": {
				"KEY_1": {
					"required": false,
					"description": "key 1 is cool"
				},
				"KEY_2": {
					"value": "k2"
				}
			}}`,
			&appFile{
				Name:        "foo",
				RequireAuth: &fals,
				Env: map[string]env{
					"KEY_1": env{
						Required:    &fals,
						Description: "key 1 is cool",
					},
					"KEY_2": env{
						Value:    "k2",
						Required: &tru,
					},
				}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAppFile(strings.NewReader(tt.args))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAppFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseAppFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseAppFile_parsesIgnoredKnownFields(t *testing.T) {
	appFile := `{
		"description": "String",
		"repository": "URL",
		"logo": "URL",
		"website": "URL",
		"keywords": ["String", "String"]
	}`
	_, err := parseAppFile(strings.NewReader(appFile))
	if err != nil {
		t.Fatalf("app.json with ignored but known fields failed: %v", err)
	}
}

func TestGetAppFile(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "app.json-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// non existing file must return zero
	v, err := getAppFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	var zero appFile
	if !reflect.DeepEqual(v, zero) {
		t.Fatalf("not zero value: got=%#v, expected=%#v", v, zero)
	}

	// captures parse error
	if err := ioutil.WriteFile(filepath.Join(dir, "app.json"), []byte(`
		{"env": {"KEY": 1 }}
	`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := getAppFile(dir); err == nil {
		t.Fatal("was expected to fail with invalid json")
	}

	// parse valid file
	if err := ioutil.WriteFile(filepath.Join(dir, "app.json"), []byte(`
		{"env": {"KEY": {"value":"bar"} }}
	`), 0644); err != nil {
		t.Fatal(err)
	}

	v, err = getAppFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	expected := appFile{Env: map[string]env{
		"KEY": env{Value: "bar", Required: &tru},
	}}
	if !reflect.DeepEqual(v, expected) {
		t.Fatalf("wrong parsed value: got=%#v, expected=%#v", v, expected)
	}
}
